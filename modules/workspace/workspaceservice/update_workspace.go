package workspaceservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// UpdateWorkspaceParams holds the parameters for updating a workspace.
type UpdateWorkspaceParams struct {
	ID   uuid.UUID
	Name *string
}

// UpdateWorkspace updates a workspace's settings.
type UpdateWorkspace struct {
	storage Storage
}

// NewUpdateWorkspace creates a new UpdateWorkspace use case.
func NewUpdateWorkspace(storage Storage) *UpdateWorkspace {
	return &UpdateWorkspace{storage: storage}
}

// Execute updates the workspace fields specified in params.
func (uc *UpdateWorkspace) Execute(ctx context.Context, params UpdateWorkspaceParams) (*Workspace, error) {
	ws, err := uc.storage.Workspaces().First(ctx, &WorkspaceFilter{
		ID: filter.Equals(params.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting workspace: %w", err)
	}

	if params.Name != nil {
		ws.Name = *params.Name
	}

	ws.UpdatedAt = time.Now()

	updated, err := uc.storage.Workspaces().Update(ctx, ws)
	if err != nil {
		return nil, fmt.Errorf("updating workspace: %w", err)
	}

	return updated, nil
}
