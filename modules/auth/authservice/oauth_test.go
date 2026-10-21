package authservice

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testVerifier    = "dBjftJeZ4CVP-mJ92ZEUt3NqSZSAagNZZcWhLk7vH8u2NwkCFX" //nolint:gosec // test PKCE verifier
	testRedirectURI = "https://claude.ai/api/mcp/auth_callback"
)

func challengeOf(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))

	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func TestRegisterOAuthClient(t *testing.T) {
	accepted := [][]string{
		{"https://claude.ai/cb"},
		{"http://localhost:3000/cb"},
		{"http://127.0.0.1/cb"},
		{"https://claude.ai/cb", "http://localhost:6274/oauth/callback"},
	}
	for _, uris := range accepted {
		t.Run("accepts "+strings.Join(uris, ","), func(t *testing.T) {
			store := newFakeStorage()

			client, err := NewRegisterOAuthClient(store).Execute(context.Background(), "Claude", uris)
			require.NoError(t, err)
			assert.NotEmpty(t, client.ClientID)
			assert.Equal(t, "Claude", client.Name)
			assert.Equal(t, uris, client.RedirectURIs)
			require.Len(t, store.oauthClients, 1)
			assert.Equal(t, client.ClientID, store.oauthClients[0].ClientID)
		})
	}

	rejected := [][]string{
		{"http://example.com/cb"},
		{"ftp://x"},
		nil,
		{"https://claude.ai/cb", "http://example.com/cb"},
		{"https://claude.ai/cb#frag"},
		{"https:///cb"},
		{"http://localhost.example.com/cb"},
	}
	for _, uris := range rejected {
		t.Run("rejects "+strings.Join(uris, ","), func(t *testing.T) {
			store := newFakeStorage()

			_, err := NewRegisterOAuthClient(store).Execute(context.Background(), "Claude", uris)

			var validationErr *ValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Empty(t, store.oauthClients)
		})
	}
}

type oauthFixture struct {
	store       *fakeStorage
	audit       *capturingAuditRecorder
	members     fakeMemberships
	client      *OAuthClient
	userID      uuid.UUID
	workspaceID uuid.UUID
}

// fakeMemberships maps a user to their workspace memberships.
type fakeMemberships map[uuid.UUID][]WorkspaceMembership

func (m fakeMemberships) ListForUser(_ context.Context, userID uuid.UUID) ([]WorkspaceMembership, error) {
	return m[userID], nil
}

func newOAuthFixture(t *testing.T) *oauthFixture {
	t.Helper()

	store := newFakeStorage()
	client, err := NewRegisterOAuthClient(store).Execute(context.Background(), "Claude", []string{testRedirectURI})
	require.NoError(t, err)

	userID, workspaceID := uuid.New(), uuid.New()
	members := fakeMemberships{userID: {{WorkspaceID: workspaceID, Role: WorkspaceRoleEditor}}}

	return &oauthFixture{store: store, audit: &capturingAuditRecorder{}, members: members, client: client, userID: userID, workspaceID: workspaceID}
}

func (f *oauthFixture) issueCode(t *testing.T, level OAuthAccessLevel) string {
	t.Helper()

	code, err := NewIssueOAuthCode(f.store, f.members).Execute(context.Background(), IssueOAuthCodeParams{
		ClientID:            f.client.ClientID,
		RedirectURI:         testRedirectURI,
		CodeChallenge:       challengeOf(testVerifier),
		CodeChallengeMethod: "S256",
		UserID:              f.userID,
		WorkspaceID:         f.workspaceID,
		Level:               level,
	})
	require.NoError(t, err)
	require.NotEmpty(t, code)

	return code
}

func (f *oauthFixture) exchange(code, verifier, clientID, redirectURI string) (*OAuthTokens, error) {
	return NewExchangeOAuthCode(f.store, f.store, f.audit).Execute(context.Background(), code, verifier, clientID, redirectURI)
}

func (f *oauthFixture) refresh(refreshToken string) (*OAuthTokens, error) {
	return NewRefreshOAuthToken(f.store, f.store, f.members).Execute(context.Background(), refreshToken, f.client.ClientID)
}

func (f *oauthFixture) tokens(t *testing.T) *OAuthTokens {
	t.Helper()

	tokens, err := f.exchange(f.issueCode(t, OAuthAccessLevelRead), testVerifier, f.client.ClientID, testRedirectURI)
	require.NoError(t, err)

	return tokens
}

