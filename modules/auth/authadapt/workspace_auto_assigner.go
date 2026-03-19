package authadapt

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
)

// WorkspaceAutoAssigner adapts workspaceservice.AutoAssignMember to authconnect.WorkspaceAutoAssigner.
type WorkspaceAutoAssigner struct {
	autoAssignMember *workspaceservice.AutoAssignMember
}

// NewWorkspaceAutoAssigner creates a new WorkspaceAutoAssigner adapter.
func NewWorkspaceAutoAssigner(autoAssignMember *workspaceservice.AutoAssignMember) *WorkspaceAutoAssigner {
	return &WorkspaceAutoAssigner{autoAssignMember: autoAssignMember}
}

// AutoAssign implements authconnect.WorkspaceAutoAssigner.
func (a *WorkspaceAutoAssigner) AutoAssign(ctx context.Context, userID uuid.UUID) error {
	return a.autoAssignMember.Execute(ctx, userID)
}
