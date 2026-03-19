package workspaceservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// RemoveMember removes a user from a workspace.
type RemoveMember struct {
	storage Storage
}

// NewRemoveMember creates a new RemoveMember use case.
func NewRemoveMember(storage Storage) *RemoveMember {
	return &RemoveMember{storage: storage}
}

// Execute removes the member identified by workspaceID and userID.
func (uc *RemoveMember) Execute(ctx context.Context, workspaceID, userID uuid.UUID) error {
	if err := uc.storage.WorkspaceMembers().Delete(ctx, &WorkspaceMemberFilter{
		WorkspaceID: filter.Equals(workspaceID),
		UserID:      filter.Equals(userID),
	}); err != nil {
		return fmt.Errorf("removing member: %w", err)
	}

	return nil
}
