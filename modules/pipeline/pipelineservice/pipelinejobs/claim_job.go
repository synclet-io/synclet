package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/go-pnp/go-pnp/logging"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ClaimJobParams holds parameters for claiming a job.
type ClaimJobParams struct {
	WorkerID string
}

// ClaimJob atomically claims the next scheduled job for a worker.
// It wraps the ClaimNextScheduledJob storage method (FOR UPDATE SKIP LOCKED)
// as a use case that DockerSyncWorker and K8sSyncWorker inject.
type ClaimJob struct {
	storage pipelineservice.Storage
	logger  *logging.Logger
}

// NewClaimJob creates a new ClaimJob use case.
func NewClaimJob(storage pipelineservice.Storage, logger *logging.Logger) *ClaimJob {
	return &ClaimJob{storage: storage, logger: logger}
}

// Execute claims the next available scheduled job for the given worker.
// Returns nil, nil when no jobs are available (not an error).
func (uc *ClaimJob) Execute(ctx context.Context, params ClaimJobParams) (*pipelineservice.Job, error) {
	job, err := uc.storage.Jobs().ClaimNextScheduledJob(ctx, params.WorkerID)
	if err != nil {
		return nil, fmt.Errorf("claiming next scheduled job: %w", err)
	}

	if job == nil {
		return nil, nil
	}

	if uc.logger != nil {
		uc.logger.WithFields(map[string]interface{}{"job_id": job.ID.String(), "connection_id": job.ConnectionID.String()}).Info(ctx, "claimed job")
	}

	return job, nil
}
