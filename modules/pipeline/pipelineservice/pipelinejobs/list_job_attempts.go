package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListJobAttemptsParams holds parameters for listing job attempts.
type ListJobAttemptsParams struct {
	JobID uuid.UUID
}

// ListJobAttempts retrieves all attempts for a job.
type ListJobAttempts struct {
	storage pipelineservice.Storage
}

// NewListJobAttempts creates a new ListJobAttempts use case.
func NewListJobAttempts(storage pipelineservice.Storage) *ListJobAttempts {
	return &ListJobAttempts{storage: storage}
}

// Execute returns all attempts for the given job.
func (uc *ListJobAttempts) Execute(ctx context.Context, params ListJobAttemptsParams) ([]*pipelineservice.JobAttempt, error) {
	attempts, err := uc.storage.JobAttempts().Find(ctx, &pipelineservice.JobAttemptFilter{
		JobID: filter.Equals(params.JobID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing job attempts: %w", err)
	}

	return attempts, nil
}
