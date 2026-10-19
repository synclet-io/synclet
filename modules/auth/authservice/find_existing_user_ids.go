package authservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// FindExistingUserIDs returns which of the given user ids have an account, with one query.
type FindExistingUserIDs struct {
	storage Storage
}

// NewFindExistingUserIDs creates a new FindExistingUserIDs use case.
func NewFindExistingUserIDs(storage Storage) *FindExistingUserIDs {
	return &FindExistingUserIDs{storage: storage}
}

// Execute returns the subset of ids that belong to existing users.
func (uc *FindExistingUserIDs) Execute(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	users, err := uc.storage.Users().Find(ctx, &UserFilter{ID: filter.In(ids...)})
	if err != nil {
		return nil, fmt.Errorf("finding users: %w", err)
	}

	existing := make([]uuid.UUID, 0, len(users))
	for _, user := range users {
		existing = append(existing, user.ID)
	}

	return existing, nil
}
