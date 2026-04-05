package pipelinetasks

import (
	"context"
	"fmt"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// FailStaleTasks times out stuck pending and orphaned running connector tasks.
type FailStaleTasks struct {
	storage        pipelineservice.Storage
	pendingTimeout time.Duration
	runningTimeout time.Duration
}

// NewFailStaleTasks creates a new FailStaleTasks use case.
func NewFailStaleTasks(storage pipelineservice.Storage, cfg pipelineservice.Config) *FailStaleTasks {
	return &FailStaleTasks{
		storage:        storage,
		pendingTimeout: cfg.ConnectorTaskPendingTimeout,
		runningTimeout: cfg.ConnectorTaskRunningTimeout,
	}
}

// Execute times out pending tasks waiting too long and orphaned running tasks.
func (uc *FailStaleTasks) Execute(ctx context.Context) error {
	// It's ok to have long-running transactions with locks, as this is a background task
	return uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		now := time.Now()

		// Timeout pending tasks that have been waiting too long.
		pendingCutoff := now.Add(-uc.pendingTimeout)

		stalePending, err := tx.ConnectorTasks().Find(ctx, &pipelineservice.ConnectorTaskFilter{
			Status:    filter.Equals(pipelineservice.ConnectorTaskStatusPending),
			CreatedAt: filter.Less(pendingCutoff),
		}, dbutil.WithForUpdate())
		if err != nil {
			return fmt.Errorf("finding stale pending tasks: %w", err)
		}

		for _, task := range stalePending {
			errMsg := "timed out waiting for executor"
			task.Status = pipelineservice.ConnectorTaskStatusFailed
			task.ErrorMessage = &errMsg
			task.CompletedAt = &now

			if _, err := tx.ConnectorTasks().Update(ctx, task); err != nil {
				return fmt.Errorf("timing out pending task %s: %w", task.ID, err)
			}
		}

		// Timeout orphaned running tasks whose executor did not report back.
		runningCutoff := now.Add(-uc.runningTimeout)

		staleRunning, err := tx.ConnectorTasks().Find(ctx, &pipelineservice.ConnectorTaskFilter{
			Status:    filter.Equals(pipelineservice.ConnectorTaskStatusRunning),
			UpdatedAt: filter.Less(runningCutoff),
		}, dbutil.WithForUpdate())
		if err != nil {
			return fmt.Errorf("finding stale running tasks: %w", err)
		}

		for _, task := range staleRunning {
			errMsg := "executor did not report result (orphaned)"
			task.Status = pipelineservice.ConnectorTaskStatusFailed
			task.ErrorMessage = &errMsg
			task.CompletedAt = &now

			if _, err := tx.ConnectorTasks().Update(ctx, task); err != nil {
				return fmt.Errorf("timing out running task %s: %w", task.ID, err)
			}
		}

		return nil
	})
}
