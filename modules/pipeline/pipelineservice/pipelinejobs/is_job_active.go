package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// IsJobActiveParams holds parameters for checking if a job is active.
type IsJobActiveParams struct {
	ID uuid.UUID
}

// IsJobActive checks whether a job is in running status.
type IsJobActive struct {
	storage pipelineservice.Storage
}

// NewIsJobActive creates a new IsJobActive use case.
func NewIsJobActive(storage pipelineservice.Storage) *IsJobActive {
	return &IsJobActive{storage: storage}
}

// Execute returns true if the job exists and has running status.
// Returns false with no error if the job is not found.
func (uc *IsJobActive) Execute(ctx context.Context, params IsJobActiveParams) (bool, error) {
	job, err := uc.storage.Jobs().First(ctx, &pipelineservice.JobFilter{
		ID: filter.Equals(params.ID),
	})
	if err != nil {
		return false, nil //nolint:nilerr // not-found is expected, return not active
	}

	return job.Status == pipelineservice.JobStatusRunning, nil
}

// ExecuteByString parses a string job ID and checks if the job is active.
// Convenience method for adapters that receive string IDs.
func (uc *IsJobActive) ExecuteByString(ctx context.Context, jobID string) (bool, error) {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return false, fmt.Errorf("parsing job ID: %w", err)
	}

	return uc.Execute(ctx, IsJobActiveParams{ID: id})
}
