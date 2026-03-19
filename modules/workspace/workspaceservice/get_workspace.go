package workspaceservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// GetWorkspace retrieves a workspace by ID.
// Verifies the caller is a member of the workspace (D-11 dynamic ownership check).
type GetWorkspace struct {
	storage       Storage
	getMembership *GetMembership
}

// NewGetWorkspace creates a new GetWorkspace use case.
func NewGetWorkspace(storage Storage, getMembership *GetMembership) *GetWorkspace {
	return &GetWorkspace{storage: storage, getMembership: getMembership}
}

// Execute returns the workspace with the given ID after verifying membership.
// When userID is non-nil, verifies the caller is a member of the workspace (D-11).
// Internal/system callers may pass nil to skip the membership check.
func (uc *GetWorkspace) Execute(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*Workspace, error) {
	// D-11: Verify caller is a member of the requested workspace
	if userID != nil {
		_, err := uc.getMembership.Execute(ctx, id, *userID)
		if err != nil {
			return nil, fmt.Errorf("not a workspace member: %w", ErrWorkspaceNotFound)
		}
	}

	return uc.storage.Workspaces().First(ctx, &WorkspaceFilter{
		ID: filter.Equals(id),
	})
}
