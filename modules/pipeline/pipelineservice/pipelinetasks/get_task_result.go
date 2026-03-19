package pipelinetasks

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetTaskResult retrieves the current status and result of a connector task.
type GetTaskResult struct {
	storage pipelineservice.Storage
}

// NewGetTaskResult creates a new GetTaskResult use case.
func NewGetTaskResult(storage pipelineservice.Storage) *GetTaskResult {
	return &GetTaskResult{storage: storage}
}

// GetTaskResultParams holds parameters for getting a task result.
type GetTaskResultParams struct {
	TaskID      uuid.UUID
	WorkspaceID uuid.UUID // Workspace scoping per Pitfall 6
}

// GetTaskResultResult holds the task status and typed result.
type GetTaskResultResult struct {
	Status       pipelineservice.ConnectorTaskStatus
	TaskType     pipelineservice.ConnectorTaskType
	ErrorMessage string
	Result       pipelineservice.ConnectorTaskResult // one_of interface (nil if not completed)
}

// Execute retrieves a connector task's status and result, scoped to workspace.
func (uc *GetTaskResult) Execute(ctx context.Context, params GetTaskResultParams) (*GetTaskResultResult, error) {
	task, err := uc.storage.ConnectorTasks().First(ctx, &pipelineservice.ConnectorTaskFilter{
		ID:          filter.Equals(params.TaskID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("finding task: %w", err)
	}

	var errMsg string
	if task.ErrorMessage != nil {
		errMsg = *task.ErrorMessage
	}

	var result pipelineservice.ConnectorTaskResult
	if task.Result != nil {
		result = *task.Result
	}

	return &GetTaskResultResult{
		Status:       task.Status,
		TaskType:     task.TaskType,
		ErrorMessage: errMsg,
		Result:       result,
	}, nil
}
