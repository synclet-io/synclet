package pipelinetasks

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// CreateSpecTask creates a connector spec task for async spec fetching.
// Used for custom connectors that need their spec retrieved asynchronously (D-09).
type CreateSpecTask struct {
	storage pipelineservice.Storage
}

// NewCreateSpecTask creates a new CreateSpecTask use case.
func NewCreateSpecTask(storage pipelineservice.Storage) *CreateSpecTask {
	return &CreateSpecTask{storage: storage}
}

// CreateSpecTaskParams holds parameters for creating a spec task.
type CreateSpecTaskParams struct {
	WorkspaceID        uuid.UUID
	ManagedConnectorID uuid.UUID
}

// Execute creates a connector spec task and returns the task ID.
func (uc *CreateSpecTask) Execute(ctx context.Context, params CreateSpecTaskParams) (*CreateTaskResult, error) {
	// Verify the managed connector exists and is scoped to the workspace.
	_, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(params.ManagedConnectorID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("finding managed connector: %w", err)
	}

	task := &pipelineservice.ConnectorTask{
		ID:          uuid.New(),
		WorkspaceID: params.WorkspaceID,
		TaskType:    pipelineservice.ConnectorTaskTypeSpec,
		Status:      pipelineservice.ConnectorTaskStatusPending,
		Payload: &pipelineservice.SpecPayload{
			ManagedConnectorID: params.ManagedConnectorID,
		},
	}

	if _, err := uc.storage.ConnectorTasks().Create(ctx, task); err != nil {
		return nil, fmt.Errorf("creating spec task: %w", err)
	}

	return &CreateTaskResult{TaskID: task.ID}, nil
}
