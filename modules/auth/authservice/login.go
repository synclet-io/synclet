package authservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// ErrTooManyLoginAttempts is returned while an email is locked out after
// repeated failed logins.
const ErrTooManyLoginAttempts baseErr = "too many failed login attempts, try again later"

const (
	loginFailureLimit     = 5
	loginFailureWindow    = 15 * time.Minute
	loginLockoutDuration  = 15 * time.Minute
	loginMaxTrackedEmails = 10000
)

// Login authenticates a user and returns a token pair.
type Login struct {
	storage  Storage
	config   Config
	audit    AuditRecorder
	attempts *loginAttempts
}

// NewLogin creates a new Login use case.
func NewLogin(storage Storage, config Config, audit AuditRecorder) *Login {
	return &Login{storage: storage, config: config, audit: audit, attempts: newLoginAttempts(time.Now)}
}

// Execute authenticates a user by email and password. After loginFailureLimit
// failures for one email within loginFailureWindow, that email is locked for
// loginLockoutDuration whether or not the account exists, so the lockout does
// not reveal which emails are registered.
func (uc *Login) Execute(ctx context.Context, email, password string) (*TokenPair, error) {
	if uc.attempts.locked(email) {
		return nil, ErrTooManyLoginAttempts
	}

	user, err := uc.storage.Users().First(ctx, &UserFilter{
		Email: filter.Equals(email),
	})
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_, _ = comparePasswordWithHash(password, randomPasswordHash) // prevent timing attacks

			uc.attempts.recordFailure(email)

			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("get user: %w", err)
	}

	ok, err := comparePasswordWithHash(password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("compare password: %w", err)
	}

	if !ok {
		uc.attempts.recordFailure(email)

		return nil, ErrInvalidCredentials
	}

	tokens, err := generateTokenPair(ctx, uc.storage, uc.config, user)
	if err != nil {
		return nil, err
	}

	uc.attempts.reset(email)
	recordLogin(ctx, uc.audit, user)

	return tokens, nil
}

func recordLogin(ctx context.Context, audit AuditRecorder, user *User) {
	audit.Record(ctx, AuditEvent{
		ActorID:       user.ID,
		ActorLabel:    user.Email,
		Action:        "login",
		ResourceType:  "user",
		ResourceID:    user.ID,
		ResourceLabel: user.Email,
	})
}

// loginAttempts counts failed logins per email in process memory. Each server
// replica keeps its own counters.
type loginAttempts struct {
	mu      sync.Mutex
	now     func() time.Time
	entries map[string]*loginAttempt
}

type loginAttempt struct {
	failures     int
	firstFailure time.Time
	lockedUntil  time.Time
}

func newLoginAttempts(now func() time.Time) *loginAttempts {
	return &loginAttempts{now: now, entries: make(map[string]*loginAttempt)}
}

func (a *loginAttempts) locked(email string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	entry, ok := a.entries[loginAttemptKey(email)]

	return ok && a.now().Before(entry.lockedUntil)
}

func (a *loginAttempts) recordFailure(email string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := a.now()
	key := loginAttemptKey(email)

	entry, ok := a.entries[key]
	if !ok {
		if len(a.entries) >= loginMaxTrackedEmails {
			a.pruneLocked(now)
		}

		if len(a.entries) >= loginMaxTrackedEmails {
			return
		}

		entry = &loginAttempt{}
		a.entries[key] = entry
	}

	if entry.failures == 0 || now.Sub(entry.firstFailure) > loginFailureWindow {
		entry.failures = 0
		entry.firstFailure = now
	}

	entry.failures++
	if entry.failures >= loginFailureLimit {
		entry.failures = 0
		entry.lockedUntil = now.Add(loginLockoutDuration)
	}
}

func (a *loginAttempts) reset(email string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.entries, loginAttemptKey(email))
}

func (a *loginAttempts) pruneLocked(now time.Time) {
	for key, entry := range a.entries {
		if !now.Before(entry.lockedUntil) && now.Sub(entry.firstFailure) > loginFailureWindow {
			delete(a.entries, key)
		}
	}
}

func loginAttemptKey(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