func TestIssueOAuthCode_Rejects(t *testing.T) {
	f := newOAuthFixture(t)
	valid := IssueOAuthCodeParams{
		ClientID:            f.client.ClientID,
		RedirectURI:         testRedirectURI,
		CodeChallenge:       challengeOf(testVerifier),
		CodeChallengeMethod: "S256",
		UserID:              f.userID,
		WorkspaceID:         f.workspaceID,
		Level:               OAuthAccessLevelWrite,
	}

	cases := map[string]func(p *IssueOAuthCodeParams){
		"plain method":          func(p *IssueOAuthCodeParams) { p.CodeChallengeMethod = "plain" },
		"no challenge":          func(p *IssueOAuthCodeParams) { p.CodeChallenge = "" },
		"unregistered redirect": func(p *IssueOAuthCodeParams) { p.RedirectURI = "https://claude.ai/other" },
		"unknown client":        func(p *IssueOAuthCodeParams) { p.ClientID = "nope" },
		"no level":              func(p *IssueOAuthCodeParams) { p.Level = 0 },
		"not a member":          func(p *IssueOAuthCodeParams) { p.WorkspaceID = uuid.New() },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			params := valid
			mutate(&params)

			_, err := NewIssueOAuthCode(f.store, f.members).Execute(context.Background(), params)
			require.Error(t, err)
			assert.Empty(t, f.store.oauthCodes)
		})
	}
}

func TestOAuthExchange_IssuesKeyAndRefreshToken(t *testing.T) {
	f := newOAuthFixture(t)
	code := f.issueCode(t, OAuthAccessLevelActions)

	tokens, err := f.exchange(code, testVerifier, f.client.ClientID, testRedirectURI)
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(tokens.AccessToken, "synclet_sk_"))
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.Equal(t, time.Hour, tokens.ExpiresIn)

	key, err := NewValidateAPIKey(f.store, nil).Execute(context.Background(), tokens.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "Claude", key.Name)
	assert.Equal(t, f.userID, key.UserID)
	assert.Equal(t, f.workspaceID, key.WorkspaceID)
	require.NotNil(t, key.OAuthLevel)
	assert.Equal(t, OAuthAccessLevelActions, *key.OAuthLevel)
	require.NotNil(t, key.OAuthGrantID)
	require.NotNil(t, key.ExpiresAt)
	assert.WithinDuration(t, time.Now().Add(time.Hour), *key.ExpiresAt, time.Minute)

	require.Len(t, f.store.oauthGrants, 1)
	assert.Equal(t, *key.OAuthGrantID, f.store.oauthGrants[0].ID)
	require.Len(t, f.store.oauthRefreshTokens, 1)
	assert.NotEqual(t, tokens.RefreshToken, f.store.oauthRefreshTokens[0].TokenHash)
	assert.WithinDuration(t, time.Now().Add(30*24*time.Hour), f.store.oauthRefreshTokens[0].ExpiresAt, time.Minute)

	require.Len(t, f.audit.events, 1)
	assert.Equal(t, "api_key", f.audit.events[0].ResourceType)
	assert.Equal(t, "create", f.audit.events[0].Action)
}

func TestOAuthExchange_RejectsInvalidGrant(t *testing.T) {
	cases := map[string]func(f *oauthFixture, code string) (*OAuthTokens, error){
		"wrong verifier": func(f *oauthFixture, code string) (*OAuthTokens, error) {
			return f.exchange(code, testVerifier+"x", f.client.ClientID, testRedirectURI)
		},
		"wrong client": func(f *oauthFixture, code string) (*OAuthTokens, error) {
			other, err := NewRegisterOAuthClient(f.store).Execute(context.Background(), "Other", []string{testRedirectURI})
			if err != nil {
				return nil, err
			}

			return f.exchange(code, testVerifier, other.ClientID, testRedirectURI)
		},
		"wrong redirect": func(f *oauthFixture, code string) (*OAuthTokens, error) {
			return f.exchange(code, testVerifier, f.client.ClientID, "https://claude.ai/other")
		},
		"unknown code": func(f *oauthFixture, _ string) (*OAuthTokens, error) {
			return f.exchange("nope", testVerifier, f.client.ClientID, testRedirectURI)
		},
		"expired code": func(f *oauthFixture, code string) (*OAuthTokens, error) {
			f.store.oauthCodes[0].ExpiresAt = time.Now().Add(-time.Second)

			return f.exchange(code, testVerifier, f.client.ClientID, testRedirectURI)
		},
	}
	for name, run := range cases {
		t.Run(name, func(t *testing.T) {
			f := newOAuthFixture(t)
			code := f.issueCode(t, OAuthAccessLevelRead)

			_, err := run(f, code)
			require.ErrorIs(t, err, ErrInvalidGrant)
			assert.Empty(t, f.store.apiKeys)
			assert.Empty(t, f.store.oauthGrants)

			if name != "unknown code" {
				_, err = f.exchange(code, testVerifier, f.client.ClientID, testRedirectURI)
				require.ErrorIs(t, err, ErrInvalidGrant, "a code is spent by a failed exchange too")
			}
		})
	}
}

