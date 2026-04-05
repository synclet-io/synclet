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
func (uc *CancelJob) Execute(ctx context.Context, params CancelJobParams) error {
	job, err := uc.storage.Jobs().First(ctx, &pipelineservice.JobFilter{
		ID: filter.Equals(params.ID),
	})
	if err != nil {
		return fmt.Errorf("getting job: %w", err)
	}

	if job.Status != pipelineservice.JobStatusScheduled && job.Status != pipelineservice.JobStatusStarting && job.Status != pipelineservice.JobStatusRunning {
		return &pipelineservice.ValidationError{Message: fmt.Sprintf("cannot cancel job with status %s", job.Status)}
	}

	job.Status = pipelineservice.JobStatusCancelled
	now := time.Now()
	job.CompletedAt = &now
	reason := "cancelled by user"
	job.FailureReason = &reason

	if _, err := uc.storage.Jobs().Update(ctx, job); err != nil {
		return fmt.Errorf("updating job: %w", err)
	}

	return nil
}
