package pipelinetasks

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// WaitForTaskResult polls a connector task until it reaches a terminal state (completed or failed).
type WaitForTaskResult struct {
	storage      pipelineservice.Storage
	pollInterval time.Duration
	timeout      time.Duration
}

// NewWaitForTaskResult creates a new WaitForTaskResult use case.
func NewWaitForTaskResult(storage pipelineservice.Storage) *WaitForTaskResult {
	return &WaitForTaskResult{
		storage:      storage,
		pollInterval: 500 * time.Millisecond,
		timeout:      5 * time.Minute,
	}
}

// WaitForTaskResultParams holds parameters for waiting on a task result.
type WaitForTaskResultParams struct {
	TaskID      uuid.UUID
	WorkspaceID uuid.UUID
}

// WaitForTaskResultResult holds the completed task's status and typed result.
type WaitForTaskResultResult struct {
	Status       pipelineservice.ConnectorTaskStatus
	TaskType     pipelineservice.ConnectorTaskType
	ErrorMessage string
	Result       pipelineservice.ConnectorTaskResult
}

// Execute polls the task until it completes or fails, respecting context cancellation.
func (uc *WaitForTaskResult) Execute(ctx context.Context, params WaitForTaskResultParams) (*WaitForTaskResultResult, error) {
	timer := time.NewTimer(uc.timeout)
	defer timer.Stop()

	ticker := time.NewTicker(uc.pollInterval)
	defer ticker.Stop()

	for {
		task, err := uc.storage.ConnectorTasks().First(ctx, &pipelineservice.ConnectorTaskFilter{
			ID:          filter.Equals(params.TaskID),
			WorkspaceID: filter.Equals(params.WorkspaceID),
		})
		if err != nil {
			return nil, fmt.Errorf("finding task: %w", err)
		}

		if task.Status == pipelineservice.ConnectorTaskStatusCompleted || task.Status == pipelineservice.ConnectorTaskStatusFailed {
			var errMsg string
			if task.ErrorMessage != nil {
				errMsg = *task.ErrorMessage
			}

			var result pipelineservice.ConnectorTaskResult
			if task.Result != nil {
				result = *task.Result
			}

			return &WaitForTaskResultResult{
				Status:       task.Status,
				TaskType:     task.TaskType,
				ErrorMessage: errMsg,
				Result:       result,
			}, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled while waiting for task %s: %w", params.TaskID, ctx.Err())
		case <-timer.C:
			return nil, fmt.Errorf("timeout waiting for task %s to complete", params.TaskID)
		case <-ticker.C:
			// Continue polling.
		}
	}
}
