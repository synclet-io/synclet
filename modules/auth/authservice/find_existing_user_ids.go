package authservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// FindExistingUserIDs returns which of the given user ids have an account, with one query.
type FindExistingUserIDs struct {
	users UserIDStorage
}

// NewFindExistingUserIDs creates a new FindExistingUserIDs use case.
func NewFindExistingUserIDs(users UserIDStorage) *FindExistingUserIDs {
	return &FindExistingUserIDs{users: users}
}

// Execute returns the subset of ids that belong to existing users.
func (uc *FindExistingUserIDs) Execute(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	existing, err := uc.users.ExistingUserIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("finding existing users: %w", err)
	}

	return existing, nil
}
