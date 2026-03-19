package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetJobParams holds parameters for getting a job by ID.
type GetJobParams struct {
	ID uuid.UUID
	// WorkspaceID is optional. When non-zero, validates that the job's connection
	// belongs to the specified workspace, preventing cross-tenant access.
	WorkspaceID uuid.UUID
}

// GetJob retrieves a single job by ID.
type GetJob struct {
	storage pipelineservice.Storage
}

// NewGetJob creates a new GetJob use case.
func NewGetJob(storage pipelineservice.Storage) *GetJob {
	return &GetJob{storage: storage}
}

// Execute returns the job matching the given ID.
// When WorkspaceID is provided, it also verifies the job's connection belongs
// to that workspace to prevent IDOR attacks.
func (uc *GetJob) Execute(ctx context.Context, params GetJobParams) (*pipelineservice.Job, error) {
	job, err := uc.storage.Jobs().First(ctx, &pipelineservice.JobFilter{
		ID: filter.Equals(params.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting job: %w", err)
	}

	// When a workspace is specified, verify the job's connection belongs to it.
	if params.WorkspaceID != uuid.Nil {
		_, err := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
			ID:          filter.Equals(job.ConnectionID),
			WorkspaceID: filter.Equals(params.WorkspaceID),
		})
		if err != nil {
			return nil, pipelineservice.ErrJobNotFound
		}
	}

	return job, nil
}
