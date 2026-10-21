package workspaceservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// ApplyDefaultWorkspaceRole puts a user into the default workspace with a
// role decided outside the workspace module (an OIDC claim).
//
// A user who is not a member joins: as admin when the workspace has no
// members, otherwise with the given role; a join that loses a race with a
// concurrent join of the same user succeeds. An existing member's role changes
// only when syncExisting is set; the change is audited, and the workspace's
// last admin is never demoted.
type ApplyDefaultWorkspaceRole struct {
	storage Storage
	audit   AuditRecorder
	logger  *logging.Logger
}

// NewApplyDefaultWorkspaceRole creates a new ApplyDefaultWorkspaceRole use case.
func NewApplyDefaultWorkspaceRole(storage Storage, audit AuditRecorder, logger *logging.Logger) *ApplyDefaultWorkspaceRole {
	return &ApplyDefaultWorkspaceRole{
		storage: storage,
		audit:   audit,
		logger:  logger.Named("apply-default-workspace-role"),
	}
}

// Execute applies role to the user's membership in the default workspace.
// Returns ErrInvalidMemberRole when the role is not a valid MemberRole.
func (uc *ApplyDefaultWorkspaceRole) Execute(ctx context.Context, userID uuid.UUID, role MemberRole, syncExisting bool) error {
	if !role.IsValid() {
		return ErrInvalidMemberRole
	}

	workspace, err := uc.storage.Workspaces().First(ctx, &WorkspaceFilter{
		Slug: filter.Equals("default"),
	})
	if err != nil {
		return fmt.Errorf("finding default workspace: %w", err)
	}

	members, err := uc.storage.WorkspaceMembers().Find(ctx, &WorkspaceMemberFilter{
		WorkspaceID: filter.Equals(workspace.ID),
		UserID:      filter.Equals(userID),
	})
	if err != nil {
		return fmt.Errorf("loading membership: %w", err)
	}

	if len(members) == 0 {
		return uc.join(ctx, workspace.ID, userID, role)
	}

	member := members[0]
	if !syncExisting || member.Role == role {
		return nil
	}

	if member.Role == MemberRoleAdmin {
		adminCount, err := uc.storage.WorkspaceMembers().Count(ctx, &WorkspaceMemberFilter{
			WorkspaceID: filter.Equals(workspace.ID),
			Role:        filter.Equals(MemberRoleAdmin),
		})
		if err != nil {
			return fmt.Errorf("counting admins: %w", err)
		}

		if adminCount <= 1 {
			uc.logger.WithFields(map[string]interface{}{
				"user_id":        userID.String(),
				"requested_role": role.String(),
			}).Warn(ctx, "OIDC role claim would demote the last admin of the default workspace; role stays admin")

			return nil
		}
	}

	previousRole := member.Role

	member.Role = role
	if _, err := uc.storage.WorkspaceMembers().Update(ctx, member); err != nil {
		return fmt.Errorf("updating member role: %w", err)
	}

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:  workspace.ID,
		Action:       "update",
		ResourceType: "workspace_member",
		ResourceID:   member.ID,
		Before:       map[string]any{"role": previousRole.String(), "user_id": userID.String()},
		After:        map[string]any{"role": role.String(), "user_id": userID.String()},
	})

	return nil
}

func (uc *ApplyDefaultWorkspaceRole) join(ctx context.Context, workspaceID, userID uuid.UUID, role MemberRole) error {
	memberCount, err := uc.storage.WorkspaceMembers().Count(ctx, &WorkspaceMemberFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return fmt.Errorf("counting workspace members: %w", err)
	}

	if memberCount == 0 {
		role = MemberRoleAdmin
	}

	member := &WorkspaceMember{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
		JoinedAt:    time.Now(),
	}

	_, err = uc.storage.WorkspaceMembers().Create(ctx, member)
	if errors.Is(err, ErrWorkspaceMemberAlreadyExists) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("creating workspace member: %w", err)
	}

	return nil
}
