package authservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// WorkspaceRole is a user's role within a workspace.
type WorkspaceRole int

const (
	WorkspaceRoleUnspecified WorkspaceRole = iota
	WorkspaceRoleAdmin
	WorkspaceRoleEditor
	WorkspaceRoleViewer
)

// WorkspaceMembership is a workspace the user belongs to, with the user's role in it.
type WorkspaceMembership struct {
	WorkspaceID   uuid.UUID
	WorkspaceName string
	Role          WorkspaceRole
}

// WorkspaceMembershipLister lists the workspaces a user belongs to.
type WorkspaceMembershipLister interface {
	ListForUser(ctx context.Context, userID uuid.UUID) ([]WorkspaceMembership, error)
}

// ListUserWorkspaces returns every workspace membership of a user.
type ListUserWorkspaces struct {
	lister WorkspaceMembershipLister
}

// NewListUserWorkspaces creates a new ListUserWorkspaces use case.
func NewListUserWorkspaces(lister WorkspaceMembershipLister) *ListUserWorkspaces {
	return &ListUserWorkspaces{lister: lister}
}

// Execute returns the user's workspace memberships.
func (uc *ListUserWorkspaces) Execute(ctx context.Context, userID uuid.UUID) ([]WorkspaceMembership, error) {
	memberships, err := uc.lister.ListForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing workspace memberships: %w", err)
	}

	return memberships, nil
}
