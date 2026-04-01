package authservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"golang.org/x/crypto/bcrypt"
)

// ChangePassword changes a user's password after verifying the current one.
type ChangePassword struct {
	storage Storage
}

// NewChangePassword creates a new ChangePassword use case.
func NewChangePassword(storage Storage) *ChangePassword {
	return &ChangePassword{storage: storage}
}

// Execute verifies the current password and updates to the new password.
func (uc *ChangePassword) Execute(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := uc.storage.Users().First(ctx, &UserFilter{
		ID: filter.Equals(userID),
	})
	if err != nil {
		return fmt.Errorf("fetching user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return errors.New("invalid current password")
	}

	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	user.PasswordHash = string(hash)

	if _, err := uc.storage.Users().Update(ctx, user); err != nil {
		return fmt.Errorf("updating password: %w", err)
	}

	return nil
}
