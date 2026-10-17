package pipelineservice

import (
	context "context"

	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	idempotency "github.com/saturn4er/boilerplate-go/lib/idempotency"
	txoutbox "github.com/saturn4er/boilerplate-go/lib/txoutbox"

	// user code 'imports'
	"time"

	"github.com/google/uuid"
	// end user code 'imports'
)

type Storage interface {
	ManagedConnectors() ManagedConnectorsStorage
	Repositories() RepositoriesStorage
	RepositoryConnectors() RepositoryConnectorsStorage
	Sources() SourcesStorage
	Destinations() DestinationsStorage
	Connections() ConnectionsStorage
	Jobs() JobsStorage
	JobAttempts() JobAttemptsStorage
	CatalogDiscoveries() CatalogDiscoveriesStorage
	ConfiguredCatalogs() ConfiguredCatalogsStorage
	JobLogs() JobLogsStorage
	JobLogSegments() JobLogSegmentsStorage
	ConnectionStates() ConnectionStatesStorage
	WorkspaceSettings() WorkspaceSettingsStorage
	ConnectorTasks() ConnectorTasksStorage
	StreamGenerations() StreamGenerationsStorage
	PipelineEvents() PipelineEventsOutbox
	IdempotencyKeys() idempotency.Storage
	ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx Storage) error) error
	WithAdvisoryLock(ctx context.Context, scope string, lockID int64) error
}
type ManagedConnectorsStorage dbutil.EntityStorage[ManagedConnector, ManagedConnectorFilter]
type RepositoriesStorage dbutil.EntityStorage[Repository, RepositoryFilter]
type RepositoryConnectorsStorage dbutil.EntityStorage[RepositoryConnector, RepositoryConnectorFilter]
type SourcesStorage dbutil.EntityStorage[Source, SourceFilter]
type DestinationsStorage dbutil.EntityStorage[Destination, DestinationFilter]
type ConnectionsStorage interface {
	dbutil.EntityStorage[Connection, ConnectionFilter]
	// user code 'Connection metods'
	FindDueConnections(ctx context.Context, limit int) ([]DueConnection, error)
	// end user code 'Connection metods'
}

// user code 'Connection definitions'

// DueConnection represents a connection that is due for a scheduled sync.
type DueConnection struct {
	ConnectionID  uuid.UUID
	SourceID      uuid.UUID
	DestinationID uuid.UUID
	Schedule      string
	MaxAttempts   int
}

// end user code 'Connection definitions'
type JobsStorage interface {
	dbutil.EntityStorage[Job, JobFilter]
	// user code 'Job metods'
	ClaimNextScheduledJob(ctx context.Context, workerID string) (*Job, error)
	CountActiveJobs(ctx context.Context) (int, error)
	UpdateIfStatus(ctx context.Context, job *Job, fromStatuses ...JobStatus) (bool, error)
	TouchHeartbeat(ctx context.Context, id uuid.UUID, at time.Time) (bool, error)
	MarkRunning(ctx context.Context, id uuid.UUID) (bool, error)
	// end user code 'Job metods'
}

// user code 'Job definitions'
// end user code 'Job definitions'
type JobAttemptsStorage dbutil.EntityStorage[JobAttempt, JobAttemptFilter]
type CatalogDiscoveriesStorage dbutil.EntityStorage[CatalogDiscovery, CatalogDiscoveryFilter]
type ConfiguredCatalogsStorage dbutil.EntityStorage[ConfiguredCatalog, ConfiguredCatalogFilter]
type JobLogsStorage interface {
	dbutil.EntityStorage[JobLog, JobLogFilter]
	// user code 'JobLog metods'
	GetLogs(ctx context.Context, jobID uuid.UUID, afterID int64, limit int) ([]JobLog, error)
	InsertLines(ctx context.Context, jobID uuid.UUID, attempt int, lines []string) error
	InsertLine(ctx context.Context, jobID uuid.UUID, attempt int, line string) (int64, error)
	UpdateLine(ctx context.Context, id int64, line string) (bool, error)
	MaxLogID(ctx context.Context, jobID uuid.UUID, attempt int) (int64, error)
	ListAttemptLogs(ctx context.Context, jobID uuid.UUID, attempt int, afterID, upToID int64, limit int) ([]JobLog, error)
	DeleteAttemptLogs(ctx context.Context, jobID uuid.UUID, attempt int, upToID int64) error
	ListUnarchivedAttempts(ctx context.Context, jobID uuid.UUID) ([]int, error)
	// end user code 'JobLog metods'
}

// user code 'JobLog definitions'
// end user code 'JobLog definitions'
type JobLogSegmentsStorage interface {
	dbutil.EntityStorage[JobLogSegment, JobLogSegmentFilter]
	// user code 'JobLogSegment metods'
	LockCurrent(ctx context.Context, jobID uuid.UUID) (*JobLogSegment, error)
	Lock(ctx context.Context, jobID uuid.UUID, attempt int) (*JobLogSegment, error)
	MarkArchived(ctx context.Context, segment *JobLogSegment) (bool, error)
	ListWorkspaceArchiveKeys(ctx context.Context, workspaceID uuid.UUID) ([]string, error)
	// end user code 'JobLogSegment metods'
}

// user code 'JobLogSegment definitions'
// end user code 'JobLogSegment definitions'
type ConnectionStatesStorage dbutil.EntityStorage[ConnectionState, ConnectionStateFilter]
type WorkspaceSettingsStorage dbutil.EntityStorage[WorkspaceSettings, WorkspaceSettingsFilter]
type ConnectorTasksStorage interface {
	dbutil.EntityStorage[ConnectorTask, ConnectorTaskFilter]
	// user code 'ConnectorTask metods'
	ClaimPendingTask(ctx context.Context, workerID string) (*ConnectorTask, error)
	// end user code 'ConnectorTask metods'
}

// user code 'ConnectorTask definitions'
// end user code 'ConnectorTask definitions'
type StreamGenerationsStorage interface {
	dbutil.EntityStorage[StreamGeneration, StreamGenerationFilter]
	// user code 'StreamGeneration metods'
	Upsert(ctx context.Context, generation *StreamGeneration) error
	// end user code 'StreamGeneration metods'
}

// user code 'StreamGeneration definitions'
// end user code 'StreamGeneration definitions'
type PipelineEventsOutbox txoutbox.Outbox[PipelineEvent]
