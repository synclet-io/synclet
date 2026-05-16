package workspaceservice

import (
	"context"
	"errors"
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

// Execute deletes the workspace with the given ID and publishes the
// workspace.deleted event for downstream subscribers. The pre-delete snapshot
// is included in the event so consumers can tear down derived state without
// re-querying.
func (uc *DeleteWorkspace) Execute(ctx context.Context, id uuid.UUID) error {
	workspace, err := uc.storage.Workspaces().First(ctx, &WorkspaceFilter{
		ID: filter.Equals(id),
	})
	if err != nil {
		// Already gone — treat as a no-op so callers stay idempotent.
		var notFound NotFoundError
		if errors.As(err, &notFound) {
			return nil
		}

		return fmt.Errorf("loading workspace: %w", err)
	}

	if err := uc.storage.Workspaces().Delete(ctx, &WorkspaceFilter{
		ID: filter.Equals(id),
	}); err != nil {
		return fmt.Errorf("deleting workspace: %w", err)
	}

	if err := uc.storage.WorkspaceEvents().Send(ctx, &WorkspaceEvent{
		EventType: WorkspaceEventTypeDeleted,
		Data:      &WorkspaceDeletedEventData{Workspace: workspace},
	}); err != nil {
		return fmt.Errorf("publishing workspace.deleted event: %w", err)
	}

	return nil
}
