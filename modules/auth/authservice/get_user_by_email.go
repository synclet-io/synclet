package authservice

import (
	"context"
	"fmt"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// GetUserByEmail retrieves a user by their email address.
type GetUserByEmail struct {
	storage Storage
}

// NewGetUserByEmail creates a new GetUserByEmail use case.
func NewGetUserByEmail(storage Storage) *GetUserByEmail {
	return &GetUserByEmail{storage: storage}
}

// Execute returns the user with the given email, or nil if not found.
func (uc *GetUserByEmail) Execute(ctx context.Context, email string) (*User, error) {
	user, err := uc.storage.Users().First(ctx, &UserFilter{Email: filter.Equals(email)})
	if err != nil {
		return nil, fmt.Errorf("getting user by email: %w", err)
	}

	return user, nil
}
