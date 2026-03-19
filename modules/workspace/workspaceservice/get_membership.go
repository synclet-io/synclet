package workspaceservice

import (
	"context"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// GetMembership returns a user's membership in a workspace.
type GetMembership struct {
	storage Storage
}

// NewGetMembership creates a new GetMembership use case.
func NewGetMembership(storage Storage) *GetMembership {
	return &GetMembership{storage: storage}
}

// Execute returns the membership for the given workspace and user.
func (uc *GetMembership) Execute(ctx context.Context, workspaceID, userID uuid.UUID) (*WorkspaceMember, error) {
	return uc.storage.WorkspaceMembers().First(ctx, &WorkspaceMemberFilter{
		WorkspaceID: filter.Equals(workspaceID),
		UserID:      filter.Equals(userID),
	})
}
