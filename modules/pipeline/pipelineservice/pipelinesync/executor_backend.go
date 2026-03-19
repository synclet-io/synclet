package pipelinesync

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// HeartbeatResult carries cancellation signal per D-05.
type HeartbeatResult struct {
	Cancelled bool
}

// UpdateJobStatusParams holds parameters for reporting job completion/failure.
type UpdateJobStatusParams struct {
	JobID        uuid.UUID
	Success      bool
	ErrorMessage string
	RecordsRead  int64
	BytesSynced  int64
	DurationMs   int64
}

// ReportCompletionParams holds parameters for reporting completion with stats.
type ReportCompletionParams struct {
	JobID        uuid.UUID
	ConnectionID uuid.UUID
	Success      bool
	ErrorMessage string
	RecordsRead  int64
	BytesSynced  int64
	DurationMs   int64
}

// ClaimConnectorTaskResult bundles everything an executor needs to run a connector task.
type ClaimConnectorTaskResult struct {
	TaskID      uuid.UUID
	TaskType    pipelineservice.ConnectorTaskType
	Image       string
	Config      []byte // Decrypted JSON config (nil for spec)
	WorkspaceID uuid.UUID
}

// ReportConnectorTaskResultParams holds parameters for reporting connector task completion.
type ReportConnectorTaskResultParams struct {
	TaskID       uuid.UUID
	Success      bool
	ErrorMessage string
	Result       []byte // JSON-encoded result
}

// ExecutorBackend abstracts all executor-to-server operations per D-14.
// Two implementations: use-case adapter (standalone) and ConnectRPC client (distributed).
type ExecutorBackend interface {
	ClaimJob(ctx context.Context, workerID string) (*pipelinejobs.ClaimJobBundleResult, error)
	UpdateJobStatus(ctx context.Context, params UpdateJobStatusParams) error
	Heartbeat(ctx context.Context, jobID uuid.UUID, recordsRead, bytesSynced int64) (*HeartbeatResult, error)
	ReportState(ctx context.Context, connectionID, jobID uuid.UUID, stateMsg *protocol.AirbyteStateMessage) error
	ReportCompletion(ctx context.Context, params ReportCompletionParams) error
	ReportConfigUpdate(ctx context.Context, connectorType pipelineservice.ConnectorType, connectorID uuid.UUID, config []byte) error
	ReportLog(ctx context.Context, jobID uuid.UUID, lines []string) error
	IsJobActive(ctx context.Context, jobID string) (bool, error)

	// ClaimConnectorTask claims the next pending connector task (D-15).
	ClaimConnectorTask(ctx context.Context, workerID string) (*ClaimConnectorTaskResult, error)

	// ReportConnectorTaskResult reports the result of a connector task (D-18/D-19).
	ReportConnectorTaskResult(ctx context.Context, params ReportConnectorTaskResultParams) error
}
