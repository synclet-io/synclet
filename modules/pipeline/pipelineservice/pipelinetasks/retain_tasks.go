package pipelinetasks

import (
	"context"
	"fmt"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// RetainTasks deletes old completed/failed connector tasks beyond the retention period.
type RetainTasks struct {
	storage         pipelineservice.Storage
	retentionPeriod time.Duration
}

// NewRetainTasks creates a new RetainTasks use case.
// Returns an error if ConnectorTaskRetention is less than 1 hour.
func NewRetainTasks(storage pipelineservice.Storage, cfg pipelineservice.Config) (*RetainTasks, error) {
	if cfg.ConnectorTaskRetention < time.Hour {
		return nil, fmt.Errorf("CONNECTOR_TASK_RETENTION must be >= 1h, got %s", cfg.ConnectorTaskRetention)
	}

	return &RetainTasks{
		storage:         storage,
		retentionPeriod: cfg.ConnectorTaskRetention,
	}, nil
}

// Execute deletes completed/failed tasks older than the retention period.
func (uc *RetainTasks) Execute(ctx context.Context) error {
	return uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		retentionCutoff := time.Now().Add(-uc.retentionPeriod)

		if err := tx.ConnectorTasks().Delete(ctx, &pipelineservice.ConnectorTaskFilter{
			Status:      filter.In(pipelineservice.ConnectorTaskStatusCompleted, pipelineservice.ConnectorTaskStatusFailed),
			CompletedAt: filter.Less(&retentionCutoff),
		}); err != nil {
			return fmt.Errorf("deleting old tasks: %w", err)
		}

		return nil
	})
}
