package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

// GetJobWithAttemptsParams holds parameters for getting a job with its attempts.
type GetJobWithAttemptsParams struct {
	JobID       uuid.UUID
	WorkspaceID uuid.UUID
}

// GetJobWithAttempts retrieves a job with its attempts, verifying workspace ownership.
type GetJobWithAttempts struct {
	getJob          *GetJob
	listJobAttempts *ListJobAttempts
	getConnection   *pipelineconnections.GetConnection
}

// NewGetJobWithAttempts creates a new GetJobWithAttempts use case.
func NewGetJobWithAttempts(
	getJob *GetJob,
	listJobAttempts *ListJobAttempts,
	getConnection *pipelineconnections.GetConnection,
) *GetJobWithAttempts {
	return &GetJobWithAttempts{
		getJob:          getJob,
		listJobAttempts: listJobAttempts,
		getConnection:   getConnection,
	}
}

// Execute gets a job, verifies workspace ownership via connection, and loads attempts.
func (uc *GetJobWithAttempts) Execute(ctx context.Context, params GetJobWithAttemptsParams) (*JobWithAttempts, error) {
	job, err := uc.getJob.Execute(ctx, GetJobParams{ID: params.JobID})
	if err != nil {
		return nil, fmt.Errorf("getting job: %w", err)
	}

	// Verify the job's connection belongs to the workspace.
	if _, err := uc.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          job.ConnectionID,
		WorkspaceID: params.WorkspaceID,
	}); err != nil {
		return nil, fmt.Errorf("connection not found in workspace: %w", err)
	}

	attempts, err := uc.listJobAttempts.Execute(ctx, ListJobAttemptsParams{JobID: params.JobID})
	if err != nil {
		return nil, fmt.Errorf("listing attempts: %w", err)
	}

	return &JobWithAttempts{
		Job:      job,
		Attempts: attempts,
	}, nil
}
