package workspaceservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// UpdateMemberRole updates a workspace member's role.
type UpdateMemberRole struct {
	storage Storage
}

// NewUpdateMemberRole creates a new UpdateMemberRole use case.
func NewUpdateMemberRole(storage Storage) *UpdateMemberRole {
	return &UpdateMemberRole{storage: storage}
}

// Execute updates the role of the member identified by workspaceID and userID.
// Returns ErrInvalidMemberRole when the role is not a valid MemberRole.
// Returns ErrWorkspaceMemberNotFound when the user is not a member of the workspace.
// Returns ErrLastAdminCannotBeDemoted when the target is the only admin and the
// requested role is not Admin.
func (uc *UpdateMemberRole) Execute(ctx context.Context, workspaceID, userID uuid.UUID, role MemberRole) error {
	if !role.IsValid() {
		return ErrInvalidMemberRole
	}

	member, err := uc.storage.WorkspaceMembers().First(ctx, &WorkspaceMemberFilter{
		WorkspaceID: filter.Equals(workspaceID),
		UserID:      filter.Equals(userID),
	})
	if err != nil {
		return fmt.Errorf("loading membership: %w", err)
	}

	// No-op when the role is unchanged.
	if member.Role == role {
		return nil
	}

	// If the target is the only admin and the new role is not Admin, block.
	if member.Role == MemberRoleAdmin && role != MemberRoleAdmin {
		adminCount, err := uc.storage.WorkspaceMembers().Count(ctx, &WorkspaceMemberFilter{
			WorkspaceID: filter.Equals(workspaceID),
			Role:        filter.Equals(MemberRoleAdmin),
		})
		if err != nil {
			return fmt.Errorf("counting admins: %w", err)
		}

		if adminCount <= 1 {
			return ErrLastAdminCannotBeDemoted
		}
	}

	member.Role = role
	if _, err := uc.storage.WorkspaceMembers().Update(ctx, member); err != nil {
		return fmt.Errorf("updating member role: %w", err)
	}

	return nil
}
