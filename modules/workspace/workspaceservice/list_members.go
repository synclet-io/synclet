package workspaceservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// ListMembers returns all members of a workspace.
// Verifies the caller is a member of the workspace (D-11 dynamic ownership check).
type ListMembers struct {
	storage       Storage
	getMembership *GetMembership
}

// NewListMembers creates a new ListMembers use case.
func NewListMembers(storage Storage, getMembership *GetMembership) *ListMembers {
	return &ListMembers{storage: storage, getMembership: getMembership}
}

// Execute returns all members for the given workspace after verifying membership.
// When userID is non-nil, verifies the caller is a member of the workspace (D-11).
// Internal/system callers may pass nil to skip the membership check.
func (uc *ListMembers) Execute(ctx context.Context, workspaceID uuid.UUID, userID *uuid.UUID) ([]*WorkspaceMember, error) {
	// D-11: Verify caller is a member of the requested workspace
	if userID != nil {
		_, err := uc.getMembership.Execute(ctx, workspaceID, *userID)
		if err != nil {
			return nil, fmt.Errorf("not a workspace member: %w", ErrWorkspaceNotFound)
		}
	}

	return uc.storage.WorkspaceMembers().Find(ctx, &WorkspaceMemberFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
}
