package authservice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func newTestLogin(t *testing.T, store *fakeStorage) (*Login, *fakeClock) {
	t.Helper()

	clock := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	login := NewLogin(store, Config{}, &capturingAuditRecorder{})
	login.attempts = newLoginAttempts(clock.now)

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

	clock.t = clock.t.Add(loginLockoutDuration)

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

func TestLoginAttempts_FailuresOutsideWindowDoNotAccumulate(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	attempts := newLoginAttempts(clock.now)

	for range loginFailureLimit - 1 {
		attempts.recordFailure("a@example.com")
	}

	clock.t = clock.t.Add(loginFailureWindow + time.Second)

	attempts.recordFailure("a@example.com")
	assert.False(t, attempts.locked("a@example.com"))
}

func TestLoginAttempts_ResetClearsFailures(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	attempts := newLoginAttempts(clock.now)

	for range loginFailureLimit - 1 {
		attempts.recordFailure("a@example.com")
	}

	attempts.reset("a@example.com")

	attempts.recordFailure("a@example.com")
	assert.False(t, attempts.locked("a@example.com"))
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
