package pipelinejobs

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// CancelJobParams holds parameters for cancelling a job.
type CancelJobParams struct {
	ID uuid.UUID
}

// CancelJob marks a pending or running job as cancelled.
// The executor's heartbeat loop detects the status change and triggers graceful shutdown
// (works uniformly for both Docker and K8s modes).
type CancelJob struct {
	storage pipelineservice.Storage
	logger  *logging.Logger
}

// NewCancelJob creates a new CancelJob use case.
func NewCancelJob(storage pipelineservice.Storage, logger *logging.Logger) *CancelJob {
	return &CancelJob{storage: storage, logger: logger.Named("cancel-job")}
}

// Execute cancels the job with the given ID. Only pending or running jobs can be cancelled.
// Publishes a JobUpdated event in the same transaction.
func (uc *CancelJob) Execute(ctx context.Context, params CancelJobParams) error {
	prev, err := uc.storage.Jobs().First(ctx, &pipelineservice.JobFilter{
		ID: filter.Equals(params.ID),
	})
	if err != nil {
		return fmt.Errorf("getting job: %w", err)
	}

	if prev.Status != pipelineservice.JobStatusScheduled && prev.Status != pipelineservice.JobStatusStarting && prev.Status != pipelineservice.JobStatusRunning {
		return &pipelineservice.ValidationError{Message: fmt.Sprintf("cannot cancel job with status %s", prev.Status)}
	}

	oldJob := *prev
	job := prev

	job.Status = pipelineservice.JobStatusCancelled
	now := time.Now()
	job.CompletedAt = &now
	reason := "cancelled by user"
	job.FailureReason = &reason

	return uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		updated, err := tx.Jobs().UpdateIfStatus(ctx, job,
			pipelineservice.JobStatusScheduled, pipelineservice.JobStatusStarting, pipelineservice.JobStatusRunning)
		if err != nil {
			return fmt.Errorf("updating job: %w", err)
		}

		if !updated {
			return &pipelineservice.ValidationError{Message: "job has already finished"}
		}

		conn, err := tx.Connections().First(ctx, &pipelineservice.ConnectionFilter{
			ID: filter.Equals(job.ConnectionID),
		})
		if err != nil {
			return fmt.Errorf("loading connection for cancel event: %w", err)
		}

		newJob := *job
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
