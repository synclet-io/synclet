package pipelineservice

import (
	context "context"

	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	idempotency "github.com/saturn4er/boilerplate-go/lib/idempotency"

	// user code 'imports'
	"github.com/google/uuid"
	// end user code 'imports'
)

type Storage interface {
	ManagedConnectors() ManagedConnectorsStorage
	Repositorys() RepositorysStorage
	RepositoryConnectors() RepositoryConnectorsStorage
	Sources() SourcesStorage
	Destinations() DestinationsStorage
	Connections() ConnectionsStorage
	Jobs() JobsStorage
	JobAttempts() JobAttemptsStorage
	CatalogDiscoverys() CatalogDiscoverysStorage
	ConfiguredCatalogs() ConfiguredCatalogsStorage
	JobLogs() JobLogsStorage
	ConnectionStates() ConnectionStatesStorage
	WorkspaceSettingss() WorkspaceSettingssStorage
	ConnectorTasks() ConnectorTasksStorage
	StreamGenerations() StreamGenerationsStorage
	IdempotencyKeys() idempotency.Storage
	ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx Storage) error) error
	WithAdvisoryLock(ctx context.Context, scope string, lockID int64) error
}
type ManagedConnectorsStorage dbutil.EntityStorage[ManagedConnector, ManagedConnectorFilter]
type RepositorysStorage dbutil.EntityStorage[Repository, RepositoryFilter]
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
	// end user code 'Job metods'
}

// user code 'Job definitions'
// end user code 'Job definitions'
type JobAttemptsStorage dbutil.EntityStorage[JobAttempt, JobAttemptFilter]
type CatalogDiscoverysStorage dbutil.EntityStorage[CatalogDiscovery, CatalogDiscoveryFilter]
type ConfiguredCatalogsStorage dbutil.EntityStorage[ConfiguredCatalog, ConfiguredCatalogFilter]
type JobLogsStorage interface {
	dbutil.EntityStorage[JobLog, JobLogFilter]
	// user code 'JobLog metods'
	AppendLog(ctx context.Context, jobID uuid.UUID, line string) error
	GetLogs(ctx context.Context, jobID uuid.UUID, afterID int64, limit int) ([]JobLog, error)
	BatchAppendLogs(ctx context.Context, jobID uuid.UUID, lines []string) error
	// end user code 'JobLog metods'
}

// user code 'JobLog definitions'
// end user code 'JobLog definitions'
type ConnectionStatesStorage dbutil.EntityStorage[ConnectionState, ConnectionStateFilter]
type WorkspaceSettingssStorage dbutil.EntityStorage[WorkspaceSettings, WorkspaceSettingsFilter]
type ConnectorTasksStorage interface {
	dbutil.EntityStorage[ConnectorTask, ConnectorTaskFilter]
	// user code 'ConnectorTask metods'
	ClaimPendingTask(ctx context.Context, workerID string) (*ConnectorTask, error)
	// end user code 'ConnectorTask metods'
}

// user code 'ConnectorTask definitions'
// end user code 'ConnectorTask definitions'
type StreamGenerationsStorage dbutil.EntityStorage[StreamGeneration, StreamGenerationFilter]
