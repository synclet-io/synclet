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

// Execute removes the member identified by workspaceID and userID. It returns
// a not-found error for a non-member and refuses to remove the last admin.
func (uc *RemoveMember) Execute(ctx context.Context, workspaceID, userID uuid.UUID) error {
	member, err := uc.storage.WorkspaceMembers().First(ctx, &WorkspaceMemberFilter{
		WorkspaceID: filter.Equals(workspaceID),
		UserID:      filter.Equals(userID),
	})
	if err != nil {
		return fmt.Errorf("loading membership: %w", err)
	}

	if member.Role == MemberRoleAdmin {
		adminCount, err := uc.storage.WorkspaceMembers().Count(ctx, &WorkspaceMemberFilter{
			WorkspaceID: filter.Equals(workspaceID),
			Role:        filter.Equals(MemberRoleAdmin),
		})
		if err != nil {
			return fmt.Errorf("counting admins: %w", err)
		}

		if adminCount <= 1 {
			return ErrLastAdminCannotBeRemoved
		}
	}

	if err := uc.storage.WorkspaceMembers().Delete(ctx, &WorkspaceMemberFilter{
		ID:          filter.Equals(member.ID),
		WorkspaceID: filter.Equals(workspaceID),
	}); err != nil {
		return fmt.Errorf("removing member: %w", err)
	}

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:  workspaceID,
		Action:       "delete",
		ResourceType: "workspace_member",
		ResourceID:   userID,
		Before:       map[string]any{"user_id": userID.String(), "role": member.Role.String()},
	})

	return nil
}
