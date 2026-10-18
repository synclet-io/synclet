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
// module's GetUserByID use case.
type UserDirectory struct {
	getUser *authservice.GetUserByID
}

// NewUserDirectory creates the adapter.
func NewUserDirectory(getUser *authservice.GetUserByID) *UserDirectory {
	return &UserDirectory{getUser: getUser}
}

var _ auditservice.UserDirectory = (*UserDirectory)(nil)

// UserExists reports whether the user account exists.
func (d *UserDirectory) UserExists(ctx context.Context, id uuid.UUID) (bool, error) {
	_, err := d.getUser.Execute(ctx, id)

	var notFound authservice.NotFoundError
	if errors.As(err, &notFound) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("getting user: %w", err)
	}

	return true, nil
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
