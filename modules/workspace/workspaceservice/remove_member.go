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
	audit   AuditRecorder
}

// NewRemoveMember creates a new RemoveMember use case.
func NewRemoveMember(storage Storage, audit AuditRecorder) *RemoveMember {
	return &RemoveMember{storage: storage, audit: audit}
}

// Execute removes the member identified by workspaceID and userID.
func (uc *RemoveMember) Execute(ctx context.Context, workspaceID, userID uuid.UUID) error {
	if err := uc.storage.WorkspaceMembers().Delete(ctx, &WorkspaceMemberFilter{
		WorkspaceID: filter.Equals(workspaceID),
		UserID:      filter.Equals(userID),
	}); err != nil {
		return fmt.Errorf("removing member: %w", err)
	}

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:  workspaceID,
		Action:       "delete",
		ResourceType: "workspace_member",
		ResourceID:   userID,
		Before:       map[string]any{"user_id": userID.String()},
	})

	return nil
}
