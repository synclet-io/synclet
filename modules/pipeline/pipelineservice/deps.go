package pipelineservice

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/pkg/protocol"
)

// SourceReader abstracts reading from a source connector.
// Returns the container's stdout as a ReadCloser, a cleanup function to remove the container,
// and any startup error. The caller must close stdout and call cleanup when done.
type SourceReader interface {
	Read(ctx context.Context, image string, config json.RawMessage, catalog *protocol.ConfiguredAirbyteCatalog, state json.RawMessage, labels map[string]string) (stdout io.ReadCloser, cleanup func(), err error)
}

// DestinationWriter abstracts writing to a destination connector.
// Accepts an io.Reader for stdin (source messages piped to dest). Returns the container's
// stdout as a ReadCloser (for reading dest output/state confirmations), a cleanup function,
// and any startup error. The caller must close stdout and call cleanup when done.
type DestinationWriter interface {
	Write(ctx context.Context, image string, config json.RawMessage, catalog *protocol.ConfiguredAirbyteCatalog, stdin io.Reader, labels map[string]string) (stdout io.ReadCloser, cleanup func(), err error)
}

// ConnectorClient provides connector operations (check connectivity).
type ConnectorClient interface {
	Check(ctx context.Context, image string, config json.RawMessage) error
}

// ResourceConfigurable is an optional interface that SourceReader or DestinationWriter
// implementations may support to accept resource limits before Read/Write calls.
type ResourceConfigurable interface {
	SetResourceLimits(memoryLimit int64, cpuLimit float64)
}

// ConnectorDiscoverer discovers the catalog from a source connector.
type ConnectorDiscoverer interface {
	Discover(ctx context.Context, image string, config json.RawMessage) (*protocol.AirbyteCatalog, error)
}

// ConnectorImageValidator validates that a connector image is allowed.
type ConnectorImageValidator interface {
	ValidateImage(ctx context.Context, dockerRepository string) error
}

// SyncEventEmitter emits events when sync jobs complete or fail.
type SyncEventEmitter interface {
	EmitSyncCompleted(ctx context.Context, event SyncCompletedEvent) error
	EmitSyncFailed(ctx context.Context, event SyncFailedEvent) error
}

// SyncCompletedEvent contains info about a completed sync.
type SyncCompletedEvent struct {
	ConnectionID uuid.UUID
	WorkspaceID  uuid.UUID
	JobID        uuid.UUID
	RecordsRead  int64
	Duration     time.Duration
}

// SyncFailedEvent contains info about a failed sync.
type SyncFailedEvent struct {
	ConnectionID uuid.UUID
	WorkspaceID  uuid.UUID
	JobID        uuid.UUID
	Error        string
}

// SyncStats holds sync execution statistics.
type SyncStats struct {
	RecordsRead int64         `json:"records_read"`
	BytesSynced int64         `json:"bytes_synced"`
	Duration    time.Duration `json:"duration"`
}

// K8sSyncLauncher launches sync jobs on Kubernetes.
type K8sSyncLauncher interface {
	Launch(ctx context.Context, jobID uuid.UUID) error
}

// ConnectorSpecFetcher extracts a connector's spec by running its spec command.
type ConnectorSpecFetcher interface {
	Spec(ctx context.Context, image string) (string, error)
}

// ImagePuller pulls container images.
type ImagePuller interface {
	Pull(ctx context.Context, image string) error
}

// StatsStorage provides raw database queries for stats aggregation.
// Defined manually (not generated via gen.models.yaml) per architecture decision.
type StatsStorage interface {
	// UpsertRollups computes and upserts rollup buckets from job data.
	UpsertRollups(ctx context.Context, bucketSize BucketSize, since time.Time, truncUnit string) error

	// QueryConnectionRollup returns aggregated rollup data for a connection over a time range.
	QueryConnectionRollup(ctx context.Context, connectionID uuid.UUID, from, to time.Time) (ConnectionRollup, error)

	// QueryLastSyncAt returns the last completed_at time for a connection.
	QueryLastSyncAt(ctx context.Context, connectionID uuid.UUID) (*time.Time, error)

	// QueryLastJobInfo returns the status and completed_at of the most recent job for a connection.
	QueryLastJobInfo(ctx context.Context, connectionID uuid.UUID) (*LastJobInfo, error)

	// QueryDurationChart returns raw duration chart rows for a connection's recent jobs.
	QueryDurationChart(ctx context.Context, connectionID uuid.UUID, from, to time.Time, limit int) ([]DurationChartRow, error)

	// QueryRecordsChart returns raw records chart rows from rollups for a connection.
	QueryRecordsChart(ctx context.Context, connectionID uuid.UUID, from, to time.Time) ([]RecordsChartRow, error)

	// QueryConnectionFailedJobs returns failed job error info for a connection in a time range.
	QueryConnectionFailedJobs(ctx context.Context, connectionID uuid.UUID, from, to time.Time) ([]FailedJobRow, error)

	// QueryWorkspaceRollup returns aggregated rollup totals for a workspace over a time range.
	QueryWorkspaceRollup(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) (RollupTotals, error)

	// QueryTopConnections returns top connections by records synced for a workspace.
	QueryTopConnections(ctx context.Context, workspaceID uuid.UUID, from, to time.Time, limit int) ([]TopConnectionRow, error)

	// QueryConnectionLastCompletedAt returns the last completed job time for a connection.
	QueryConnectionLastCompletedAt(ctx context.Context, connectionID uuid.UUID) (*time.Time, error)

	// QueryConnectionSparkline returns recent rollup records_read values for a connection.
	QueryConnectionSparkline(ctx context.Context, connectionID uuid.UUID, from, to time.Time, limit int) ([]int64, error)

	// QueryWorkspaceFailedJobs returns failed job error info for a workspace in a time range.
	QueryWorkspaceFailedJobs(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]FailedJobRow, error)

	// QuerySyncTimeline returns time-bucketed sync data for timeline charts.
	QuerySyncTimeline(ctx context.Context, workspaceID uuid.UUID, bucketSize BucketSize, from, to time.Time, connectionID *uuid.UUID) ([]TimelineRow, error)

	// QueryLastJobInfoBatch returns the most recent job info for each of the given connection IDs.
	QueryLastJobInfoBatch(ctx context.Context, connectionIDs []uuid.UUID) (map[uuid.UUID]*LastJobInfo, error)

	// QueryConnectionLastCompletedAtBatch returns the last completed job time for each of the given connection IDs.
	QueryConnectionLastCompletedAtBatch(ctx context.Context, connectionIDs []uuid.UUID) (map[uuid.UUID]*time.Time, error)

	// QueryConnectionSparklineBatch returns recent rollup records_read values for each of the given connection IDs.
	QueryConnectionSparklineBatch(ctx context.Context, connectionIDs []uuid.UUID, from, to time.Time, limit int) (map[uuid.UUID][]int64, error)
}

// JobRetentionStorage provides bulk deletion for job retention cleanup.
type JobRetentionStorage interface {
	DeleteOldestTerminalJobs(ctx context.Context, workspaceID uuid.UUID, keepCount int) (int64, error)
}

// SecretsProvider manages encrypted secret storage.
type SecretsProvider interface {
	StoreSecret(ctx context.Context, ownerType string, ownerID uuid.UUID, plaintext string) (secretRef string, err error)
	RetrieveSecret(ctx context.Context, secretRef string) (plaintext string, err error)
	DeleteSecret(ctx context.Context, secretRef string) error
	DeleteByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) error
}
