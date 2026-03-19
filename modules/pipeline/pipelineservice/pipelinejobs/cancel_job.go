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

// K8sSyncStopper stops a K8s sync job. Optional dependency -- nil when running in Docker mode.
type K8sSyncStopper interface {
	StopSync(ctx context.Context, jobName string) error
}

// CancelJob marks a pending or running job as cancelled.
// For K8s jobs, it also stops the K8s job directly.
// For Docker jobs, the running goroutine's pollForCancel detects the status change
// and cancels the execution context, triggering container shutdown via StopWithTimeout.
type CancelJob struct {
	storage        pipelineservice.Storage
	k8sSyncStopper K8sSyncStopper // nil in Docker mode
	logger         *logging.Logger
}

// NewCancelJob creates a new CancelJob use case.
func NewCancelJob(storage pipelineservice.Storage, k8sSyncStopper K8sSyncStopper, logger *logging.Logger) *CancelJob {
	return &CancelJob{storage: storage, k8sSyncStopper: k8sSyncStopper, logger: logger.Named("cancel-job")}
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

	// For K8s jobs, stop the K8s job directly.
	if job.K8sJobName != nil && *job.K8sJobName != "" && uc.k8sSyncStopper != nil {
		if err := uc.k8sSyncStopper.StopSync(ctx, *job.K8sJobName); err != nil {
			// Log but continue -- still mark as cancelled in DB.
			// The reconciler will clean up if StopSync fails.
			uc.logger.WithError(err).Warn(ctx, "failed to stop K8s job during cancel",
				"job_id", params.ID.String(),
				"k8s_job_name", *job.K8sJobName)
		}
	}

	// For Docker jobs: Setting status to Cancelled in DB is sufficient.
	// The running goroutine's pollForCancel will detect this status change,
	// cancel the execution context, and the deferred cleanup functions in
	// the connector client will call StopWithTimeout(30) on the containers
	// (SIGTERM -> wait 30s -> SIGKILL).

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
