// Package oauthgrants_test runs the OAuth flow and its retention against a
// real Postgres. It needs TEST_DATABASE_DSN.
package oauthgrants_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/synclet-io/synclet/modules/auth/authdbstate"
	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/modules/auth/authstorage"
	"github.com/synclet-io/synclet/tests/testdb"
)

const (
	verifier    = "dBjftJeZ4CVP-mJ92ZEUt3NqSZSAagNZZcWhLk7vH8u2NwkCFX" //nolint:gosec // test PKCE verifier
	redirectURI = "http://127.0.0.1:33418/callback"
)

type noAudit struct{}

func (noAudit) Record(context.Context, authservice.AuditEvent) {}

// editorOf makes every user an editor of one workspace.
type editorOf uuid.UUID

func (w editorOf) ListForUser(context.Context, uuid.UUID) ([]authservice.WorkspaceMembership, error) {
	return []authservice.WorkspaceMembership{{WorkspaceID: uuid.UUID(w), Role: authservice.WorkspaceRoleEditor}}, nil
}

type flow struct {
	db       *gorm.DB
	storage  authservice.Storage
	oauth    *authstorage.OAuthStorage
	clientID string
	userID   uuid.UUID
	wsID     uuid.UUID
}

func setup(t *testing.T) *flow {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping OAuth database test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	testdb.Migrate(t, sqlDB, authdbstate.MigrationsFS, "auth_goose_db_version")

	ctx := context.Background()
	storage := authstorage.NewStorages(db, nil, nil)
	userID := uuid.New()

	_, err = storage.Users().Create(ctx, &authservice.User{ID: userID, Email: userID.String() + "@example.com", PasswordHash: "hash", CreatedAt: time.Now().UTC()})
	require.NoError(t, err)

	client, err := authservice.NewRegisterOAuthClient(storage).Execute(ctx, "Claude", []string{redirectURI})
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, storage.OAuthClients().Delete(ctx, &authservice.OAuthClientFilter{ClientID: filter.Equals(client.ClientID)}))
		require.NoError(t, storage.Users().Delete(ctx, &authservice.UserFilter{ID: filter.Equals(userID)}))
	})

	return &flow{db: db, storage: storage, oauth: authstorage.NewOAuthStorage(db), clientID: client.ClientID, userID: userID, wsID: uuid.New()}
}

func (f *flow) tokens(t *testing.T) *authservice.OAuthTokens {
	t.Helper()

	sum := sha256.Sum256([]byte(verifier))
	code, err := authservice.NewIssueOAuthCode(f.storage, editorOf(f.wsID)).Execute(context.Background(), authservice.IssueOAuthCodeParams{
		ClientID:            f.clientID,
		RedirectURI:         redirectURI,
		CodeChallenge:       base64.RawURLEncoding.EncodeToString(sum[:]),
		CodeChallengeMethod: "S256",
		UserID:              f.userID,
		WorkspaceID:         f.wsID,
		Level:               authservice.OAuthAccessLevelActions,
	})
	require.NoError(t, err)

	exchange := authservice.NewExchangeOAuthCode(f.storage, f.oauth, noAudit{})

	tokens, err := exchange.Execute(context.Background(), code, verifier, f.clientID, redirectURI)
	require.NoError(t, err)

	_, err = exchange.Execute(context.Background(), code, verifier, f.clientID, redirectURI)
	require.ErrorIs(t, err, authservice.ErrInvalidGrant, "a code is redeemed once")

	return tokens
}

func (f *flow) grantKeys(t *testing.T) []*authservice.APIKey {
	t.Helper()

	keys, err := f.storage.APIKeys().Find(context.Background(), &authservice.APIKeyFilter{UserID: filter.Equals(f.userID)})
	require.NoError(t, err)

	return keys
}

func TestOAuthFlow_RefreshAndReuse(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	refresh := authservice.NewRefreshOAuthToken(f.storage, f.oauth, editorOf(f.wsID))

	first := f.tokens(t)
	keys := f.grantKeys(t)
	require.Len(t, keys, 1)
	require.NotNil(t, keys[0].OAuthLevel)
	assert.Equal(t, authservice.OAuthAccessLevelActions, *keys[0].OAuthLevel)
	assert.Equal(t, "Claude", keys[0].Name)

	second, err := refresh.Execute(ctx, first.RefreshToken, f.clientID)
	require.NoError(t, err)

	keys = f.grantKeys(t)
	require.Len(t, keys, 1, "a refresh replaces the grant's key")
	assert.NotEqual(t, first.AccessToken, second.AccessToken)

	_, err = refresh.Execute(ctx, first.RefreshToken, f.clientID)
	require.ErrorIs(t, err, authservice.ErrInvalidGrant)

	assert.Empty(t, f.grantKeys(t), "reuse ends the grant")

	_, err = refresh.Execute(ctx, second.RefreshToken, f.clientID)
	require.ErrorIs(t, err, authservice.ErrInvalidGrant)
}

