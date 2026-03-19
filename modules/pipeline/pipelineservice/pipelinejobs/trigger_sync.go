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
}

// NewTriggerSync creates a new TriggerSync use case.
func NewTriggerSync(
	getConnection *pipelineconnections.GetConnection,
	queueJob *QueueJob,
) *TriggerSync {
	return &TriggerSync{
		getConnection: getConnection,
		queueJob:      queueJob,
	}
}

// Execute verifies workspace ownership and queues a sync job.
func (uc *TriggerSync) Execute(ctx context.Context, params TriggerSyncParams) (*pipelineservice.Job, error) {
	if _, err := uc.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          params.ConnectionID,
		WorkspaceID: params.WorkspaceID,
	}); err != nil {
		return nil, fmt.Errorf("connection not found in workspace: %w", err)
	}

	job, err := uc.queueJob.Execute(ctx, QueueJobParams{
		ConnectionID: params.ConnectionID,
		JobType:      pipelineservice.JobTypeSync,
	})
	if err != nil {
		return nil, fmt.Errorf("queuing job: %w", err)
	}

	return job, nil
}
