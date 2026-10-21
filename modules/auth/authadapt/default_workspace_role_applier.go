package authadapt

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
)

// DefaultWorkspaceRoleApplier adapts workspaceservice.ApplyDefaultWorkspaceRole
// to authservice.DefaultWorkspaceRoleApplier.
type DefaultWorkspaceRoleApplier struct {
	applyDefaultWorkspaceRole *workspaceservice.ApplyDefaultWorkspaceRole
}

// NewDefaultWorkspaceRoleApplier creates a new DefaultWorkspaceRoleApplier adapter.
func NewDefaultWorkspaceRoleApplier(applyDefaultWorkspaceRole *workspaceservice.ApplyDefaultWorkspaceRole) *DefaultWorkspaceRoleApplier {
	return &DefaultWorkspaceRoleApplier{applyDefaultWorkspaceRole: applyDefaultWorkspaceRole}
}

// Apply implements authservice.DefaultWorkspaceRoleApplier.
func (a *DefaultWorkspaceRoleApplier) Apply(ctx context.Context, userID uuid.UUID, role string, syncExisting bool) error {
	memberRole, err := memberRoleFromString(role)
	if err != nil {
		return err
	}

	return a.applyDefaultWorkspaceRole.Execute(ctx, userID, memberRole, syncExisting)
}

func memberRoleFromString(role string) (workspaceservice.MemberRole, error) {
	switch role {
	case "admin":
		return workspaceservice.MemberRoleAdmin, nil
	case "editor":
		return workspaceservice.MemberRoleEditor, nil
	case "viewer":
		return workspaceservice.MemberRoleViewer, nil
	default:
		return 0, fmt.Errorf("unknown workspace role %q: %w", role, workspaceservice.ErrInvalidMemberRole)
	}
}
