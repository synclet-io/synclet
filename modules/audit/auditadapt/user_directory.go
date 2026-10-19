package auditadapt

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/audit/auditservice"
	"github.com/synclet-io/synclet/modules/auth/authservice"
)

// UserDirectory bridges the audit module's UserDirectory port to the auth
// module's user lookup use cases.
type UserDirectory struct {
	getUser      *authservice.GetUserByID
	findExisting *authservice.FindExistingUserIDs
}

// NewUserDirectory creates the adapter.
func NewUserDirectory(getUser *authservice.GetUserByID, findExisting *authservice.FindExistingUserIDs) *UserDirectory {
	return &UserDirectory{getUser: getUser, findExisting: findExisting}
}

var _ auditservice.UserDirectory = (*UserDirectory)(nil)

// ExistingUsers reports which of ids still have a user account, in one query.
func (d *UserDirectory) ExistingUsers(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	found, err := d.findExisting.Execute(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("finding users: %w", err)
	}

	existing := make(map[uuid.UUID]bool, len(found))
	for _, id := range found {
		existing[id] = true
	}

	return existing, nil
}

// UserEmail returns the user's email, or "" when the account does not exist.
func (d *UserDirectory) UserEmail(ctx context.Context, id uuid.UUID) (string, error) {
	user, err := d.getUser.Execute(ctx, id)

	var notFound authservice.NotFoundError
	if errors.As(err, &notFound) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("getting user: %w", err)
	}

	return user.Email, nil
}
