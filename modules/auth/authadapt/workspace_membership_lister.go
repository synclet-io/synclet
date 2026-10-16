package authadapt

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
)

// WorkspaceMembershipLister adapts workspaceservice.ListWorkspacesForUser to authservice.WorkspaceMembershipLister.
type WorkspaceMembershipLister struct {
	listWorkspacesForUser *workspaceservice.ListWorkspacesForUser
}

// NewWorkspaceMembershipLister creates a new WorkspaceMembershipLister adapter.
func NewWorkspaceMembershipLister(listWorkspacesForUser *workspaceservice.ListWorkspacesForUser) *WorkspaceMembershipLister {
	return &WorkspaceMembershipLister{listWorkspacesForUser: listWorkspacesForUser}
}

// ListForUser implements authservice.WorkspaceMembershipLister.
func (a *WorkspaceMembershipLister) ListForUser(ctx context.Context, userID uuid.UUID) ([]authservice.WorkspaceMembership, error) {
	workspaces, err := a.listWorkspacesForUser.Execute(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := make([]authservice.WorkspaceMembership, len(workspaces))
	for i, uw := range workspaces {
		out[i] = authservice.WorkspaceMembership{
			WorkspaceID:   uw.Workspace.ID,
			WorkspaceName: uw.Workspace.Name,
			Role:          workspaceRole(uw.Role),
		}
	}

	return out, nil
}

func workspaceRole(r workspaceservice.MemberRole) authservice.WorkspaceRole {
	switch r {
	case workspaceservice.MemberRoleAdmin:
		return authservice.WorkspaceRoleAdmin
	case workspaceservice.MemberRoleEditor:
		return authservice.WorkspaceRoleEditor
	case workspaceservice.MemberRoleViewer:
		return authservice.WorkspaceRoleViewer
	default:
		return authservice.WorkspaceRoleUnspecified
	}
}
