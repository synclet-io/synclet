package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
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
	audit         pipelineservice.AuditRecorder
}

// NewCancelJobForWorkspace creates a new CancelJobForWorkspace use case.
func NewCancelJobForWorkspace(
	getJob *GetJob,
	getConnection *pipelineconnections.GetConnection,
	cancelJob *CancelJob,
	audit pipelineservice.AuditRecorder,
) *CancelJobForWorkspace {
	return &CancelJobForWorkspace{
		getJob:        getJob,
		getConnection: getConnection,
		cancelJob:     cancelJob,
		audit:         audit,
	}
}

// Execute gets the job, verifies its connection belongs to the workspace, and cancels it.
// The cancellation is recorded as an "update" audit event on the connection.
func (uc *CancelJobForWorkspace) Execute(ctx context.Context, params CancelJobForWorkspaceParams) error {
	job, err := uc.getJob.Execute(ctx, GetJobParams{ID: params.JobID})
	if err != nil {
		return fmt.Errorf("getting job: %w", err)
	}

	conn, err := uc.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          job.ConnectionID,
		WorkspaceID: params.WorkspaceID,
	})
	if err != nil {
		return fmt.Errorf("connection not found in workspace: %w", err)
	}

	if err := uc.cancelJob.Execute(ctx, CancelJobParams{ID: params.JobID}); err != nil {
		return fmt.Errorf("cancelling job: %w", err)
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "update",
		ResourceType:  "connection",
		ResourceID:    conn.ID,
		ResourceLabel: conn.Name,
		Before:        map[string]any{"job_id": job.ID.String(), "job_status": job.Status.String()},
		After:         map[string]any{"job_id": job.ID.String(), "job_status": pipelineservice.JobStatusCancelled.String()},
	})

	return nil
}
