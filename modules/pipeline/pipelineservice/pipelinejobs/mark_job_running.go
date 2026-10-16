package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// MarkJobRunningParams holds parameters for marking a job as running.
type MarkJobRunningParams struct {
	ID uuid.UUID
}

// MarkJobRunning moves a job from starting to running once its connectors have started.
type MarkJobRunning struct {
	storage pipelineservice.Storage
}

// NewMarkJobRunning creates a new MarkJobRunning use case.
func NewMarkJobRunning(storage pipelineservice.Storage) *MarkJobRunning {
	return &MarkJobRunning{storage: storage}
}

// Execute sets the job to running and publishes a JobUpdated event in the same
// transaction. Only a starting job changes; any other status is a no-op.
func (uc *MarkJobRunning) Execute(ctx context.Context, params MarkJobRunningParams) error {
	prevJob, err := uc.storage.Jobs().First(ctx, &pipelineservice.JobFilter{
		ID: filter.Equals(params.ID),
	})
	if err != nil {
		return fmt.Errorf("getting job: %w", err)
	}

	if prevJob.Status != pipelineservice.JobStatusStarting {
		return nil
	}

	oldJob := *prevJob
	newJob := *prevJob
	newJob.Status = pipelineservice.JobStatusRunning

	return uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		updated, err := tx.Jobs().MarkRunning(ctx, params.ID)
		if err != nil {
			return fmt.Errorf("updating job: %w", err)
		}

		if !updated {
			return nil
		}

		conn, err := tx.Connections().First(ctx, &pipelineservice.ConnectionFilter{
			ID: filter.Equals(newJob.ConnectionID),
		})
		if err != nil {
			return fmt.Errorf("getting connection for event emission: %w", err)
		}

		if err := tx.PipelineEvents().Send(ctx, &pipelineservice.PipelineEvent{
			EventType: pipelineservice.PipelineEventTypeJobUpdated,
			Data: &pipelineservice.JobUpdatedEventData{
				WorkspaceID: conn.WorkspaceID,
				OldData:     &oldJob,
				NewData:     &newJob,
			},
		}); err != nil {
			return fmt.Errorf("publishing job.updated event: %w", err)
		}

		return nil
	})
}