func TestOAuthExchange_CodeWorksOnce(t *testing.T) {
	f := newOAuthFixture(t)
	code := f.issueCode(t, OAuthAccessLevelRead)

	_, err := f.exchange(code, testVerifier, f.client.ClientID, testRedirectURI)
	require.NoError(t, err)

	_, err = f.exchange(code, testVerifier, f.client.ClientID, testRedirectURI)
	require.ErrorIs(t, err, ErrInvalidGrant)
	assert.Len(t, f.store.apiKeys, 1)
}

func TestOAuthRefresh_RotatesTokenAndKey(t *testing.T) {
	f := newOAuthFixture(t)
	first := f.tokens(t)

	second, err := f.refresh(first.RefreshToken)
	require.NoError(t, err)

	assert.NotEqual(t, first.RefreshToken, second.RefreshToken)
	assert.NotEqual(t, first.AccessToken, second.AccessToken)
	assert.Equal(t, time.Hour, second.ExpiresIn)

	validate := NewValidateAPIKey(f.store, nil)
	_, err = validate.Execute(context.Background(), first.AccessToken)
	require.ErrorIs(t, err, ErrInvalidAPIKey, "the previous key is deleted")

	key, err := validate.Execute(context.Background(), second.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, OAuthAccessLevelRead, *key.OAuthLevel)
	assert.Len(t, f.store.apiKeys, 1, "a grant shows as one key")

	third, err := f.refresh(second.RefreshToken)
	require.NoError(t, err)
	assert.NotEqual(t, second.RefreshToken, third.RefreshToken)
}

func TestOAuthRefresh_ReuseRevokesGrant(t *testing.T) {
	f := newOAuthFixture(t)
	first := f.tokens(t)

	second, err := f.refresh(first.RefreshToken)
	require.NoError(t, err)

	_, err = f.refresh(first.RefreshToken)
	require.ErrorIs(t, err, ErrInvalidGrant)

	assert.Empty(t, f.store.oauthGrants)
	assert.Empty(t, f.store.apiKeys)
	assert.Empty(t, f.store.oauthRefreshTokens)

	_, err = f.refresh(second.RefreshToken)
	require.ErrorIs(t, err, ErrInvalidGrant)
}

func TestOAuthRefresh_RejectsInvalid(t *testing.T) {
	t.Run("unknown", func(t *testing.T) {
		f := newOAuthFixture(t)

		_, err := f.refresh("nope")
		require.ErrorIs(t, err, ErrInvalidGrant)
	})

	t.Run("expired", func(t *testing.T) {
		f := newOAuthFixture(t)
		tokens := f.tokens(t)
		f.store.oauthRefreshTokens[0].ExpiresAt = time.Now().Add(-time.Second)

		_, err := f.refresh(tokens.RefreshToken)
		require.ErrorIs(t, err, ErrInvalidGrant)
	})

	t.Run("other client", func(t *testing.T) {
		f := newOAuthFixture(t)
		tokens := f.tokens(t)

		_, err := NewRefreshOAuthToken(f.store, f.store, f.members).Execute(context.Background(), tokens.RefreshToken, "other")
		require.ErrorIs(t, err, ErrInvalidGrant)

		_, err = f.refresh(tokens.RefreshToken)
		require.NoError(t, err, "a refresh by another client does not spend the token")
	})
}

func TestOAuthRefresh_AfterKeyRevoked(t *testing.T) {
	f := newOAuthFixture(t)
	tokens := f.tokens(t)
	key := f.store.apiKeys[0]

	require.NoError(t, NewRevokeAPIKey(f.store, f.audit).Execute(context.Background(), key.ID, f.userID))

	assert.Empty(t, f.store.oauthGrants)
	assert.Empty(t, f.store.oauthRefreshTokens)

	_, err := f.refresh(tokens.RefreshToken)
	require.ErrorIs(t, err, ErrInvalidGrant)
}

