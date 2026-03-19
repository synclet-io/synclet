package workspaceservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// DeleteWorkspace deletes a workspace by ID.
type DeleteWorkspace struct {
	storage Storage
}

// NewDeleteWorkspace creates a new DeleteWorkspace use case.
func NewDeleteWorkspace(storage Storage) *DeleteWorkspace {
	return &DeleteWorkspace{storage: storage}
}

// Execute deletes the workspace with the given ID.
func (uc *DeleteWorkspace) Execute(ctx context.Context, id uuid.UUID) error {
	if err := uc.storage.Workspaces().Delete(ctx, &WorkspaceFilter{
		ID: filter.Equals(id),
	}); err != nil {
		return fmt.Errorf("deleting workspace: %w", err)
	}

	return nil
}
