package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// IsTaskActiveParams holds parameters for checking if a connector task is active.
type IsTaskActiveParams struct {
	ID uuid.UUID
}

// IsTaskActive checks whether a connector task is in pending or running status.
type IsTaskActive struct {
	storage pipelineservice.Storage
}

// NewIsTaskActive creates a new IsTaskActive use case.
func NewIsTaskActive(storage pipelineservice.Storage) *IsTaskActive {
	return &IsTaskActive{storage: storage}
}

// Execute returns true if the task exists and has pending or running status.
// Returns false with no error if the task is not found.
func (uc *IsTaskActive) Execute(ctx context.Context, params IsTaskActiveParams) (bool, error) {
	task, err := uc.storage.ConnectorTasks().First(ctx, &pipelineservice.ConnectorTaskFilter{
		ID: filter.Equals(params.ID),
	})
	if err != nil {
		return false, nil //nolint:nilerr // not-found is expected, return not active
	}

	return task.Status == pipelineservice.ConnectorTaskStatusPending ||
		task.Status == pipelineservice.ConnectorTaskStatusRunning, nil
}

// ExecuteByString parses a string task ID and checks if the task is active.
// Convenience method for adapters that receive string IDs.
func (uc *IsTaskActive) ExecuteByString(ctx context.Context, taskID string) (bool, error) {
	id, err := uuid.Parse(taskID)
	if err != nil {
		return false, fmt.Errorf("parsing task ID: %w", err)
	}

	return uc.Execute(ctx, IsTaskActiveParams{ID: id})
}
