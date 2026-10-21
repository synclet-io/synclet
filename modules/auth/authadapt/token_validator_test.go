package authadapt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

type keyStorage struct {
	authservice.Storage

	keys *keyTable
}

func (s *keyStorage) APIKeys() authservice.APIKeysStorage { return s.keys }

type keyTable struct {
	authservice.APIKeysStorage

	keys []*authservice.APIKey
}

func (t *keyTable) First(_ context.Context, f *authservice.APIKeyFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*authservice.APIKey, error) {
	hash := f.KeyHash.(*filter.EqualsFilter[string]).Value
	for _, k := range t.keys {
		if k.KeyHash == hash {
			cp := *k

			return &cp, nil
		}
	}

	return nil, authservice.ErrAPIKeyNotFound
}

func (t *keyTable) Update(_ context.Context, k *authservice.APIKey) (*authservice.APIKey, error) {
	return k, nil
}

func hashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))

	return hex.EncodeToString(sum[:])
}

func TestTokenValidator_MCPKey(t *testing.T) {
	userID, wsID, grantID := uuid.New(), uuid.New(), uuid.New()
	level := authservice.OAuthAccessLevelActions
	expiresAt := time.Now().Add(time.Hour)
	keys := &keyTable{keys: []*authservice.APIKey{
		{ID: uuid.New(), UserID: userID, WorkspaceID: wsID, KeyHash: hashKey("synclet_sk_plain")},
		{ID: uuid.New(), UserID: userID, WorkspaceID: wsID, KeyHash: hashKey("synclet_sk_oauth"), ExpiresAt: &expiresAt, OAuthGrantID: &grantID, OAuthLevel: &level},
	}}
	validator := NewTokenValidator(nil, authservice.NewValidateAPIKey(&keyStorage{keys: keys}, nil))
	ctx := context.Background()

	_, _, err := validator.ValidateAPIKey(ctx, "synclet_sk_oauth")
	require.ErrorIs(t, err, connectutil.ErrInvalidCredential, "the public API refuses an OAuth key")

	gotUser, gotWS, err := validator.ValidateAPIKey(ctx, "synclet_sk_plain")
	require.NoError(t, err)
	assert.Equal(t, userID.String(), gotUser)
	assert.Equal(t, wsID.String(), gotWS)

	oauthKey, err := validator.ValidateMCPKey(ctx, "synclet_sk_oauth")
	require.NoError(t, err)
	assert.Equal(t, MCPKey{UserID: userID.String(), WorkspaceID: wsID.String(), Level: "actions", ExpiresAt: &expiresAt}, oauthKey)

	plainKey, err := validator.ValidateMCPKey(ctx, "synclet_sk_plain")
	require.NoError(t, err)
	assert.Equal(t, MCPKey{UserID: userID.String(), WorkspaceID: wsID.String()}, plainKey)

	_, err = validator.ValidateMCPKey(ctx, "synclet_sk_unknown")
	require.ErrorIs(t, err, connectutil.ErrInvalidCredential)
}
