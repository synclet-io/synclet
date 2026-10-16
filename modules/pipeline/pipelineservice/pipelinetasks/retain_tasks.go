package pipelinetasks

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ownsSecrets reports whether the task stored encrypted config secrets under
// its own ID, which only direct-config check tasks do.
func ownsSecrets(task *pipelineservice.ConnectorTask) bool {
	payload, ok := task.Payload.(*pipelineservice.CheckPayload)

	return ok && payload.Config != nil
}

// RetainTasks deletes old completed/failed connector tasks beyond the retention period.
type RetainTasks struct {
	storage         pipelineservice.Storage
	secrets         pipelineservice.SecretsProvider
	retentionPeriod time.Duration
}

// NewRetainTasks creates a new RetainTasks use case.
// Returns an error if ConnectorTaskRetention is less than 1 hour.
func NewRetainTasks(storage pipelineservice.Storage, secrets pipelineservice.SecretsProvider, cfg pipelineservice.Config) (*RetainTasks, error) {
	if cfg.ConnectorTaskRetention < time.Hour {
		return nil, fmt.Errorf("CONNECTOR_TASK_RETENTION must be >= 1h, got %s", cfg.ConnectorTaskRetention)
	}

	return &RetainTasks{
		storage:         storage,
		secrets:         secrets,
		retentionPeriod: cfg.ConnectorTaskRetention,
	}, nil
}

// Execute deletes completed/failed tasks older than the retention period
// together with any config secrets they own. Secrets go first, so a task row
// is only removed once nothing it owns is left behind.
func (uc *RetainTasks) Execute(ctx context.Context) error {
	retentionCutoff := time.Now().Add(-uc.retentionPeriod)

	expired, err := uc.storage.ConnectorTasks().Find(ctx, &pipelineservice.ConnectorTaskFilter{
		Status:      filter.In(pipelineservice.ConnectorTaskStatusCompleted, pipelineservice.ConnectorTaskStatusFailed),
		CompletedAt: filter.Less(&retentionCutoff),
	})
	if err != nil {
		return fmt.Errorf("finding old tasks: %w", err)
	}

	if len(expired) == 0 {
		return nil
	}

	for _, task := range expired {
		if !ownsSecrets(task) {
			continue
		}

		if err := uc.secrets.DeleteByOwner(ctx, pipelineservice.SecretOwnerConnectorTask, task.ID); err != nil {
			return fmt.Errorf("deleting secrets of task %s: %w", task.ID, err)
		}
	}

	ids := make([]uuid.UUID, len(expired))
	for i, task := range expired {
		ids[i] = task.ID
	}

	if err := uc.storage.ConnectorTasks().Delete(ctx, &pipelineservice.ConnectorTaskFilter{
		ID: filter.In(ids...),
	}); err != nil {
		return fmt.Errorf("deleting old tasks: %w", err)
	}

	return nil
}
