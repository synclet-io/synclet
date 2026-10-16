package authservice

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resetStore struct {
	*passwordStore
	resetTokens *fakeResetTokens
}

func (s *resetStore) PasswordResetTokens() PasswordResetTokensStorage { return s.resetTokens }

func (s *resetStore) ExecuteInTransaction(ctx context.Context, cb func(context.Context, Storage) error) error {
	return cb(ctx, s)
}

type fakeResetTokens struct {
	PasswordResetTokensStorage

	mu     sync.Mutex
	tokens []*PasswordResetToken
}

func (s *fakeResetTokens) matches(token *PasswordResetToken, flt *PasswordResetTokenFilter) bool {
	if flt == nil {
		return true
	}

	if id, ok := equalsValue[uuid.UUID](flt.ID); ok && token.ID != id {
		return false
	}

	if userID, ok := equalsValue[uuid.UUID](flt.UserID); ok && token.UserID != userID {
		return false
	}

	if hash, ok := equalsValue[string](flt.TokenHash); ok && token.TokenHash != hash {
		return false
	}

	if less, ok := flt.ExpiresAt.(*filter.LessFilter[time.Time]); ok && !token.ExpiresAt.Before(less.Value) {
		return false
	}

	return true
}

func (s *fakeResetTokens) Create(_ context.Context, token *PasswordResetToken) (*PasswordResetToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *token
	s.tokens = append(s.tokens, &cp)

	return token, nil
}

func (s *fakeResetTokens) First(_ context.Context, flt *PasswordResetTokenFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*PasswordResetToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, token := range s.tokens {
		if s.matches(token, flt) {
			cp := *token

			return &cp, nil
		}
	}

	return nil, ErrPasswordResetTokenNotFound
}

func (s *fakeResetTokens) Delete(_ context.Context, flt *PasswordResetTokenFilter) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	kept := s.tokens[:0]
	for _, token := range s.tokens {
		if !s.matches(token, flt) {
			kept = append(kept, token)
		}
	}

	s.tokens = kept

	return nil
}

func (s *fakeResetTokens) all() []*PasswordResetToken {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]*PasswordResetToken(nil), s.tokens...)
}

type capturingResetSender struct {
	sent chan PasswordResetEmailParams
}

func newCapturingResetSender() *capturingResetSender {
	return &capturingResetSender{sent: make(chan PasswordResetEmailParams, 4)}
}

func (s *capturingResetSender) SendPasswordResetEmail(_ context.Context, params PasswordResetEmailParams) error {
	s.sent <- params

	return nil
}

const resetTestPassword = "forgotten-password-1"

func newResetStore(t *testing.T) (*resetStore, *User) {
	t.Helper()

	store, user := newPasswordStore(t, resetTestPassword)

	return &resetStore{passwordStore: store, resetTokens: &fakeResetTokens{}}, user
}

func resetConfig(emailEnabled bool) Config {
	return Config{
		MinPasswordLength:     8,
		EmailEnabled:          emailEnabled,
		FrontendURL:           "https://synclet.example.com/",
		PasswordResetCooldown: 2 * time.Minute,
	}
}

func receiveResetEmail(t *testing.T, sender *capturingResetSender) PasswordResetEmailParams {
	t.Helper()

	select {
	case sent := <-sender.sent:
		return sent
	case <-time.After(5 * time.Second):
		t.Fatal("reset email was not sent")
	}

	return PasswordResetEmailParams{}
}

func newRequestPasswordReset(store Storage, cfg Config, sender PasswordResetEmailSender, audit AuditRecorder) *RequestPasswordReset {
	return NewRequestPasswordReset(cfg, NewIssuePasswordResetLink(store, cfg), sender, audit, nil)
}

func tokenFromURL(t *testing.T, raw string) string {
	t.Helper()

	parsed, err := url.Parse(raw)
	require.NoError(t, err)

	return parsed.Query().Get("token")
}

func TestRequestPasswordReset_KnownEmailIssuesTokenAndSendsLink(t *testing.T) {
	store, user := newResetStore(t)
	sender := newCapturingResetSender()
	recorder := &capturingAuditRecorder{}

	err := newRequestPasswordReset(store, resetConfig(true), sender, recorder).Execute(context.Background(), "  A@Example.com ")
	require.NoError(t, err)

	tokens := store.resetTokens.all()
	require.Len(t, tokens, 1)
	assert.Equal(t, user.ID, tokens[0].UserID)
	assert.WithinDuration(t, time.Now().Add(time.Hour), tokens[0].ExpiresAt, time.Minute)

	sent := receiveResetEmail(t, sender)

	assert.Equal(t, user.Email, sent.To)
	assert.True(t, strings.HasPrefix(sent.ResetURL, "https://synclet.example.com/reset-password?token="), sent.ResetURL)

	raw := tokenFromURL(t, sent.ResetURL)
	require.NotEmpty(t, raw)
	assert.NotEqual(t, raw, tokens[0].TokenHash)
	assert.Equal(t, hashToken(raw), tokens[0].TokenHash)

	require.Len(t, recorder.events, 1)
	assert.Equal(t, "update", recorder.events[0].Action)
	assert.Equal(t, "user", recorder.events[0].ResourceType)
	assert.Equal(t, user.ID, recorder.events[0].ResourceID)
	assert.Equal(t, map[string]any{"password_reset_requested": true}, recorder.events[0].After)
}

