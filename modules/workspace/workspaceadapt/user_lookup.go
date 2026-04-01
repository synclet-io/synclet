package workspaceadapt

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
)

// UserLookupAdapter implements workspaceservice.UserLookup using auth module use cases.
type UserLookupAdapter struct {
	getUserByEmail *authservice.GetUserByEmail
	getUserByID    *authservice.GetUserByID
}

// NewUserLookupAdapter creates a new UserLookupAdapter.
func NewUserLookupAdapter(
	getUserByEmail *authservice.GetUserByEmail,
	getUserByID *authservice.GetUserByID,
) *UserLookupAdapter {
	return &UserLookupAdapter{
		getUserByEmail: getUserByEmail,
		getUserByID:    getUserByID,
	}
}

// GetUserByEmail returns user info for the given email, or nil if not found.
func (a *UserLookupAdapter) GetUserByEmail(ctx context.Context, email string) (*workspaceservice.UserInfo, error) {
	user, err := a.getUserByEmail.Execute(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("looking up user by email: %w", err)
	}

	if user == nil {
		return nil, nil
	}

	return &workspaceservice.UserInfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}, nil
}

// GetUserByID returns user info for the given ID, or nil if not found.
func (a *UserLookupAdapter) GetUserByID(ctx context.Context, id uuid.UUID) (*workspaceservice.UserInfo, error) {
	user, err := a.getUserByID.Execute(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("looking up user by ID: %w", err)
	}

	if user == nil {
		return nil, nil
	}

	return &workspaceservice.UserInfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}, nil
}