func TestPurgeWorkspaceAPIKeys_EndsOAuthGrants(t *testing.T) {
	f := newOAuthFixture(t)
	tokens := f.tokens(t)

	require.NoError(t, NewPurgeWorkspaceAPIKeys(f.store).Execute(context.Background(), f.workspaceID))

	assert.Empty(t, f.store.oauthGrants)

	_, err := f.refresh(tokens.RefreshToken)
	require.ErrorIs(t, err, ErrInvalidGrant)
}

func TestCleanupOAuthGrants(t *testing.T) {
	f := newOAuthFixture(t)
	live := f.tokens(t)
	f.tokens(t)

	// The second grant's key expired and its refresh token lapsed.
	f.store.apiKeys[1].ExpiresAt = new(time.Now().Add(-time.Minute))
	f.store.oauthRefreshTokens[1].ExpiresAt = time.Now().Add(-time.Minute)
	f.issueCode(t, OAuthAccessLevelRead)
	f.store.oauthCodes[0].ExpiresAt = time.Now().Add(-time.Minute)

	plainExpiry := time.Now().Add(-time.Minute)
	f.store.apiKeys = append(f.store.apiKeys, &APIKey{ID: uuid.New(), UserID: f.userID, WorkspaceID: f.workspaceID, ExpiresAt: &plainExpiry})

	require.NoError(t, NewCleanupOAuthGrants(f.store, f.store).Execute(context.Background()))

	require.Len(t, f.store.oauthGrants, 1)
	assert.Empty(t, f.store.oauthCodes)
	require.Len(t, f.store.apiKeys, 2, "the live OAuth key and the plain key stay")
	assert.Nil(t, f.store.apiKeys[1].OAuthGrantID)

	_, err := f.refresh(live.RefreshToken)
	require.NoError(t, err)
}

func TestOAuthExchange_RecordsLastGrant(t *testing.T) {
	f := newOAuthFixture(t)
	require.Nil(t, f.store.oauthClients[0].LastGrantedAt)

	f.tokens(t)

	require.NotNil(t, f.store.oauthClients[0].LastGrantedAt)
	assert.WithinDuration(t, time.Now(), *f.store.oauthClients[0].LastGrantedAt, time.Minute)
}

func TestCleanupOAuthGrants_DeletesUnusedClients(t *testing.T) {
	f := newOAuthFixture(t)
	f.tokens(t)

	now := time.Now()
	f.store.oauthClients[0].CreatedAt = now.Add(-90 * 24 * time.Hour)
	f.store.oauthClients[0].LastGrantedAt = new(now.Add(-60 * 24 * time.Hour))

	client := func(name string, created time.Time, lastGranted *time.Time) {
		f.store.oauthClients = append(f.store.oauthClients, &OAuthClient{
			ID: uuid.New(), ClientID: name, Name: name, RedirectURIs: []string{testRedirectURI}, CreatedAt: created, LastGrantedAt: lastGranted,
		})
	}
	client("fresh-never-granted", now.Add(-time.Hour), nil)
	client("old-never-granted", now.Add(-25*time.Hour), nil)
	client("grants-gone-recently", now.Add(-90*24*time.Hour), new(now.Add(-24*time.Hour)))
	client("grants-gone-long-ago", now.Add(-90*24*time.Hour), new(now.Add(-31*24*time.Hour)))

	require.NoError(t, NewCleanupOAuthGrants(f.store, f.store).Execute(context.Background()))

	left := make([]string, 0, len(f.store.oauthClients))
	for _, c := range f.store.oauthClients {
		left = append(left, c.ClientID)
	}

	assert.ElementsMatch(t, []string{f.client.ClientID, "fresh-never-granted", "grants-gone-recently"}, left,
		"a client with a live grant stays however old")
}

