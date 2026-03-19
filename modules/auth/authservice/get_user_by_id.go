package authservice

import (
	"context"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// GetUserByID retrieves a user by their ID.
type GetUserByID struct {
	storage Storage
}

// NewGetUserByID creates a new GetUserByID use case.
func NewGetUserByID(storage Storage) *GetUserByID {
	return &GetUserByID{storage: storage}
}

// Execute returns the user with the given ID.
func (uc *GetUserByID) Execute(ctx context.Context, id uuid.UUID) (*User, error) {
	return uc.storage.Users().First(ctx, &UserFilter{
		ID: filter.Equals(id),
	})
}