func TestRequestPasswordReset_UnknownEmailDoesNothing(t *testing.T) {
	store, _ := newResetStore(t)
	sender := newCapturingResetSender()
	recorder := &capturingAuditRecorder{}

	err := newRequestPasswordReset(store, resetConfig(true), sender, recorder).Execute(context.Background(), "nobody@example.com")
	require.NoError(t, err)

	assert.Empty(t, store.resetTokens.all())
	assert.Empty(t, sender.sent)
	assert.Empty(t, recorder.events)
}

func TestRequestPasswordReset_EmailDisabledDoesNothing(t *testing.T) {
	store, _ := newResetStore(t)
	sender := newCapturingResetSender()
	recorder := &capturingAuditRecorder{}

	err := newRequestPasswordReset(store, resetConfig(false), sender, recorder).Execute(context.Background(), "a@example.com")
	require.NoError(t, err)

	assert.Empty(t, store.resetTokens.all())
	assert.Empty(t, sender.sent)
	assert.Empty(t, recorder.events)
}

func TestRequestPasswordReset_WithinCooldownKeepsFirstToken(t *testing.T) {
	store, _ := newResetStore(t)
	sender := newCapturingResetSender()
	recorder := &capturingAuditRecorder{}
	request := newRequestPasswordReset(store, resetConfig(true), sender, recorder)

	require.NoError(t, request.Execute(context.Background(), "a@example.com"))
	first := receiveResetEmail(t, sender)

	require.NoError(t, request.Execute(context.Background(), "a@example.com"))

	tokens := store.resetTokens.all()
	require.Len(t, tokens, 1)
	assert.Equal(t, hashToken(tokenFromURL(t, first.ResetURL)), tokens[0].TokenHash)
	assert.Len(t, recorder.events, 1)

	select {
	case extra := <-sender.sent:
		t.Fatalf("unexpected second reset email to %s", extra.To)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestRequestPasswordReset_AfterCooldownIssuesNewToken(t *testing.T) {
	store, _ := newResetStore(t)
	sender := newCapturingResetSender()
	recorder := &capturingAuditRecorder{}
	request := newRequestPasswordReset(store, resetConfig(true), sender, recorder)

	require.NoError(t, request.Execute(context.Background(), "a@example.com"))
	first := receiveResetEmail(t, sender)

	store.resetTokens.mu.Lock()
	store.resetTokens.tokens[0].CreatedAt = time.Now().Add(-3 * time.Minute)
	store.resetTokens.mu.Unlock()

	require.NoError(t, request.Execute(context.Background(), "a@example.com"))
	second := receiveResetEmail(t, sender)

	tokens := store.resetTokens.all()
	require.Len(t, tokens, 1)
	assert.Equal(t, hashToken(tokenFromURL(t, second.ResetURL)), tokens[0].TokenHash)
	assert.NotEqual(t, first.ResetURL, second.ResetURL)
	assert.Len(t, recorder.events, 2)
}

func TestIssuePasswordResetLink_ReplacesPreviousToken(t *testing.T) {
	store, _ := newResetStore(t)
	issue := NewIssuePasswordResetLink(store, resetConfig(true))

	first, err := issue.Execute(context.Background(), IssuePasswordResetLinkParams{Email: "a@example.com"})
	require.NoError(t, err)

	second, err := issue.Execute(context.Background(), IssuePasswordResetLinkParams{Email: "a@example.com"})
	require.NoError(t, err)

	tokens := store.resetTokens.all()
	require.Len(t, tokens, 1)
	assert.Equal(t, hashToken(tokenFromURL(t, second.URL)), tokens[0].TokenHash)
	assert.NotEqual(t, hashToken(tokenFromURL(t, first.URL)), tokens[0].TokenHash)
}

func TestIssuePasswordResetLink_UnknownEmailIsNotFound(t *testing.T) {
	store, _ := newResetStore(t)

	_, err := NewIssuePasswordResetLink(store, resetConfig(false)).Execute(context.Background(), IssuePasswordResetLinkParams{Email: "nobody@example.com"})
	require.ErrorIs(t, err, ErrUserNotFound)
	assert.Empty(t, store.resetTokens.all())
}

func issueResetToken(t *testing.T, store *resetStore) string {
	t.Helper()

	link, err := NewIssuePasswordResetLink(store, resetConfig(false)).Execute(context.Background(), IssuePasswordResetLinkParams{Email: "a@example.com"})
	require.NoError(t, err)

	return tokenFromURL(t, link.URL)
}

func TestResetPassword_ValidTokenSetsPasswordAndRevokesSessions(t *testing.T) {
	store, user := newResetStore(t)
	raw := issueResetToken(t, store)
	recorder := &capturingAuditRecorder{}

	err := NewResetPassword(store, resetConfig(false), recorder).Execute(context.Background(), raw, "new-password-456")
	require.NoError(t, err)

	updated, err := store.Users().First(context.Background(), &UserFilter{ID: filter.Equals(user.ID)})
	require.NoError(t, err)

	matches, err := comparePasswordWithHash("new-password-456", updated.PasswordHash)
	require.NoError(t, err)
	assert.True(t, matches)

	assert.Empty(t, store.resetTokens.all())

	require.Len(t, store.refreshTokens.deleted, 1)
	assert.Equal(t, &RefreshTokenFilter{UserID: filter.Equals(user.ID)}, store.refreshTokens.deleted[0])

	require.Len(t, recorder.events, 1)
	event := recorder.events[0]
	assert.Equal(t, "update", event.Action)
	assert.Equal(t, "user", event.ResourceType)
	assert.Equal(t, user.ID, event.ResourceID)
	assert.Equal(t, map[string]any{"password_reset": true, "sessions_revoked": true}, event.After)
	assert.Nil(t, event.Before)
}

func TestResetPassword_TokenIsSingleUse(t *testing.T) {
	store, _ := newResetStore(t)
	raw := issueResetToken(t, store)
	reset := NewResetPassword(store, resetConfig(false), &capturingAuditRecorder{})

	require.NoError(t, reset.Execute(context.Background(), raw, "new-password-456"))

	err := reset.Execute(context.Background(), raw, "other-password-789")
	require.ErrorIs(t, err, ErrInvalidResetToken)
}

func TestResetPassword_ExpiredTokenIsRejected(t *testing.T) {
	store, user := newResetStore(t)
	_, err := store.resetTokens.Create(context.Background(), &PasswordResetToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashToken("expired-token"),
		ExpiresAt: time.Now().Add(-time.Minute),
		CreatedAt: time.Now().Add(-time.Hour),
	})
	require.NoError(t, err)

	recorder := &capturingAuditRecorder{}

	err = NewResetPassword(store, resetConfig(false), recorder).Execute(context.Background(), "expired-token", "new-password-456")
	require.ErrorIs(t, err, ErrInvalidResetToken)
	assert.Empty(t, store.refreshTokens.deleted)
	assert.Empty(t, recorder.events)
}

func TestResetPassword_UnknownTokenIsRejected(t *testing.T) {
	store, _ := newResetStore(t)

	err := NewResetPassword(store, resetConfig(false), &capturingAuditRecorder{}).Execute(context.Background(), "unknown-token", "new-password-456")
	require.ErrorIs(t, err, ErrInvalidResetToken)
}

func TestResetPassword_WeakPasswordKeepsToken(t *testing.T) {
	store, user := newResetStore(t)
	raw := issueResetToken(t, store)

	err := NewResetPassword(store, resetConfig(false), &capturingAuditRecorder{}).Execute(context.Background(), raw, "short")

	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Len(t, store.resetTokens.all(), 1)
	assert.Empty(t, store.refreshTokens.deleted)

	unchanged, err := store.Users().First(context.Background(), &UserFilter{ID: filter.Equals(user.ID)})
	require.NoError(t, err)

	matches, err := comparePasswordWithHash(resetTestPassword, unchanged.PasswordHash)
	require.NoError(t, err)
	assert.True(t, matches)
}

func TestCleanupExpiredTokens_SweepsExpiredResetTokens(t *testing.T) {
	store, user := newResetStore(t)
	raw := issueResetToken(t, store)
	_, err := store.resetTokens.Create(context.Background(), &PasswordResetToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: hashToken("expired-token"),
		ExpiresAt: time.Now().Add(-time.Minute),
	})
	require.NoError(t, err)

	cleanup := &cleanupStore{resetStore: store}
	require.NoError(t, NewCleanupExpiredTokens(cleanup, NewStateStore(cleanup)).Execute(context.Background()))

	tokens := store.resetTokens.all()
	require.Len(t, tokens, 1)
	assert.Equal(t, user.ID, tokens[0].UserID)
	assert.Equal(t, hashToken(raw), tokens[0].TokenHash)
}

type cleanupStore struct {
	*resetStore
}

func (s *cleanupStore) RefreshTokens() RefreshTokensStorage { return emptyRefreshTokens{} }
func (s *cleanupStore) OIDCStates() OIDCStatesStorage       { return emptyOIDCStates{} }

type emptyRefreshTokens struct{ RefreshTokensStorage }

func (emptyRefreshTokens) Find(context.Context, *RefreshTokenFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*RefreshToken, error) {
	return nil, nil
}

type emptyOIDCStates struct{ OIDCStatesStorage }

func (emptyOIDCStates) Find(context.Context, *OIDCStateFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*OIDCState, error) {
	return nil, nil
}
