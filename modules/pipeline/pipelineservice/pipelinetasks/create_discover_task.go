package pipelinetasks

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// CreateDiscoverTask creates a connector discover task for a source.
type CreateDiscoverTask struct {
	storage pipelineservice.Storage
}

// NewCreateDiscoverTask creates a new CreateDiscoverTask use case.
func NewCreateDiscoverTask(storage pipelineservice.Storage) *CreateDiscoverTask {
	return &CreateDiscoverTask{storage: storage}
}

// CreateDiscoverTaskParams holds parameters for creating a discover task.
type CreateDiscoverTaskParams struct {
	WorkspaceID uuid.UUID
	SourceID    uuid.UUID
}

// Execute creates a connector discover task and returns the task ID.
// If a pending or running discover task already exists for the same source, it returns that task instead.
func (uc *CreateDiscoverTask) Execute(ctx context.Context, params CreateDiscoverTaskParams) (*CreateTaskResult, error) {
	// Resolve the managed connector ID from the source, scoped to workspace.
	src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID:          filter.Equals(params.SourceID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("finding source: %w", err)
	}

	// Deduplicate: check for existing pending/running discover tasks for this source.
	existingTasks, err := uc.storage.ConnectorTasks().Find(ctx, &pipelineservice.ConnectorTaskFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
		TaskType:    filter.Equals(pipelineservice.ConnectorTaskTypeDiscover),
		Status:      filter.In(pipelineservice.ConnectorTaskStatusPending, pipelineservice.ConnectorTaskStatusRunning),
	})
	if err != nil {
		return nil, fmt.Errorf("checking existing discover tasks: %w", err)
	}

	for _, t := range existingTasks {
		if payload, ok := t.Payload.(*pipelineservice.DiscoverPayload); ok && payload.SourceID == params.SourceID {
			return &CreateTaskResult{TaskID: t.ID}, nil
		}
	}

	task := &pipelineservice.ConnectorTask{
		ID:          uuid.New(),
		WorkspaceID: params.WorkspaceID,
		TaskType:    pipelineservice.ConnectorTaskTypeDiscover,
		Status:      pipelineservice.ConnectorTaskStatusPending,
		Payload: &pipelineservice.DiscoverPayload{
			SourceID:           params.SourceID,
			ManagedConnectorID: src.ManagedConnectorID,
		},
	}

	if _, err := uc.storage.ConnectorTasks().Create(ctx, task); err != nil {
		return nil, fmt.Errorf("creating discover task: %w", err)
	}

	return &CreateTaskResult{TaskID: task.ID}, nil
}