func TestIssueOAuthCode_CapsLevelAtRole(t *testing.T) {
	cases := []struct {
		role    WorkspaceRole
		allowed []OAuthAccessLevel
		refused []OAuthAccessLevel
	}{
		{WorkspaceRoleViewer, []OAuthAccessLevel{OAuthAccessLevelRead}, []OAuthAccessLevel{OAuthAccessLevelActions, OAuthAccessLevelWrite}},
		{WorkspaceRoleEditor, []OAuthAccessLevel{OAuthAccessLevelRead, OAuthAccessLevelActions, OAuthAccessLevelWrite}, nil},
		{WorkspaceRoleAdmin, []OAuthAccessLevel{OAuthAccessLevelRead, OAuthAccessLevelActions, OAuthAccessLevelWrite}, nil},
	}
	for _, tc := range cases {
		f := newOAuthFixture(t)
		f.members[f.userID][0].Role = tc.role

		for _, level := range tc.allowed {
			f.issueCode(t, level)
		}

		for _, level := range tc.refused {
			_, err := NewIssueOAuthCode(f.store, f.members).Execute(context.Background(), IssueOAuthCodeParams{
				ClientID:            f.client.ClientID,
				RedirectURI:         testRedirectURI,
				CodeChallenge:       challengeOf(testVerifier),
				CodeChallengeMethod: "S256",
				UserID:              f.userID,
				WorkspaceID:         f.workspaceID,
				Level:               level,
			})

			var validationErr *ValidationError
			require.ErrorAs(t, err, &validationErr, "%v approving %v", tc.role, level)
		}

		assert.Len(t, f.store.oauthCodes, len(tc.allowed))
	}
}

func TestOAuthRefresh_RevokesGrantOfFormerMember(t *testing.T) {
	f := newOAuthFixture(t)
	tokens := f.tokens(t)
	delete(f.members, f.userID)

	_, err := f.refresh(tokens.RefreshToken)
	require.ErrorIs(t, err, ErrInvalidGrant)

	assert.Empty(t, f.store.oauthGrants)
	assert.Empty(t, f.store.oauthRefreshTokens)
	assert.Empty(t, f.store.apiKeys)
}

func TestOAuthExchange_RejectsMalformedVerifier(t *testing.T) {
	cases := map[string]string{
		"42 characters":      strings.Repeat("a", 42),
		"129 characters":     strings.Repeat("a", 129),
		"reserved character": strings.Repeat("a", 50) + "+",
		"space":              strings.Repeat("a", 50) + " ",
	}
	for name, verifier := range cases {
		t.Run(name, func(t *testing.T) {
			f := newOAuthFixture(t)
			code, err := NewIssueOAuthCode(f.store, f.members).Execute(context.Background(), IssueOAuthCodeParams{
				ClientID:            f.client.ClientID,
				RedirectURI:         testRedirectURI,
				CodeChallenge:       challengeOf(verifier),
				CodeChallengeMethod: "S256",
				UserID:              f.userID,
				WorkspaceID:         f.workspaceID,
				Level:               OAuthAccessLevelRead,
			})
			require.NoError(t, err)

			_, err = f.exchange(code, verifier, f.client.ClientID, testRedirectURI)
			require.ErrorIs(t, err, ErrInvalidGrant)
			assert.Empty(t, f.store.oauthGrants)
		})
	}

	for _, verifier := range []string{strings.Repeat("a", 43), strings.Repeat("A1-._~", 21) + "zz"} {
		f := newOAuthFixture(t)
		code, err := NewIssueOAuthCode(f.store, f.members).Execute(context.Background(), IssueOAuthCodeParams{
			ClientID:            f.client.ClientID,
			RedirectURI:         testRedirectURI,
			CodeChallenge:       challengeOf(verifier),
			CodeChallengeMethod: "S256",
			UserID:              f.userID,
			WorkspaceID:         f.workspaceID,
			Level:               OAuthAccessLevelRead,
		})
		require.NoError(t, err)

		_, err = f.exchange(code, verifier, f.client.ClientID, testRedirectURI)
		require.NoError(t, err, "a %d-character verifier of unreserved characters is accepted", len(verifier))
	}
}

func TestCleanupOAuthGrants_KeepsLatestKeyOfLiveGrant(t *testing.T) {
	f := newOAuthFixture(t)
	tokens := f.tokens(t)
	f.store.apiKeys[0].ExpiresAt = new(time.Now().Add(-2 * time.Hour))
	key := f.store.apiKeys[0]

	require.NoError(t, NewCleanupOAuthGrants(f.store, f.store).Execute(context.Background()))

	require.Len(t, f.store.apiKeys, 1, "the grant stays visible and revocable through its latest key")
	assert.Equal(t, key.ID, f.store.apiKeys[0].ID)

	require.NoError(t, NewRevokeAPIKey(f.store, f.audit).Execute(context.Background(), key.ID, f.userID))

	_, err := f.refresh(tokens.RefreshToken)
	require.ErrorIs(t, err, ErrInvalidGrant)
}
