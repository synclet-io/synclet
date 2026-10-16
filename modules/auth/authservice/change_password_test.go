package authservice

import (
	"context"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type passwordStore struct {
	*fakeStorage
	refreshTokens *revokingRefreshTokens
}

func (s *passwordStore) RefreshTokens() RefreshTokensStorage { return s.refreshTokens }

func (s *passwordStore) ExecuteInTransaction(ctx context.Context, cb func(context.Context, Storage) error) error {
	return cb(ctx, s)
}

type revokingRefreshTokens struct {
	RefreshTokensStorage
	deleted []*RefreshTokenFilter
}

func (s *revokingRefreshTokens) Delete(_ context.Context, flt *RefreshTokenFilter) error {
	s.deleted = append(s.deleted, flt)

	return nil
}

func newPasswordStore(t *testing.T, password string) (*passwordStore, *User) {
	t.Helper()

	hash, err := hashPassword(password)
	require.NoError(t, err)

	user := &User{ID: uuid.New(), Email: "a@example.com", PasswordHash: hash}
	store := &passwordStore{fakeStorage: newFakeStorage(), refreshTokens: &revokingRefreshTokens{}}
	store.seedUser(user)

	return store, user
}

func TestChangePassword_RevokesOtherSessions(t *testing.T) {
	store, user := newPasswordStore(t, "old-password-123")
	recorder := &capturingAuditRecorder{}

	err := NewChangePassword(store, Config{MinPasswordLength: 8}, recorder).Execute(context.Background(), ChangePasswordParams{
		UserID:           user.ID,
		CurrentPassword:  "old-password-123",
		NewPassword:      "new-password-456",
		KeepRefreshToken: "current-session",
	})
	require.NoError(t, err)

	require.Len(t, store.refreshTokens.deleted, 1)
	revoked := store.refreshTokens.deleted[0]
	assert.Equal(t, filter.Equals(user.ID), revoked.UserID)
	assert.Equal(t, filter.NotEquals(hashToken("current-session")), revoked.TokenHash)

	updated, err := store.Users().First(context.Background(), &UserFilter{ID: filter.Equals(user.ID)})
	require.NoError(t, err)
	assert.NotEqual(t, user.PasswordHash, updated.PasswordHash)

	require.Len(t, recorder.events, 1)
	assert.Equal(t, map[string]any{"password_changed": true, "other_sessions_revoked": true}, recorder.events[0].After)
}

func TestChangePassword_WithoutSessionRevokesAll(t *testing.T) {
	store, user := newPasswordStore(t, "old-password-123")

	err := NewChangePassword(store, Config{MinPasswordLength: 8}, &capturingAuditRecorder{}).Execute(context.Background(), ChangePasswordParams{
		UserID:          user.ID,
		CurrentPassword: "old-password-123",
		NewPassword:     "new-password-456",
	})
	require.NoError(t, err)

	require.Len(t, store.refreshTokens.deleted, 1)
	assert.Nil(t, store.refreshTokens.deleted[0].TokenHash)
}

func TestChangePassword_WrongCurrentPasswordRevokesNothing(t *testing.T) {
	store, user := newPasswordStore(t, "old-password-123")
	recorder := &capturingAuditRecorder{}

	err := NewChangePassword(store, Config{MinPasswordLength: 8}, recorder).Execute(context.Background(), ChangePasswordParams{
		UserID:          user.ID,
		CurrentPassword: "wrong-password",
		NewPassword:     "new-password-456",
	})
	require.ErrorIs(t, err, ErrInvalidCurrentPassword)
	assert.Empty(t, store.refreshTokens.deleted)
	assert.Empty(t, recorder.events)
}

type lookupAPIKeys struct {
	APIKeysStorage
	err error
}

func (s *lookupAPIKeys) First(context.Context, *APIKeyFilter, ...optionutil.Option[dbutil.SelectOptions]) (*APIKey, error) {
	return nil, s.err
}

type apiKeyStore struct {
	Storage
	keys *lookupAPIKeys
}

func (s *apiKeyStore) APIKeys() APIKeysStorage { return s.keys }

func TestValidateAPIKey_SeparatesUnknownKeyFromLookupFailure(t *testing.T) {
	unknown := NewValidateAPIKey(&apiKeyStore{keys: &lookupAPIKeys{err: ErrAPIKeyNotFound}}, nil)
	_, err := unknown.Execute(context.Background(), "synclet_sk_unknown")
	require.ErrorIs(t, err, ErrInvalidAPIKey)

	failing := NewValidateAPIKey(&apiKeyStore{keys: &lookupAPIKeys{err: assert.AnError}}, nil)
	_, err = failing.Execute(context.Background(), "synclet_sk_any")
	require.ErrorIs(t, err, assert.AnError)
	assert.NotErrorIs(t, err, ErrInvalidAPIKey)
}
