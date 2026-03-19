package pipelinetasks

import (
	"context"
	"fmt"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// CleanupTasksConfig holds configuration for task cleanup.
type CleanupTasksConfig struct {
	RetentionPeriod time.Duration // Default 24h, min 1h (D-04)
	PendingTimeout  time.Duration // Default 5m (D-05)
	RunningTimeout  time.Duration // Default 10m (orphan recovery)
}

// CleanupTasks deletes old completed/failed tasks and times out stale pending/running tasks.
type CleanupTasks struct {
	storage pipelineservice.Storage
	config  CleanupTasksConfig
}

// NewCleanupTasks creates a new CleanupTasks use case.
// Returns an error if RetentionPeriod is less than 1 hour.
func NewCleanupTasks(storage pipelineservice.Storage, config CleanupTasksConfig) (*CleanupTasks, error) {
	if config.RetentionPeriod < time.Hour {
		return nil, fmt.Errorf("CONNECTOR_TASK_RETENTION must be >= 1h, got %s", config.RetentionPeriod)
	}
	return &CleanupTasks{storage: storage, config: config}, nil
}

// Execute performs all three cleanup operations in a single transaction:
// 1. Delete completed/failed tasks older than retention period (D-04)
// 2. Timeout pending tasks that have been waiting too long (D-05)
// 3. Timeout orphaned running tasks that lost their executor (Pitfall 4)
func (uc *CleanupTasks) Execute(ctx context.Context) error {
	return uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		now := time.Now()

		// 1. Delete completed/failed tasks older than retention period (D-04).
		retentionCutoff := now.Add(-uc.config.RetentionPeriod)
		if err := tx.ConnectorTasks().Delete(ctx, &pipelineservice.ConnectorTaskFilter{
			Status:      filter.In(pipelineservice.ConnectorTaskStatusCompleted, pipelineservice.ConnectorTaskStatusFailed),
			CompletedAt: filter.Less(&retentionCutoff),
		}); err != nil {
			return fmt.Errorf("deleting old tasks: %w", err)
		}

		// 2. Timeout pending tasks that have been waiting too long (D-05).
		pendingCutoff := now.Add(-uc.config.PendingTimeout)
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

		// 3. Timeout orphaned running tasks whose executor did not report back (Pitfall 4).
		runningCutoff := now.Add(-uc.config.RunningTimeout)
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
