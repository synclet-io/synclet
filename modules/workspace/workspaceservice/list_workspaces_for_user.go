package workspaceservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"go.uber.org/zap"
)

// ListWorkspacesForUser returns all workspaces a user is a member of.
type ListWorkspacesForUser struct {
	storage Storage
}

// NewListWorkspacesForUser creates a new ListWorkspacesForUser use case.
func NewListWorkspacesForUser(storage Storage) *ListWorkspacesForUser {
	return &ListWorkspacesForUser{storage: storage}
}

// Execute returns all workspaces for the given user.
func (uc *ListWorkspacesForUser) Execute(ctx context.Context, userID uuid.UUID) ([]*Workspace, error) {
	members, err := uc.storage.WorkspaceMembers().Find(ctx, &WorkspaceMemberFilter{
		UserID: filter.Equals(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing memberships: %w", err)
	}

	workspaces := make([]*Workspace, 0, len(members))
	for _, m := range members {
		ws, err := uc.storage.Workspaces().First(ctx, &WorkspaceFilter{
			ID: filter.Equals(m.WorkspaceID),
		})
		if err != nil {
			zap.L().Warn("failed to load workspace", zap.String("workspace_id", m.WorkspaceID.String()), zap.Error(err))
			continue
		}
		workspaces = append(workspaces, ws)
	}

	return workspaces, nil
}
