package authservice

import (
	"context"

	"github.com/google/uuid"
)

// UserIDStorage answers id-only questions about user accounts.
type UserIDStorage interface {
	// ExistingUserIDs returns which of ids belong to existing users, reading only the id column, in one query.
	ExistingUserIDs(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error)
}
