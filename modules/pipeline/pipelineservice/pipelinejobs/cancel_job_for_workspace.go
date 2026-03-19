package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

// CancelJobForWorkspaceParams holds parameters for cancelling a job with workspace verification.
type CancelJobForWorkspaceParams struct {
	JobID       uuid.UUID
	WorkspaceID uuid.UUID
}

// CancelJobForWorkspace verifies workspace ownership via the job's connection and cancels the job.
type CancelJobForWorkspace struct {
	getJob        *GetJob
	getConnection *pipelineconnections.GetConnection
	cancelJob     *CancelJob
}

// NewCancelJobForWorkspace creates a new CancelJobForWorkspace use case.
func NewCancelJobForWorkspace(
	getJob *GetJob,
	getConnection *pipelineconnections.GetConnection,
	cancelJob *CancelJob,
) *CancelJobForWorkspace {
	return &CancelJobForWorkspace{
		getJob:        getJob,
		getConnection: getConnection,
		cancelJob:     cancelJob,
	}
}

// Execute gets the job, verifies its connection belongs to the workspace, and cancels it.
func (uc *CancelJobForWorkspace) Execute(ctx context.Context, params CancelJobForWorkspaceParams) error {
	job, err := uc.getJob.Execute(ctx, GetJobParams{ID: params.JobID})
	if err != nil {
		return fmt.Errorf("getting job: %w", err)
	}

	if _, err := uc.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          job.ConnectionID,
		WorkspaceID: params.WorkspaceID,
	}); err != nil {
		return fmt.Errorf("connection not found in workspace: %w", err)
	}

	if err := uc.cancelJob.Execute(ctx, CancelJobParams{ID: params.JobID}); err != nil {
		return fmt.Errorf("cancelling job: %w", err)
	}

	return nil
}
