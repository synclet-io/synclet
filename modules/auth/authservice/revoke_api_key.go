package authservice

import (
	"context"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// RevokeAPIKey deletes an API key.
type RevokeAPIKey struct {
	storage Storage
}

// NewRevokeAPIKey creates a new RevokeAPIKey use case.
func NewRevokeAPIKey(storage Storage) *RevokeAPIKey {
	return &RevokeAPIKey{storage: storage}
}

// Execute deletes the API key with the given ID, scoped to the calling user.
func (uc *RevokeAPIKey) Execute(ctx context.Context, id, userID uuid.UUID) error {
	return uc.storage.APIKeys().Delete(ctx, &APIKeyFilter{
		ID:     filter.Equals(id),
		UserID: filter.Equals(userID),
	})
}
