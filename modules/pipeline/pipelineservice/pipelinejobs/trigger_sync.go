package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

// TriggerSyncParams holds parameters for triggering a sync with workspace verification.
type TriggerSyncParams struct {
	ConnectionID uuid.UUID
	WorkspaceID  uuid.UUID
}

// TriggerSync verifies workspace ownership and queues a sync job.
type TriggerSync struct {
	getConnection *pipelineconnections.GetConnection
	queueJob      *QueueJob
	audit         pipelineservice.AuditRecorder
}

// NewTriggerSync creates a new TriggerSync use case.
func NewTriggerSync(
	getConnection *pipelineconnections.GetConnection,
	queueJob *QueueJob,
	audit pipelineservice.AuditRecorder,
) *TriggerSync {
	return &TriggerSync{
		getConnection: getConnection,
		queueJob:      queueJob,
		audit:         audit,
	}
}

// Execute verifies workspace ownership and queues a sync job.
func (uc *TriggerSync) Execute(ctx context.Context, params TriggerSyncParams) (*pipelineservice.Job, error) {
	conn, err := uc.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          params.ConnectionID,
		WorkspaceID: params.WorkspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("connection not found in workspace: %w", err)
	}

	job, err := uc.queueJob.Execute(ctx, QueueJobParams{
		ConnectionID: params.ConnectionID,
		JobType:      pipelineservice.JobTypeSync,
	})
	if err != nil {
		return nil, fmt.Errorf("queuing job: %w", err)
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "sync",
		ResourceType:  "connection",
		ResourceID:    conn.ID,
		ResourceLabel: conn.Name,
		After:         map[string]any{"job_id": job.ID},
	})

	return job, nil
}
