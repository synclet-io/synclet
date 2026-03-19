package authservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// UpdateProfile updates a user's name.
type UpdateProfile struct {
	storage Storage
}

// NewUpdateProfile creates a new UpdateProfile use case.
func NewUpdateProfile(storage Storage) *UpdateProfile {
	return &UpdateProfile{storage: storage}
}

// Execute updates the name of the user with the given ID.
func (uc *UpdateProfile) Execute(ctx context.Context, userID uuid.UUID, name string) (*User, error) {
	user, err := uc.storage.Users().First(ctx, &UserFilter{
		ID: filter.Equals(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}

	user.Name = name

	updated, err := uc.storage.Users().Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
	}

	return updated, nil
}
