package authservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// Login authenticates a user and returns a token pair.
type Login struct {
	storage Storage
	config  Config
	audit   AuditRecorder
}

// NewLogin creates a new Login use case.
func NewLogin(storage Storage, config Config, audit AuditRecorder) *Login {
	return &Login{storage: storage, config: config, audit: audit}
}

// Execute authenticates a user by email and password.
func (uc *Login) Execute(ctx context.Context, email, password string) (*TokenPair, error) {
	user, err := uc.storage.Users().First(ctx, &UserFilter{
		Email: filter.Equals(email),
	})
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_, _ = comparePasswordWithHash(password, randomPasswordHash) // prevent timing attacks

			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("get user: %w", err)
	}

	ok, err := comparePasswordWithHash(password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("compare password: %w", err)
	}

	if !ok {
		return nil, ErrInvalidCredentials
	}

	tokens, err := generateTokenPair(ctx, uc.storage, uc.config, user)
	if err != nil {
		return nil, err
	}

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
