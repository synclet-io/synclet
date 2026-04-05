package authservice

import (
	"context"
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
)

// Register creates a new user account.
type Register struct {
	storage Storage
	cfg     Config
}

// NewRegister creates a new Register use case.
func NewRegister(storage Storage, cfg Config) *Register {
	return &Register{storage: storage, cfg: cfg}
}

// Execute creates a new user with hashed password.
func (uc *Register) Execute(ctx context.Context, email, password, name string) (*User, error) {
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, &ValidationError{Message: "invalid email format"}
	}

	if err := ValidatePassword(password, uc.cfg.MinPasswordLength); err != nil {
		return nil, err
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hash,
		Name:         name,
		CreatedAt:    time.Now(),
	}

	created, err := uc.storage.Users().Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return created, nil
}
