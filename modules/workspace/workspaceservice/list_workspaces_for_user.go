package workspaceservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"go.uber.org/zap"
)

// ListWorkspacesForUser returns all workspaces a user is a member of, along
// with the user's role in each.
type ListWorkspacesForUser struct {
	storage Storage
}

// UserWorkspace couples a workspace with the caller's role in it.
type UserWorkspace struct {
	Workspace *Workspace
	Role      MemberRole
}

// NewListWorkspacesForUser creates a new ListWorkspacesForUser use case.
func NewListWorkspacesForUser(storage Storage) *ListWorkspacesForUser {
	return &ListWorkspacesForUser{storage: storage}
}

// Execute returns all workspaces for the given user.
func (uc *ListWorkspacesForUser) Execute(ctx context.Context, userID uuid.UUID) ([]UserWorkspace, error) {
	members, err := uc.storage.WorkspaceMembers().Find(ctx, &WorkspaceMemberFilter{
		UserID: filter.Equals(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing memberships: %w", err)
	}

	out := make([]UserWorkspace, 0, len(members))
	for _, member := range members {
		workspace, err := uc.storage.Workspaces().First(ctx, &WorkspaceFilter{
			ID: filter.Equals(member.WorkspaceID),
		})
		if err != nil {
			zap.L().Warn("failed to load workspace", zap.String("workspace_id", member.WorkspaceID.String()), zap.Error(err))

			continue
		}

		out = append(out, UserWorkspace{Workspace: workspace, Role: member.Role})
	}

	return out, nil
}
