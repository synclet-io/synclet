package pipelineexec

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
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
	// SourceExitCode and DestExitCode are the connectors' non-zero exit codes;
	// 0 means a clean exit or none known.
	SourceExitCode int
	DestExitCode   int
}

// ClaimJobResult bundles everything an executor needs to run a sync.
type ClaimJobResult struct {
	Job                   *pipelineservice.Job
	ConnectionID          uuid.UUID
	WorkspaceID           uuid.UUID
	SourceID              uuid.UUID
	DestinationID         uuid.UUID
	SourceImage           string
	SourceConfig          []byte // Decrypted JSON
	DestImage             string
	DestConfig            []byte // Decrypted JSON
	ConfiguredCatalog     []byte // JSON-marshaled
	StateBlob             []byte // JSON, may be nil
	SourceRuntimeConfig   string
	DestRuntimeConfig     string
	NamespaceDefinition   pipelineservice.NamespaceDefinition
	CustomNamespaceFormat string
	StreamPrefix          string
	MaxAttempts           int
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

// SyncReporter is the part of the executor API a running sync reports
// through: confirmed state, connector config updates and log lines.
type SyncReporter interface {
	ReportState(ctx context.Context, connectionID, jobID uuid.UUID, stateMsg *protocol.AirbyteStateMessage) error
	ReportConfigUpdate(ctx context.Context, connectorType pipelineservice.ConnectorType, connectorID uuid.UUID, config []byte) error
	ReportLog(ctx context.Context, jobID uuid.UUID, lines []string) error
}

// ExecutorBackend is the only way an executor reaches the control plane.
// Two implementations: the use-case adapter inside `server --standalone`
// and the ConnectRPC client of the executor commands.
type ExecutorBackend interface {
	SyncReporter

	ClaimJob(ctx context.Context, workerID string) (*ClaimJobResult, error)
	UpdateJobStatus(ctx context.Context, params UpdateJobStatusParams) error
	// MarkJobRunning reports that the job's connectors have started.
	MarkJobRunning(ctx context.Context, jobID uuid.UUID) error
	Heartbeat(ctx context.Context, jobID uuid.UUID, recordsRead, bytesSynced int64) (*HeartbeatResult, error)
	ReportCompletion(ctx context.Context, params ReportCompletionParams) error
	// SetK8sJobName stores the Kubernetes Job name launched for a job.
	SetK8sJobName(ctx context.Context, jobID uuid.UUID, name string) error
	IsJobActive(ctx context.Context, jobID string) (bool, error)
	IsTaskActive(ctx context.Context, taskID string) (bool, error)
	FailJob(ctx context.Context, jobID string, reason string) error

	// ClaimConnectorTask claims the next pending connector task (D-15).
	ClaimConnectorTask(ctx context.Context, workerID string) (*ClaimConnectorTaskResult, error)

	// ReportConnectorTaskResult reports the result of a connector task (D-18/D-19).
	ReportConnectorTaskResult(ctx context.Context, params ReportConnectorTaskResultParams) error
}
