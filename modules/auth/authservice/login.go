package authservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/pkg/connectutil"
)

// ErrTooManyLoginAttempts is returned while an email is locked out for a
// client after repeated failed logins.
const ErrTooManyLoginAttempts baseErr = "too many failed login attempts, try again later"

const (
	loginFailureLimit  = 5
	loginFailureWindow = 15 * time.Minute
)

// Login authenticates a user and returns a token pair.
type Login struct {
	storage    Storage
	config     Config
	audit      AuditRecorder
	rateLimits *RateLimits
}

// NewLogin creates a new Login use case.
func NewLogin(storage Storage, config Config, audit AuditRecorder, rateLimits *RateLimits) *Login {
	return &Login{storage: storage, config: config, audit: audit, rateLimits: rateLimits}
}

// Execute authenticates a user by email and password. Failures are counted
// per email and client IP: after loginFailureLimit failures within
// loginFailureWindow, that email is locked for that client until the window
// ends, whether or not the account exists, so the lockout does not reveal
// which emails are registered and a client cannot lock the account out for
// others.
func (uc *Login) Execute(ctx context.Context, email, password string) (*TokenPair, error) {
	key := loginFailureKey(ctx, email)

	failures, err := uc.rateLimits.Peek(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("reading login failures: %w", err)
	}

	if failures >= loginFailureLimit {
		return nil, ErrTooManyLoginAttempts
	}

	user, err := uc.storage.Users().First(ctx, &UserFilter{
		Email: filter.Equals(email),
	})
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_, _ = comparePasswordWithHash(password, randomPasswordHash) // prevent timing attacks

			return nil, uc.fail(ctx, key)
		}

		return nil, fmt.Errorf("get user: %w", err)
	}

	if user.PasswordHash == "" {
		_, _ = comparePasswordWithHash(password, randomPasswordHash) // an OIDC-only account answers like a wrong password

		return nil, uc.fail(ctx, key)
	}

	ok, err := comparePasswordWithHash(password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("compare password: %w", err)
	}

	if !ok {
		return nil, uc.fail(ctx, key)
	}

	tokens, err := generateTokenPair(ctx, uc.storage, uc.config, user)
	if err != nil {
		return nil, err
	}

	if err := uc.rateLimits.Reset(ctx, key); err != nil {
		return nil, fmt.Errorf("resetting login failures: %w", err)
	}

	recordLogin(ctx, uc.audit, user)

	return tokens, nil
}

// fail counts a failed login and returns ErrInvalidCredentials.
func (uc *Login) fail(ctx context.Context, key string) error {
	if _, err := uc.rateLimits.Hit(ctx, key, loginFailureLimit, loginFailureWindow); err != nil {
		return fmt.Errorf("counting login failure: %w", err)
	}

	return ErrInvalidCredentials
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

func loginFailureKey(ctx context.Context, email string) string {
	ip, _ := connectutil.ClientIPFromContext(ctx)

	return "login|" + strings.ToLower(strings.TrimSpace(email)) + "|" + ip
}
