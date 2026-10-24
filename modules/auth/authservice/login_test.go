package authservice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/synclet-io/synclet/pkg/connectutil"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func newTestLogin(t *testing.T, store *fakeStorage) (*Login, *fakeClock) {
	t.Helper()

	clock := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	rateLimits := NewRateLimits(store, store)
	rateLimits.now = clock.now
	login := NewLogin(store, Config{}, &capturingAuditRecorder{}, rateLimits)

	return login, clock
}

func TestLogin_LocksEmailAfterRepeatedFailures(t *testing.T) {
	ctx := context.Background()

	hash, err := bcrypt.GenerateFromPassword([]byte(normalizePassword("correct")), bcrypt.MinCost)
	require.NoError(t, err)

	store := newFakeStorage()
	store.seedUser(&User{ID: uuid.New(), Email: "a@example.com", PasswordHash: string(hash)})

	login, clock := newTestLogin(t, store)

	for range loginFailureLimit {
		_, err := login.Execute(ctx, "a@example.com", "wrong")
		require.ErrorIs(t, err, ErrInvalidCredentials)
	}

	_, err = login.Execute(ctx, "A@Example.com ", "correct")
	require.ErrorIs(t, err, ErrTooManyLoginAttempts, "correct password is refused while locked")

	clock.t = clock.t.Add(loginFailureWindow)

	_, err = login.Execute(ctx, "a@example.com", "wrong")
	require.ErrorIs(t, err, ErrInvalidCredentials, "lockout expires")
}

func TestLogin_LocksUnknownEmailsToo(t *testing.T) {
	ctx := context.Background()
	login, _ := newTestLogin(t, newFakeStorage())

	for range loginFailureLimit {
		_, err := login.Execute(ctx, "ghost@example.com", "wrong")
		require.ErrorIs(t, err, ErrInvalidCredentials)
	}

	_, err := login.Execute(ctx, "ghost@example.com", "wrong")
	require.ErrorIs(t, err, ErrTooManyLoginAttempts)

	_, err = login.Execute(ctx, "other@example.com", "wrong")
	require.ErrorIs(t, err, ErrInvalidCredentials, "other emails are unaffected")
}

func seedPasswordUser(t *testing.T, store *fakeStorage, email, password string) {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(normalizePassword(password)), bcrypt.MinCost)
	require.NoError(t, err)

	store.seedUser(&User{ID: uuid.New(), Email: email, PasswordHash: string(hash)})
}

func TestLogin_FailuresOutsideWindowDoNotAccumulate(t *testing.T) {
	ctx := context.Background()
	login, clock := newTestLogin(t, newFakeStorage())

	for range loginFailureLimit - 1 {
		_, _ = login.Execute(ctx, "a@example.com", "wrong")
	}

	clock.t = clock.t.Add(loginFailureWindow + time.Second)

	_, _ = login.Execute(ctx, "a@example.com", "wrong")

	_, err := login.Execute(ctx, "a@example.com", "wrong")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_SuccessClearsFailures(t *testing.T) {
	ctx := context.Background()

	store := newFakeStorage()
	seedPasswordUser(t, store, "a@example.com", "correct")

	login, _ := newTestLogin(t, store)

	for range loginFailureLimit - 1 {
		_, _ = login.Execute(ctx, "a@example.com", "wrong")
	}

	_, err := login.Execute(ctx, "a@example.com", "correct")
	require.NoError(t, err)

	_, _ = login.Execute(ctx, "a@example.com", "wrong")

	_, err = login.Execute(ctx, "a@example.com", "wrong")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_LockoutIsPerEmailAndIP(t *testing.T) {
	store := newFakeStorage()
	seedPasswordUser(t, store, "a@example.com", "correct")

	login, _ := newTestLogin(t, store)
	fromA := connectutil.ContextWithClientIP(context.Background(), "203.0.113.1")
	fromB := connectutil.ContextWithClientIP(context.Background(), "203.0.113.2")

	for range loginFailureLimit {
		_, err := login.Execute(fromA, "a@example.com", "wrong")
		require.ErrorIs(t, err, ErrInvalidCredentials)
	}

	_, err := login.Execute(fromA, "a@example.com", "correct")
	require.ErrorIs(t, err, ErrTooManyLoginAttempts, "the client that failed is locked")

	_, err = login.Execute(fromB, "a@example.com", "correct")
	require.NoError(t, err, "another client is not locked out")
}

func TestLoginWithUserInfo_PassesLockoutThrough(t *testing.T) {
	ctx := context.Background()
	login, _ := newTestLogin(t, newFakeStorage())

	for range loginFailureLimit {
		_, _ = login.Execute(ctx, "a@example.com", "wrong")
	}

	_, err := NewLoginWithUserInfo(login, nil, nil, nil).Execute(ctx, LoginWithUserInfoParams{Email: "a@example.com", Password: "wrong"})
	require.ErrorIs(t, err, ErrTooManyLoginAttempts)
	assert.NotErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_NoPasswordHashIsWrongPassword(t *testing.T) {
	ctx := context.Background()

	store := newFakeStorage()
	store.seedUser(&User{ID: uuid.New(), Email: "oidc@example.com", PasswordHash: ""})

	login, _ := newTestLogin(t, store)

	for range loginFailureLimit {
		_, err := login.Execute(ctx, "oidc@example.com", "anything")
		require.ErrorIs(t, err, ErrInvalidCredentials)
	}

	_, err := login.Execute(ctx, "oidc@example.com", "anything")
	require.ErrorIs(t, err, ErrTooManyLoginAttempts)
}
