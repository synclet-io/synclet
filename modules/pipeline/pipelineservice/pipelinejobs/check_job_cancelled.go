package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// CheckJobCancelledParams holds parameters for checking job cancellation.
type CheckJobCancelledParams struct {
	JobID uuid.UUID
}

// CheckJobCancelled checks whether a job has been cancelled.
// Used by Heartbeat handler instead of direct storage access.
type CheckJobCancelled struct {
	storage pipelineservice.Storage
}

// NewCheckJobCancelled creates a new CheckJobCancelled use case.
func NewCheckJobCancelled(storage pipelineservice.Storage) *CheckJobCancelled {
	return &CheckJobCancelled{storage: storage}
}

// Execute returns true if the job status is Cancelled.
func (uc *CheckJobCancelled) Execute(ctx context.Context, params CheckJobCancelledParams) (bool, error) {
	job, err := uc.storage.Jobs().First(ctx, &pipelineservice.JobFilter{
		ID: filter.Equals(params.JobID),
	})
	if err != nil {
		return false, fmt.Errorf("checking job cancelled: %w", err)
	}

	return job.Status == pipelineservice.JobStatusCancelled, nil
}