func TestCleanupOAuthGrants_DeletesAbandoned(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	live := f.tokens(t)
	f.tokens(t)

	grants, err := f.storage.OAuthGrants().Find(ctx, &authservice.OAuthGrantFilter{UserID: filter.Equals(f.userID)})
	require.NoError(t, err)
	require.Len(t, grants, 2)

	liveKey, err := authservice.NewValidateAPIKey(f.storage, nil).Execute(ctx, live.AccessToken)
	require.NoError(t, err)

	abandoned := grants[0].ID
	if abandoned == *liveKey.OAuthGrantID {
		abandoned = grants[1].ID
	}

	past := time.Now().Add(-time.Minute).UTC()
	require.NoError(t, f.db.Exec(`UPDATE auth.oauth_refresh_tokens SET expires_at = ? WHERE grant_id = ?`, past, abandoned).Error)
	require.NoError(t, f.db.Exec(`UPDATE auth.api_keys SET expires_at = ? WHERE oauth_grant_id = ?`, past, abandoned).Error)

	require.NoError(t, authservice.NewCleanupOAuthGrants(f.storage, f.oauth).Execute(ctx))

	grants, err = f.storage.OAuthGrants().Find(ctx, &authservice.OAuthGrantFilter{UserID: filter.Equals(f.userID)})
	require.NoError(t, err)
	require.Len(t, grants, 1)
	assert.Equal(t, *liveKey.OAuthGrantID, grants[0].ID)

	keys := f.grantKeys(t)
	require.Len(t, keys, 1)
	assert.Equal(t, liveKey.ID, keys[0].ID)

	_, err = authservice.NewRefreshOAuthToken(f.storage, f.oauth, editorOf(f.wsID)).Execute(ctx, live.RefreshToken, f.clientID)
	require.NoError(t, err)
}

func TestCleanupOAuthGrants_KeepsLatestKeyOfLiveGrant(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	tokens := f.tokens(t)
	keys := f.grantKeys(t)
	require.Len(t, keys, 1)

	require.NoError(t, f.db.Exec(`UPDATE auth.api_keys SET expires_at = ? WHERE id = ?`, time.Now().Add(-2*time.Hour).UTC(), keys[0].ID).Error)
	require.NoError(t, authservice.NewCleanupOAuthGrants(f.storage, f.oauth).Execute(ctx))

	left := f.grantKeys(t)
	require.Len(t, left, 1, "the grant stays listed and revocable through its latest key")
	assert.Equal(t, keys[0].ID, left[0].ID)

	require.NoError(t, authservice.NewRevokeAPIKey(f.storage, noAudit{}).Execute(ctx, keys[0].ID, f.userID))

	_, err := authservice.NewRefreshOAuthToken(f.storage, f.oauth, editorOf(f.wsID)).Execute(ctx, tokens.RefreshToken, f.clientID)
	require.ErrorIs(t, err, authservice.ErrInvalidGrant)
}

func TestCleanupOAuthGrants_DeletesUnusedClients(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	now := time.Now().UTC()

	f.tokens(t)
	require.NoError(t, f.db.Exec(`UPDATE auth.oauth_clients SET created_at = ?, last_granted_at = ? WHERE client_id = ?`,
		now.Add(-90*24*time.Hour), now.Add(-60*24*time.Hour), f.clientID).Error)

	client := func(created time.Time, lastGranted *time.Time) string {
		id := uuid.NewString()
		_, err := f.storage.OAuthClients().Create(ctx, &authservice.OAuthClient{
			ID: uuid.New(), ClientID: id, Name: "test", RedirectURIs: []string{redirectURI}, CreatedAt: created, LastGrantedAt: lastGranted,
		})
		require.NoError(t, err)

		t.Cleanup(func() {
			require.NoError(t, f.storage.OAuthClients().Delete(ctx, &authservice.OAuthClientFilter{ClientID: filter.Equals(id)}))
		})

		return id
	}
	freshNeverGranted := client(now.Add(-time.Hour), nil)
	oldNeverGranted := client(now.Add(-25*time.Hour), nil)
	goneRecently := client(now.Add(-90*24*time.Hour), new(now.Add(-24*time.Hour)))
	goneLongAgo := client(now.Add(-90*24*time.Hour), new(now.Add(-31*24*time.Hour)))

	require.NoError(t, authservice.NewCleanupOAuthGrants(f.storage, f.oauth).Execute(ctx))

	ids := []string{f.clientID, freshNeverGranted, oldNeverGranted, goneRecently, goneLongAgo}
	left, err := f.storage.OAuthClients().Find(ctx, &authservice.OAuthClientFilter{ClientID: filter.In(ids...)})
	require.NoError(t, err)

	leftIDs := make([]string, 0, len(left))
	for _, c := range left {
		leftIDs = append(leftIDs, c.ClientID)
	}

	assert.ElementsMatch(t, []string{f.clientID, freshNeverGranted, goneRecently}, leftIDs, "a client with a live grant stays however old")
}
