package pipelinestorage

import (
	fmt "fmt"
	time "time"

	uuid "github.com/google/uuid"

	pipelineservice "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	// user code 'imports'
	// end user code 'imports'
)

type dbManagedConnector struct {
	ID            uuid.UUID  `gorm:"column:id;"`
	WorkspaceID   uuid.UUID  `gorm:"column:workspace_id;"`
	DockerImage   string     `gorm:"column:docker_image;type:text;"`
	DockerTag     string     `gorm:"column:docker_tag;type:text;"`
	Name          string     `gorm:"column:name;type:text;"`
	ConnectorType string     `gorm:"column:connector_type;type:text;"`
	Spec          jsonb      `gorm:"column:spec;"`
	CreatedAt     time.Time  `gorm:"column:created_at;"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;"`
	RepositoryID  *uuid.UUID `gorm:"column:repository_id;"`
}

func convertManagedConnectorToDB(src *pipelineservice.ManagedConnector) (*dbManagedConnector, error) {
	result := &dbManagedConnector{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.DockerImage = src.DockerImage
	result.DockerTag = src.DockerTag
	result.Name = src.Name
	tmp5, err := convertConnectorTypeToDB(src.ConnectorType)
	if err != nil {
		return nil, err
	}
	result.ConnectorType = tmp5
	result.Spec = src.Spec
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	if src.RepositoryID == nil {
		result.RepositoryID = nil
	} else {
		result.RepositoryID = toPtr(fromPtr(src.RepositoryID))
	}
	return result, nil
}

func convertManagedConnectorFromDB(src *dbManagedConnector) (*pipelineservice.ManagedConnector, error) {
	result := &pipelineservice.ManagedConnector{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.DockerImage = src.DockerImage
	result.DockerTag = src.DockerTag
	result.Name = src.Name
	tmp16, err := convertConnectorTypeFromDB(src.ConnectorType)
	if err != nil {
		return nil, err
	}
	result.ConnectorType = tmp16
	result.Spec = src.Spec
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	if src.RepositoryID == nil {
		result.RepositoryID = nil
	} else {
		result.RepositoryID = toPtr(fromPtr(src.RepositoryID))
	}
	return result, nil
}
func (a dbManagedConnector) TableName() string {
	return "pipeline.managed_connectors"
}

type dbRepository struct {
	ID             uuid.UUID  `gorm:"column:id;"`
	WorkspaceID    uuid.UUID  `gorm:"column:workspace_id;"`
	Name           string     `gorm:"column:name;type:text;"`
	URL            string     `gorm:"column:url;type:text;"`
	AuthHeader     *string    `gorm:"column:auth_header;type:text;"`
	Status         string     `gorm:"column:status;type:text;"`
	LastSyncedAt   *time.Time `gorm:"column:last_synced_at;"`
	ConnectorCount int        `gorm:"column:connector_count;"`
	LastError      *string    `gorm:"column:last_error;type:text;"`
	CreatedAt      time.Time  `gorm:"column:created_at;"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;"`
}

func convertRepositoryToDB(src *pipelineservice.Repository) (*dbRepository, error) {
	result := &dbRepository{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	result.URL = src.URL
	if src.AuthHeader == nil {
		result.AuthHeader = nil
	} else {
		result.AuthHeader = toPtr(fromPtr(src.AuthHeader))
	}
	tmp6, err := convertRepositoryStatusToDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp6
	if src.LastSyncedAt == nil {
		result.LastSyncedAt = nil
	} else {
		result.LastSyncedAt = toPtr((fromPtr(src.LastSyncedAt)).UTC())
	}
	result.ConnectorCount = src.ConnectorCount
	if src.LastError == nil {
		result.LastError = nil
	} else {
		result.LastError = toPtr(fromPtr(src.LastError))
	}
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	return result, nil
}

func convertRepositoryFromDB(src *dbRepository) (*pipelineservice.Repository, error) {
	result := &pipelineservice.Repository{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	result.URL = src.URL
	if src.AuthHeader == nil {
		result.AuthHeader = nil
	} else {
		result.AuthHeader = toPtr(fromPtr(src.AuthHeader))
	}
	tmp20, err := convertRepositoryStatusFromDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp20
	if src.LastSyncedAt == nil {
		result.LastSyncedAt = nil
	} else {
		result.LastSyncedAt = toPtr(fromPtr(src.LastSyncedAt))
	}
	result.ConnectorCount = src.ConnectorCount
	if src.LastError == nil {
		result.LastError = nil
	} else {
		result.LastError = toPtr(fromPtr(src.LastError))
	}
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	return result, nil
}
func (a dbRepository) TableName() string {
	return "pipeline.repositories"
}

type dbRepositoryConnector struct {
	ID               uuid.UUID `gorm:"column:id;"`
	RepositoryID     uuid.UUID `gorm:"column:repository_id;"`
	DockerRepository string    `gorm:"column:docker_repository;type:text;"`
	DockerImageTag   string    `gorm:"column:docker_image_tag;type:text;"`
	Name             string    `gorm:"column:name;type:text;"`
	ConnectorType    string    `gorm:"column:connector_type;type:text;"`
	DocumentationURL string    `gorm:"column:documentation_url;type:text;"`
	ReleaseStage     string    `gorm:"column:release_stage;type:text;"`
	IconURL          string    `gorm:"column:icon_url;type:text;"`
	Spec             jsonb     `gorm:"column:spec;"`
	SupportLevel     string    `gorm:"column:support_level;type:text;"`
	License          string    `gorm:"column:license;type:text;"`
	SourceType       string    `gorm:"column:source_type;type:text;"`
	Metadata         jsonb     `gorm:"column:metadata;"`
}

func convertRepositoryConnectorToDB(src *pipelineservice.RepositoryConnector) (*dbRepositoryConnector, error) {
	result := &dbRepositoryConnector{}
	result.ID = src.ID
	result.RepositoryID = src.RepositoryID
	result.DockerRepository = src.DockerRepository
	result.DockerImageTag = src.DockerImageTag
	result.Name = src.Name
	tmp5, err := convertConnectorTypeToDB(src.ConnectorType)
	if err != nil {
		return nil, err
	}
	result.ConnectorType = tmp5
	result.DocumentationURL = src.DocumentationURL
	tmp7, err := convertReleaseStageToDB(src.ReleaseStage)
	if err != nil {
		return nil, err
	}
	result.ReleaseStage = tmp7
	result.IconURL = src.IconURL
	result.Spec = src.Spec
	tmp10, err := convertSupportLevelToDB(src.SupportLevel)
	if err != nil {
		return nil, err
	}
	result.SupportLevel = tmp10
	result.License = src.License
	tmp12, err := convertSourceTypeToDB(src.SourceType)
	if err != nil {
		return nil, err
	}
	result.SourceType = tmp12
	result.Metadata = src.Metadata
	return result, nil
}

func convertRepositoryConnectorFromDB(src *dbRepositoryConnector) (*pipelineservice.RepositoryConnector, error) {
	result := &pipelineservice.RepositoryConnector{}
	result.ID = src.ID
	result.RepositoryID = src.RepositoryID
	result.DockerRepository = src.DockerRepository
	result.DockerImageTag = src.DockerImageTag
	result.Name = src.Name
	tmp19, err := convertConnectorTypeFromDB(src.ConnectorType)
	if err != nil {
		return nil, err
	}
	result.ConnectorType = tmp19
	result.DocumentationURL = src.DocumentationURL
	tmp21, err := convertReleaseStageFromDB(src.ReleaseStage)
	if err != nil {
		return nil, err
	}
	result.ReleaseStage = tmp21
	result.IconURL = src.IconURL
	result.Spec = src.Spec
	tmp24, err := convertSupportLevelFromDB(src.SupportLevel)
	if err != nil {
		return nil, err
	}
	result.SupportLevel = tmp24
	result.License = src.License
	tmp26, err := convertSourceTypeFromDB(src.SourceType)
	if err != nil {
		return nil, err
	}
	result.SourceType = tmp26
	result.Metadata = src.Metadata
	return result, nil
}
func (a dbRepositoryConnector) TableName() string {
	return "pipeline.repository_connectors"
}

type dbSource struct {
	ID                 uuid.UUID `gorm:"column:id;"`
	WorkspaceID        uuid.UUID `gorm:"column:workspace_id;"`
	Name               string    `gorm:"column:name;type:text;"`
	ManagedConnectorID uuid.UUID `gorm:"column:managed_connector_id;"`
	Config             jsonb     `gorm:"column:config;"`
	CreatedAt          time.Time `gorm:"column:created_at;"`
	UpdatedAt          time.Time `gorm:"column:updated_at;"`
	RuntimeConfig      *jsonb    `gorm:"column:runtime_config;"`
}

func convertSourceToDB(src *pipelineservice.Source) (*dbSource, error) {
	result := &dbSource{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	result.ManagedConnectorID = src.ManagedConnectorID
	result.Config = src.Config
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	if src.RuntimeConfig == nil {
		result.RuntimeConfig = nil
	} else {
		result.RuntimeConfig = toPtr(fromPtr(src.RuntimeConfig))
	}
	return result, nil
}

func convertSourceFromDB(src *dbSource) (*pipelineservice.Source, error) {
	result := &pipelineservice.Source{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	result.ManagedConnectorID = src.ManagedConnectorID
	result.Config = src.Config
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	if src.RuntimeConfig == nil {
		result.RuntimeConfig = nil
	} else {
		result.RuntimeConfig = toPtr(fromPtr(src.RuntimeConfig))
	}
	return result, nil
}
func (a dbSource) TableName() string {
	return "pipeline.sources"
}

type dbDestination struct {
	ID                 uuid.UUID `gorm:"column:id;"`
	WorkspaceID        uuid.UUID `gorm:"column:workspace_id;"`
	Name               string    `gorm:"column:name;type:text;"`
	ManagedConnectorID uuid.UUID `gorm:"column:managed_connector_id;"`
	Config             jsonb     `gorm:"column:config;"`
	CreatedAt          time.Time `gorm:"column:created_at;"`
	UpdatedAt          time.Time `gorm:"column:updated_at;"`
	RuntimeConfig      *jsonb    `gorm:"column:runtime_config;"`
}

func convertDestinationToDB(src *pipelineservice.Destination) (*dbDestination, error) {
	result := &dbDestination{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	result.ManagedConnectorID = src.ManagedConnectorID
	result.Config = src.Config
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	if src.RuntimeConfig == nil {
		result.RuntimeConfig = nil
	} else {
		result.RuntimeConfig = toPtr(fromPtr(src.RuntimeConfig))
	}
	return result, nil
}

func convertDestinationFromDB(src *dbDestination) (*pipelineservice.Destination, error) {
	result := &pipelineservice.Destination{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	result.ManagedConnectorID = src.ManagedConnectorID
	result.Config = src.Config
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	if src.RuntimeConfig == nil {
		result.RuntimeConfig = nil
	} else {
		result.RuntimeConfig = toPtr(fromPtr(src.RuntimeConfig))
	}
	return result, nil
}
func (a dbDestination) TableName() string {
	return "pipeline.destinations"
}

type dbConnection struct {
	ID                    uuid.UUID  `gorm:"column:id;"`
	WorkspaceID           uuid.UUID  `gorm:"column:workspace_id;"`
	Name                  string     `gorm:"column:name;type:text;"`
	Status                string     `gorm:"column:status;type:text;"`
	SourceID              uuid.UUID  `gorm:"column:source_id;"`
	DestinationID         uuid.UUID  `gorm:"column:destination_id;"`
	Schedule              *string    `gorm:"column:schedule;type:text;"`
	SchemaChangePolicy    string     `gorm:"column:schema_change_policy;type:text;"`
	MaxAttempts           int        `gorm:"column:max_attempts;"`
	NamespaceDefinition   string     `gorm:"column:namespace_definition;type:text;"`
	CustomNamespaceFormat *string    `gorm:"column:custom_namespace_format;type:text;"`
	StreamPrefix          *string    `gorm:"column:stream_prefix;type:text;"`
	NextScheduledAt       *time.Time `gorm:"column:next_scheduled_at;"`
	CreatedAt             time.Time  `gorm:"column:created_at;"`
	UpdatedAt             time.Time  `gorm:"column:updated_at;"`
}

func convertConnectionToDB(src *pipelineservice.Connection) (*dbConnection, error) {
	result := &dbConnection{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	tmp3, err := convertConnectionStatusToDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp3
	result.SourceID = src.SourceID
	result.DestinationID = src.DestinationID
	if src.Schedule == nil {
		result.Schedule = nil
	} else {
		result.Schedule = toPtr(fromPtr(src.Schedule))
	}
	tmp8, err := convertSchemaChangePolicyToDB(src.SchemaChangePolicy)
	if err != nil {
		return nil, err
	}
	result.SchemaChangePolicy = tmp8
	result.MaxAttempts = src.MaxAttempts
	tmp10, err := convertNamespaceDefinitionToDB(src.NamespaceDefinition)
	if err != nil {
		return nil, err
	}
	result.NamespaceDefinition = tmp10
	if src.CustomNamespaceFormat == nil {
		result.CustomNamespaceFormat = nil
	} else {
		result.CustomNamespaceFormat = toPtr(fromPtr(src.CustomNamespaceFormat))
	}
	if src.StreamPrefix == nil {
		result.StreamPrefix = nil
	} else {
		result.StreamPrefix = toPtr(fromPtr(src.StreamPrefix))
	}
	if src.NextScheduledAt == nil {
		result.NextScheduledAt = nil
	} else {
		result.NextScheduledAt = toPtr((fromPtr(src.NextScheduledAt)).UTC())
	}
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	return result, nil
}

func convertConnectionFromDB(src *dbConnection) (*pipelineservice.Connection, error) {
	result := &pipelineservice.Connection{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	tmp22, err := convertConnectionStatusFromDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp22
	result.SourceID = src.SourceID
	result.DestinationID = src.DestinationID
	if src.Schedule == nil {
		result.Schedule = nil
	} else {
		result.Schedule = toPtr(fromPtr(src.Schedule))
	}
	tmp27, err := convertSchemaChangePolicyFromDB(src.SchemaChangePolicy)
	if err != nil {
		return nil, err
	}
	result.SchemaChangePolicy = tmp27
	result.MaxAttempts = src.MaxAttempts
	tmp29, err := convertNamespaceDefinitionFromDB(src.NamespaceDefinition)
	if err != nil {
		return nil, err
	}
	result.NamespaceDefinition = tmp29
	if src.CustomNamespaceFormat == nil {
		result.CustomNamespaceFormat = nil
	} else {
		result.CustomNamespaceFormat = toPtr(fromPtr(src.CustomNamespaceFormat))
	}
	if src.StreamPrefix == nil {
		result.StreamPrefix = nil
	} else {
		result.StreamPrefix = toPtr(fromPtr(src.StreamPrefix))
	}
	if src.NextScheduledAt == nil {
		result.NextScheduledAt = nil
	} else {
		result.NextScheduledAt = toPtr(fromPtr(src.NextScheduledAt))
	}
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	return result, nil
}
func (a dbConnection) TableName() string {
	return "pipeline.connections"
}

type dbJob struct {
	ID            uuid.UUID  `gorm:"column:id;"`
	ConnectionID  uuid.UUID  `gorm:"column:connection_id;"`
	Status        string     `gorm:"column:status;type:text;"`
	JobType       string     `gorm:"column:job_type;type:text;"`
	ScheduledAt   time.Time  `gorm:"column:scheduled_at;"`
	StartedAt     *time.Time `gorm:"column:started_at;"`
	CompletedAt   *time.Time `gorm:"column:completed_at;"`
	Error         *string    `gorm:"column:error;type:text;"`
	Attempt       int        `gorm:"column:attempt;"`
	MaxAttempts   int        `gorm:"column:max_attempts;"`
	WorkerID      *string    `gorm:"column:worker_id;type:text;"`
	HeartbeatAt   *time.Time `gorm:"column:heartbeat_at;"`
	K8sJobName    *string    `gorm:"column:k8s_job_name;type:text;"`
	FailureReason *string    `gorm:"column:failure_reason;type:text;"`
	CreatedAt     time.Time  `gorm:"column:created_at;"`
}

func convertJobToDB(src *pipelineservice.Job) (*dbJob, error) {
	result := &dbJob{}
	result.ID = src.ID
	result.ConnectionID = src.ConnectionID
	tmp2, err := convertJobStatusToDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp2
	tmp3, err := convertJobTypeToDB(src.JobType)
	if err != nil {
		return nil, err
	}
	result.JobType = tmp3
	result.ScheduledAt = (src.ScheduledAt).UTC()
	if src.StartedAt == nil {
		result.StartedAt = nil
	} else {
		result.StartedAt = toPtr((fromPtr(src.StartedAt)).UTC())
	}
	if src.CompletedAt == nil {
		result.CompletedAt = nil
	} else {
		result.CompletedAt = toPtr((fromPtr(src.CompletedAt)).UTC())
	}
	if src.Error == nil {
		result.Error = nil
	} else {
		result.Error = toPtr(fromPtr(src.Error))
	}
	result.Attempt = src.Attempt
	result.MaxAttempts = src.MaxAttempts
	if src.WorkerID == nil {
		result.WorkerID = nil
	} else {
		result.WorkerID = toPtr(fromPtr(src.WorkerID))
	}
	if src.HeartbeatAt == nil {
		result.HeartbeatAt = nil
	} else {
		result.HeartbeatAt = toPtr((fromPtr(src.HeartbeatAt)).UTC())
	}
	if src.K8sJobName == nil {
		result.K8sJobName = nil
	} else {
		result.K8sJobName = toPtr(fromPtr(src.K8sJobName))
	}
	if src.FailureReason == nil {
		result.FailureReason = nil
	} else {
		result.FailureReason = toPtr(fromPtr(src.FailureReason))
	}
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertJobFromDB(src *dbJob) (*pipelineservice.Job, error) {
	result := &pipelineservice.Job{}
	result.ID = src.ID
	result.ConnectionID = src.ConnectionID
	tmp24, err := convertJobStatusFromDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp24
	tmp25, err := convertJobTypeFromDB(src.JobType)
	if err != nil {
		return nil, err
	}
	result.JobType = tmp25
	result.ScheduledAt = src.ScheduledAt
	if src.StartedAt == nil {
		result.StartedAt = nil
	} else {
		result.StartedAt = toPtr(fromPtr(src.StartedAt))
	}
	if src.CompletedAt == nil {
		result.CompletedAt = nil
	} else {
		result.CompletedAt = toPtr(fromPtr(src.CompletedAt))
	}
	if src.Error == nil {
		result.Error = nil
	} else {
		result.Error = toPtr(fromPtr(src.Error))
	}
	result.Attempt = src.Attempt
	result.MaxAttempts = src.MaxAttempts
	if src.WorkerID == nil {
		result.WorkerID = nil
	} else {
		result.WorkerID = toPtr(fromPtr(src.WorkerID))
	}
	if src.HeartbeatAt == nil {
		result.HeartbeatAt = nil
	} else {
		result.HeartbeatAt = toPtr(fromPtr(src.HeartbeatAt))
	}
	if src.K8sJobName == nil {
		result.K8sJobName = nil
	} else {
		result.K8sJobName = toPtr(fromPtr(src.K8sJobName))
	}
	if src.FailureReason == nil {
		result.FailureReason = nil
	} else {
		result.FailureReason = toPtr(fromPtr(src.FailureReason))
	}
	result.CreatedAt = src.CreatedAt
	return result, nil
}
func (a dbJob) TableName() string {
	return "pipeline.jobs"
}

type dbJobAttempt struct {
	ID            uuid.UUID  `gorm:"column:id;"`
	JobID         uuid.UUID  `gorm:"column:job_id;"`
	AttemptNumber int        `gorm:"column:attempt_number;"`
	StartedAt     time.Time  `gorm:"column:started_at;"`
	CompletedAt   *time.Time `gorm:"column:completed_at;"`
	Error         *string    `gorm:"column:error;type:text;"`
	SyncStatsJSON jsonb      `gorm:"column:sync_stats_json;"`
}

func convertJobAttemptToDB(src *pipelineservice.JobAttempt) (*dbJobAttempt, error) {
	result := &dbJobAttempt{}
	result.ID = src.ID
	result.JobID = src.JobID
	result.AttemptNumber = src.AttemptNumber
	result.StartedAt = (src.StartedAt).UTC()
	if src.CompletedAt == nil {
		result.CompletedAt = nil
	} else {
		result.CompletedAt = toPtr((fromPtr(src.CompletedAt)).UTC())
	}
	if src.Error == nil {
		result.Error = nil
	} else {
		result.Error = toPtr(fromPtr(src.Error))
	}
	result.SyncStatsJSON = src.SyncStatsJSON
	return result, nil
}

func convertJobAttemptFromDB(src *dbJobAttempt) (*pipelineservice.JobAttempt, error) {
	result := &pipelineservice.JobAttempt{}
	result.ID = src.ID
	result.JobID = src.JobID
	result.AttemptNumber = src.AttemptNumber
	result.StartedAt = src.StartedAt
	if src.CompletedAt == nil {
		result.CompletedAt = nil
	} else {
		result.CompletedAt = toPtr(fromPtr(src.CompletedAt))
	}
	if src.Error == nil {
		result.Error = nil
	} else {
		result.Error = toPtr(fromPtr(src.Error))
	}
	result.SyncStatsJSON = src.SyncStatsJSON
	return result, nil
}
func (a dbJobAttempt) TableName() string {
	return "pipeline.job_attempts"
}

type dbCatalogDiscovery struct {
	ID           uuid.UUID `gorm:"column:id;"`
	SourceID     uuid.UUID `gorm:"column:source_id;"`
	Version      int       `gorm:"column:version;"`
	CatalogJSON  jsonb     `gorm:"column:catalog_json;"`
	DiscoveredAt time.Time `gorm:"column:discovered_at;"`
}

func convertCatalogDiscoveryToDB(src *pipelineservice.CatalogDiscovery) (*dbCatalogDiscovery, error) {
	result := &dbCatalogDiscovery{}
	result.ID = src.ID
	result.SourceID = src.SourceID
	result.Version = src.Version
	result.CatalogJSON = src.CatalogJSON
	result.DiscoveredAt = (src.DiscoveredAt).UTC()
	return result, nil
}

func convertCatalogDiscoveryFromDB(src *dbCatalogDiscovery) (*pipelineservice.CatalogDiscovery, error) {
	result := &pipelineservice.CatalogDiscovery{}
	result.ID = src.ID
	result.SourceID = src.SourceID
	result.Version = src.Version
	result.CatalogJSON = src.CatalogJSON
	result.DiscoveredAt = src.DiscoveredAt
	return result, nil
}
func (a dbCatalogDiscovery) TableName() string {
	return "pipeline.catalog_discoveries"
}

type dbConfiguredCatalog struct {
	ID           uuid.UUID `gorm:"column:id;"`
	ConnectionID uuid.UUID `gorm:"column:connection_id;"`
	StreamsJSON  jsonb     `gorm:"column:streams_json;"`
	CreatedAt    time.Time `gorm:"column:created_at;"`
	UpdatedAt    time.Time `gorm:"column:updated_at;"`
}

func convertConfiguredCatalogToDB(src *pipelineservice.ConfiguredCatalog) (*dbConfiguredCatalog, error) {
	result := &dbConfiguredCatalog{}
	result.ID = src.ID
	result.ConnectionID = src.ConnectionID
	result.StreamsJSON = src.StreamsJSON
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	return result, nil
}

func convertConfiguredCatalogFromDB(src *dbConfiguredCatalog) (*pipelineservice.ConfiguredCatalog, error) {
	result := &pipelineservice.ConfiguredCatalog{}
	result.ID = src.ID
	result.ConnectionID = src.ConnectionID
	result.StreamsJSON = src.StreamsJSON
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	return result, nil
}
func (a dbConfiguredCatalog) TableName() string {
	return "pipeline.configured_catalogs"
}

type dbJobLog struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	JobID     uuid.UUID `gorm:"column:job_id;"`
	LogLine   string    `gorm:"column:log_line;type:text;"`
	CreatedAt time.Time `gorm:"column:created_at;"`
}

func convertJobLogToDB(src *pipelineservice.JobLog) (*dbJobLog, error) {
	result := &dbJobLog{}
	result.ID = src.ID
	result.JobID = src.JobID
	result.LogLine = src.LogLine
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertJobLogFromDB(src *dbJobLog) (*pipelineservice.JobLog, error) {
	result := &pipelineservice.JobLog{}
	result.ID = src.ID
	result.JobID = src.JobID
	result.LogLine = src.LogLine
	result.CreatedAt = src.CreatedAt
	return result, nil
}
func (a dbJobLog) TableName() string {
	return "pipeline.job_logs"
}

type dbConnectionState struct {
	ConnectionID uuid.UUID `gorm:"column:connection_id;primaryKey"`
	StateType    string    `gorm:"column:state_type;type:text;"`
	StateBlob    jsonb     `gorm:"column:state_blob;"`
	UpdatedAt    time.Time `gorm:"column:updated_at;"`
}

func convertConnectionStateToDB(src *pipelineservice.ConnectionState) (*dbConnectionState, error) {
	result := &dbConnectionState{}
	result.ConnectionID = src.ConnectionID
	result.StateType = src.StateType
	result.StateBlob = src.StateBlob
	result.UpdatedAt = (src.UpdatedAt).UTC()
	return result, nil
}

func convertConnectionStateFromDB(src *dbConnectionState) (*pipelineservice.ConnectionState, error) {
	result := &pipelineservice.ConnectionState{}
	result.ConnectionID = src.ConnectionID
	result.StateType = src.StateType
	result.StateBlob = src.StateBlob
	result.UpdatedAt = src.UpdatedAt
	return result, nil
}
func (a dbConnectionState) TableName() string {
	return "pipeline.connection_state"
}

type dbWorkspaceSettings struct {
	WorkspaceID         uuid.UUID `gorm:"column:workspace_id;primaryKey"`
	MaxJobsPerWorkspace int       `gorm:"column:max_jobs_per_workspace;"`
	CreatedAt           time.Time `gorm:"column:created_at;"`
	UpdatedAt           time.Time `gorm:"column:updated_at;"`
}

func convertWorkspaceSettingsToDB(src *pipelineservice.WorkspaceSettings) (*dbWorkspaceSettings, error) {
	result := &dbWorkspaceSettings{}
	result.WorkspaceID = src.WorkspaceID
	result.MaxJobsPerWorkspace = src.MaxJobsPerWorkspace
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	return result, nil
}

func convertWorkspaceSettingsFromDB(src *dbWorkspaceSettings) (*pipelineservice.WorkspaceSettings, error) {
	result := &pipelineservice.WorkspaceSettings{}
	result.WorkspaceID = src.WorkspaceID
	result.MaxJobsPerWorkspace = src.MaxJobsPerWorkspace
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	return result, nil
}
func (a dbWorkspaceSettings) TableName() string {
	return "pipeline.workspace_settings"
}

type dbConnectorTask struct {
	ID           uuid.UUID                 `gorm:"column:id;"`
	WorkspaceID  uuid.UUID                 `gorm:"column:workspace_id;"`
	TaskType     string                    `gorm:"column:task_type;type:text;"`
	Status       string                    `gorm:"column:status;type:text;"`
	Payload      *jsonConnectorTaskPayload `gorm:"column:payload;"`
	Result       *jsonConnectorTaskResult  `gorm:"column:result;"`
	ErrorMessage *string                   `gorm:"column:error_message;type:text;"`
	WorkerID     *string                   `gorm:"column:worker_id;type:text;"`
	CreatedAt    time.Time                 `gorm:"column:created_at;"`
	UpdatedAt    time.Time                 `gorm:"column:updated_at;"`
	CompletedAt  *time.Time                `gorm:"column:completed_at;"`
}

func convertConnectorTaskToDB(src *pipelineservice.ConnectorTask) (*dbConnectorTask, error) {
	result := &dbConnectorTask{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	tmp2, err := convertConnectorTaskTypeToDB(src.TaskType)
	if err != nil {
		return nil, err
	}
	result.TaskType = tmp2
	tmp3, err := convertConnectorTaskStatusToDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp3
	tmp4, err := convertConnectorTaskPayloadToDB(src.Payload)
	if err != nil {
		return nil, err
	}
	result.Payload = tmp4
	if src.Result != nil {
		tmp5, err := convertConnectorTaskResultToDB(*src.Result)
		if err != nil {
			return nil, err
		}
		result.Result = tmp5
	} else {
		result.Result = nil
	}
	if src.ErrorMessage == nil {
		result.ErrorMessage = nil
	} else {
		result.ErrorMessage = toPtr(fromPtr(src.ErrorMessage))
	}
	if src.WorkerID == nil {
		result.WorkerID = nil
	} else {
		result.WorkerID = toPtr(fromPtr(src.WorkerID))
	}
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	if src.CompletedAt == nil {
		result.CompletedAt = nil
	} else {
		result.CompletedAt = toPtr((fromPtr(src.CompletedAt)).UTC())
	}
	return result, nil
}

func convertConnectorTaskFromDB(src *dbConnectorTask) (*pipelineservice.ConnectorTask, error) {
	result := &pipelineservice.ConnectorTask{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	tmp16, err := convertConnectorTaskTypeFromDB(src.TaskType)
	if err != nil {
		return nil, err
	}
	result.TaskType = tmp16
	tmp17, err := convertConnectorTaskStatusFromDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp17
	tmp18, err := convertConnectorTaskPayloadFromDB(src.Payload)
	if err != nil {
		return nil, fmt.Errorf("convert ConnectorTaskPayload to service type: %w", err)
	}
	result.Payload = tmp18
	if src.Result != nil {
		tmp19, err := convertConnectorTaskResultFromDB(src.Result)
		if err != nil {
			return nil, fmt.Errorf("convert ConnectorTaskResult to service type: %w", err)
		}
		result.Result = toPtr(tmp19)
	} else {
		result.Result = nil
	}
	if src.ErrorMessage == nil {
		result.ErrorMessage = nil
	} else {
		result.ErrorMessage = toPtr(fromPtr(src.ErrorMessage))
	}
	if src.WorkerID == nil {
		result.WorkerID = nil
	} else {
		result.WorkerID = toPtr(fromPtr(src.WorkerID))
	}
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	if src.CompletedAt == nil {
		result.CompletedAt = nil
	} else {
		result.CompletedAt = toPtr(fromPtr(src.CompletedAt))
	}
	return result, nil
}
func (a dbConnectorTask) TableName() string {
	return "pipeline.connector_tasks"
}

type dbStreamGeneration struct {
	ConnectionID    uuid.UUID `gorm:"column:connection_id;primaryKey"`
	StreamNamespace string    `gorm:"column:stream_namespace;type:text;primaryKey"`
	StreamName      string    `gorm:"column:stream_name;type:text;primaryKey"`
	GenerationID    int64     `gorm:"column:generation_id;"`
	UpdatedAt       time.Time `gorm:"column:updated_at;"`
}

func convertStreamGenerationToDB(src *pipelineservice.StreamGeneration) (*dbStreamGeneration, error) {
	result := &dbStreamGeneration{}
	result.ConnectionID = src.ConnectionID
	result.StreamNamespace = src.StreamNamespace
	result.StreamName = src.StreamName
	result.GenerationID = src.GenerationID
	result.UpdatedAt = (src.UpdatedAt).UTC()
	return result, nil
}

func convertStreamGenerationFromDB(src *dbStreamGeneration) (*pipelineservice.StreamGeneration, error) {
	result := &pipelineservice.StreamGeneration{}
	result.ConnectionID = src.ConnectionID
	result.StreamNamespace = src.StreamNamespace
	result.StreamName = src.StreamName
	result.GenerationID = src.GenerationID
	result.UpdatedAt = src.UpdatedAt
	return result, nil
}
func (a dbStreamGeneration) TableName() string {
	return "pipeline.stream_generations"
}
