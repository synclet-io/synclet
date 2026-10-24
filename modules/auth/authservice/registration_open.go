package authservice

import (
	"context"
	"errors"
	"fmt"
)

// RegistrationOpen reports whether new accounts may sign up: when the config
// allows registration, or while no user exists yet, so a fresh install can
// create its first account.
type RegistrationOpen struct {
	storage Storage
	config  Config
}

// NewRegistrationOpen creates a new RegistrationOpen use case.
func NewRegistrationOpen(storage Storage, config Config) *RegistrationOpen {
	return &RegistrationOpen{storage: storage, config: config}
}

// Execute reports whether sign-up is open.
func (uc *RegistrationOpen) Execute(ctx context.Context) (bool, error) {
	if uc.config.RegistrationEnabled {
		return true, nil
	}

	if _, err := uc.storage.Users().First(ctx, &UserFilter{}); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return true, nil
		}

		return false, fmt.Errorf("checking for existing users: %w", err)
	}

	return false, nil
}
