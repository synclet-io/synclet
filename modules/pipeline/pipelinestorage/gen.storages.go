package pipelinestorage

import (
	context "context"
	strconv "strconv"

	xxhash "github.com/cespare/xxhash"
	logging "github.com/go-pnp/go-pnp/logging"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	idempotency "github.com/saturn4er/boilerplate-go/lib/idempotency"
	txoutbox "github.com/saturn4er/boilerplate-go/lib/txoutbox"
	gorm "gorm.io/gorm"
	clause "gorm.io/gorm/clause"

	pipelinesvc "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	// user code 'imports'
	// end user code 'imports'
)

type Storages struct {
	db         *gorm.DB
	logger     *logging.Logger
	processors []txoutbox.MessageProcessor
}

var _ pipelinesvc.Storage = &Storages{}

func (s Storages) ManagedConnectors() pipelinesvc.ManagedConnectorsStorage {
	return NewManagedConnectorsStorage(s.db, s.logger)
}
func (s Storages) Repositorys() pipelinesvc.RepositorysStorage {
	return NewRepositorysStorage(s.db, s.logger)
}
func (s Storages) RepositoryConnectors() pipelinesvc.RepositoryConnectorsStorage {
	return NewRepositoryConnectorsStorage(s.db, s.logger)
}
func (s Storages) Sources() pipelinesvc.SourcesStorage {
	return NewSourcesStorage(s.db, s.logger)
}
func (s Storages) Destinations() pipelinesvc.DestinationsStorage {
	return NewDestinationsStorage(s.db, s.logger)
}
func (s Storages) Connections() pipelinesvc.ConnectionsStorage {
	return NewConnectionsStorage(s.db, s.logger)
}
func (s Storages) Jobs() pipelinesvc.JobsStorage {
	return NewJobsStorage(s.db, s.logger)
}
func (s Storages) JobAttempts() pipelinesvc.JobAttemptsStorage {
	return NewJobAttemptsStorage(s.db, s.logger)
}
func (s Storages) CatalogDiscoverys() pipelinesvc.CatalogDiscoverysStorage {
	return NewCatalogDiscoverysStorage(s.db, s.logger)
}
func (s Storages) ConfiguredCatalogs() pipelinesvc.ConfiguredCatalogsStorage {
	return NewConfiguredCatalogsStorage(s.db, s.logger)
}
func (s Storages) JobLogs() pipelinesvc.JobLogsStorage {
	return NewJobLogsStorage(s.db, s.logger)
}
func (s Storages) ConnectionStates() pipelinesvc.ConnectionStatesStorage {
	return NewConnectionStatesStorage(s.db, s.logger)
}

func (s Storages) WorkspaceSettingss() pipelinesvc.WorkspaceSettingssStorage {
	return NewWorkspaceSettingssStorage(s.db, s.logger)
}
func (s Storages) ConnectorTasks() pipelinesvc.ConnectorTasksStorage {
	return NewConnectorTasksStorage(s.db, s.logger)
}
func (s Storages) StreamGenerations() pipelinesvc.StreamGenerationsStorage {
	return NewStreamGenerationsStorage(s.db, s.logger)
}

func (s Storages) IdempotencyKeys() idempotency.Storage {
	return idempotency.GormStorage{
		DB: s.db,
	}

}

func (s *Storages) WithAdvisoryLock(ctx context.Context, scope string, lockID int64) error {
	hasher := xxhash.New()
	hasher.Write([]byte(scope))
	hasher.Write([]byte{':'})
	hasher.Write(strconv.AppendInt(nil, lockID, 10))

	result := s.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(?)", int64(hasher.Sum64()))
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (s Storages) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelinesvc.Storage) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return cb(ctx, &Storages{db: tx, logger: s.logger, processors: s.processors})
	})
}

func NewStorages(db *gorm.DB, logger *logging.Logger, processors []txoutbox.MessageProcessor) *Storages {
	return &Storages{db: db, logger: logger, processors: processors}
}

func NewManagedConnectorsStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.ManagedConnectorsStorage {
	return dbutil.GormEntityStorage[pipelinesvc.ManagedConnector, dbManagedConnector, pipelinesvc.ManagedConnectorFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapManagedConnectorQueryError,
		ConvertToInternal: convertManagedConnectorToDB,
		ConvertToExternal: convertManagedConnectorFromDB,
		BuildFilterExpression: func(filter *pipelinesvc.ManagedConnectorFilter) (clause.Expression, error) {
			return buildManagedConnectorFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			pipelinesvc.ManagedConnectorFieldID:            {Name: "id"},
			pipelinesvc.ManagedConnectorFieldWorkspaceID:   {Name: "workspace_id"},
			pipelinesvc.ManagedConnectorFieldDockerImage:   {Name: "docker_image"},
			pipelinesvc.ManagedConnectorFieldDockerTag:     {Name: "docker_tag"},
			pipelinesvc.ManagedConnectorFieldName:          {Name: "name"},
			pipelinesvc.ManagedConnectorFieldConnectorType: {Name: "connector_type"},
			pipelinesvc.ManagedConnectorFieldSpec:          {Name: "spec"},
			pipelinesvc.ManagedConnectorFieldCreatedAt:     {Name: "created_at"},
			pipelinesvc.ManagedConnectorFieldUpdatedAt:     {Name: "updated_at"},
			pipelinesvc.ManagedConnectorFieldRepositoryID:  {Name: "repository_id"},
		},
		LockScope: "pipeline.ManagedConnectors",
	}
}

func NewRepositorysStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.RepositorysStorage {
	return dbutil.GormEntityStorage[pipelinesvc.Repository, dbRepository, pipelinesvc.RepositoryFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapRepositoryQueryError,
		ConvertToInternal: convertRepositoryToDB,
		ConvertToExternal: convertRepositoryFromDB,
		BuildFilterExpression: func(filter *pipelinesvc.RepositoryFilter) (clause.Expression, error) {
			return buildRepositoryFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			pipelinesvc.RepositoryFieldID:             {Name: "id"},
			pipelinesvc.RepositoryFieldWorkspaceID:    {Name: "workspace_id"},
			pipelinesvc.RepositoryFieldName:           {Name: "name"},
			pipelinesvc.RepositoryFieldURL:            {Name: "url"},
			pipelinesvc.RepositoryFieldAuthHeader:     {Name: "auth_header"},
			pipelinesvc.RepositoryFieldStatus:         {Name: "status"},
			pipelinesvc.RepositoryFieldLastSyncedAt:   {Name: "last_synced_at"},
			pipelinesvc.RepositoryFieldConnectorCount: {Name: "connector_count"},
			pipelinesvc.RepositoryFieldLastError:      {Name: "last_error"},
			pipelinesvc.RepositoryFieldCreatedAt:      {Name: "created_at"},
			pipelinesvc.RepositoryFieldUpdatedAt:      {Name: "updated_at"},
		},
		LockScope: "pipeline.Repositorys",
	}
}

func NewRepositoryConnectorsStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.RepositoryConnectorsStorage {
	return dbutil.GormEntityStorage[pipelinesvc.RepositoryConnector, dbRepositoryConnector, pipelinesvc.RepositoryConnectorFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapRepositoryConnectorQueryError,
		ConvertToInternal: convertRepositoryConnectorToDB,
		ConvertToExternal: convertRepositoryConnectorFromDB,
		BuildFilterExpression: func(filter *pipelinesvc.RepositoryConnectorFilter) (clause.Expression, error) {
			return buildRepositoryConnectorFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			pipelinesvc.RepositoryConnectorFieldID:               {Name: "id"},
			pipelinesvc.RepositoryConnectorFieldRepositoryID:     {Name: "repository_id"},
			pipelinesvc.RepositoryConnectorFieldDockerRepository: {Name: "docker_repository"},
			pipelinesvc.RepositoryConnectorFieldDockerImageTag:   {Name: "docker_image_tag"},
			pipelinesvc.RepositoryConnectorFieldName:             {Name: "name"},
			pipelinesvc.RepositoryConnectorFieldConnectorType:    {Name: "connector_type"},
			pipelinesvc.RepositoryConnectorFieldDocumentationURL: {Name: "documentation_url"},
			pipelinesvc.RepositoryConnectorFieldReleaseStage:     {Name: "release_stage"},
			pipelinesvc.RepositoryConnectorFieldIconURL:          {Name: "icon_url"},
			pipelinesvc.RepositoryConnectorFieldSpec:             {Name: "spec"},
			pipelinesvc.RepositoryConnectorFieldSupportLevel:     {Name: "support_level"},
			pipelinesvc.RepositoryConnectorFieldLicense:          {Name: "license"},
			pipelinesvc.RepositoryConnectorFieldSourceType:       {Name: "source_type"},
			pipelinesvc.RepositoryConnectorFieldMetadata:         {Name: "metadata"},
		},
		LockScope: "pipeline.RepositoryConnectors",
	}
}

func NewSourcesStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.SourcesStorage {
	return dbutil.GormEntityStorage[pipelinesvc.Source, dbSource, pipelinesvc.SourceFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapSourceQueryError,
		ConvertToInternal: convertSourceToDB,
		ConvertToExternal: convertSourceFromDB,
		BuildFilterExpression: func(filter *pipelinesvc.SourceFilter) (clause.Expression, error) {
			return buildSourceFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			pipelinesvc.SourceFieldID:                 {Name: "id"},
			pipelinesvc.SourceFieldWorkspaceID:        {Name: "workspace_id"},
			pipelinesvc.SourceFieldName:               {Name: "name"},
			pipelinesvc.SourceFieldManagedConnectorID: {Name: "managed_connector_id"},
			pipelinesvc.SourceFieldConfig:             {Name: "config"},
			pipelinesvc.SourceFieldCreatedAt:          {Name: "created_at"},
			pipelinesvc.SourceFieldUpdatedAt:          {Name: "updated_at"},
			pipelinesvc.SourceFieldRuntimeConfig:      {Name: "runtime_config"},
		},
		LockScope: "pipeline.Sources",
	}
}

func NewDestinationsStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.DestinationsStorage {
	return dbutil.GormEntityStorage[pipelinesvc.Destination, dbDestination, pipelinesvc.DestinationFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapDestinationQueryError,
		ConvertToInternal: convertDestinationToDB,
		ConvertToExternal: convertDestinationFromDB,
		BuildFilterExpression: func(filter *pipelinesvc.DestinationFilter) (clause.Expression, error) {
			return buildDestinationFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			pipelinesvc.DestinationFieldID:                 {Name: "id"},
			pipelinesvc.DestinationFieldWorkspaceID:        {Name: "workspace_id"},
			pipelinesvc.DestinationFieldName:               {Name: "name"},
			pipelinesvc.DestinationFieldManagedConnectorID: {Name: "managed_connector_id"},
			pipelinesvc.DestinationFieldConfig:             {Name: "config"},
			pipelinesvc.DestinationFieldCreatedAt:          {Name: "created_at"},
			pipelinesvc.DestinationFieldUpdatedAt:          {Name: "updated_at"},
			pipelinesvc.DestinationFieldRuntimeConfig:      {Name: "runtime_config"},
		},
		LockScope: "pipeline.Destinations",
	}
}

type ConnectionsStorage struct {
	dbutil.GormEntityStorage[pipelinesvc.Connection, dbConnection, pipelinesvc.ConnectionFilter]
}

// user code 'Connection custom methods'
// end user code 'Connection custom methods'
func NewConnectionsStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.ConnectionsStorage {
	return &ConnectionsStorage{
		GormEntityStorage: dbutil.GormEntityStorage[pipelinesvc.Connection, dbConnection, pipelinesvc.ConnectionFilter]{
			Logger:            logger,
			DB:                db,
			DBErrorsWrapper:   wrapConnectionQueryError,
			ConvertToInternal: convertConnectionToDB,
			ConvertToExternal: convertConnectionFromDB,
			BuildFilterExpression: func(filter *pipelinesvc.ConnectionFilter) (clause.Expression, error) {
				return buildConnectionFilterExpr(filter)
			},
			FieldMapping: map[any]clause.Column{
				pipelinesvc.ConnectionFieldID:                    {Name: "id"},
				pipelinesvc.ConnectionFieldWorkspaceID:           {Name: "workspace_id"},
				pipelinesvc.ConnectionFieldName:                  {Name: "name"},
				pipelinesvc.ConnectionFieldStatus:                {Name: "status"},
				pipelinesvc.ConnectionFieldSourceID:              {Name: "source_id"},
				pipelinesvc.ConnectionFieldDestinationID:         {Name: "destination_id"},
				pipelinesvc.ConnectionFieldSchedule:              {Name: "schedule"},
				pipelinesvc.ConnectionFieldSchemaChangePolicy:    {Name: "schema_change_policy"},
				pipelinesvc.ConnectionFieldMaxAttempts:           {Name: "max_attempts"},
				pipelinesvc.ConnectionFieldNamespaceDefinition:   {Name: "namespace_definition"},
				pipelinesvc.ConnectionFieldCustomNamespaceFormat: {Name: "custom_namespace_format"},
				pipelinesvc.ConnectionFieldStreamPrefix:          {Name: "stream_prefix"},
				pipelinesvc.ConnectionFieldNextScheduledAt:       {Name: "next_scheduled_at"},
				pipelinesvc.ConnectionFieldCreatedAt:             {Name: "created_at"},
				pipelinesvc.ConnectionFieldUpdatedAt:             {Name: "updated_at"},
			},
			LockScope: "pipeline.Connections",
		},
		// user code 'Connection custom metods'
		// end user code 'Connection custom metods'
	}
}

type JobsStorage struct {
	dbutil.GormEntityStorage[pipelinesvc.Job, dbJob, pipelinesvc.JobFilter]
}

// user code 'Job custom methods'
// end user code 'Job custom methods'
func NewJobsStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.JobsStorage {
	return &JobsStorage{
		GormEntityStorage: dbutil.GormEntityStorage[pipelinesvc.Job, dbJob, pipelinesvc.JobFilter]{
			Logger:            logger,
			DB:                db,
			DBErrorsWrapper:   wrapJobQueryError,
			ConvertToInternal: convertJobToDB,
			ConvertToExternal: convertJobFromDB,
			BuildFilterExpression: func(filter *pipelinesvc.JobFilter) (clause.Expression, error) {
				return buildJobFilterExpr(filter)
			},
			FieldMapping: map[any]clause.Column{
				pipelinesvc.JobFieldID:            {Name: "id"},
				pipelinesvc.JobFieldConnectionID:  {Name: "connection_id"},
				pipelinesvc.JobFieldStatus:        {Name: "status"},
				pipelinesvc.JobFieldJobType:       {Name: "job_type"},
				pipelinesvc.JobFieldScheduledAt:   {Name: "scheduled_at"},
				pipelinesvc.JobFieldStartedAt:     {Name: "started_at"},
				pipelinesvc.JobFieldCompletedAt:   {Name: "completed_at"},
				pipelinesvc.JobFieldError:         {Name: "error"},
				pipelinesvc.JobFieldAttempt:       {Name: "attempt"},
				pipelinesvc.JobFieldMaxAttempts:   {Name: "max_attempts"},
				pipelinesvc.JobFieldWorkerID:      {Name: "worker_id"},
				pipelinesvc.JobFieldHeartbeatAt:   {Name: "heartbeat_at"},
				pipelinesvc.JobFieldK8sJobName:    {Name: "k8s_job_name"},
				pipelinesvc.JobFieldFailureReason: {Name: "failure_reason"},
				pipelinesvc.JobFieldCreatedAt:     {Name: "created_at"},
			},
			LockScope: "pipeline.Jobs",
		},
		// user code 'Job custom metods'
		// end user code 'Job custom metods'
	}
}

func NewJobAttemptsStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.JobAttemptsStorage {
	return dbutil.GormEntityStorage[pipelinesvc.JobAttempt, dbJobAttempt, pipelinesvc.JobAttemptFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapJobAttemptQueryError,
		ConvertToInternal: convertJobAttemptToDB,
		ConvertToExternal: convertJobAttemptFromDB,
		BuildFilterExpression: func(filter *pipelinesvc.JobAttemptFilter) (clause.Expression, error) {
			return buildJobAttemptFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			pipelinesvc.JobAttemptFieldID:            {Name: "id"},
			pipelinesvc.JobAttemptFieldJobID:         {Name: "job_id"},
			pipelinesvc.JobAttemptFieldAttemptNumber: {Name: "attempt_number"},
			pipelinesvc.JobAttemptFieldStartedAt:     {Name: "started_at"},
			pipelinesvc.JobAttemptFieldCompletedAt:   {Name: "completed_at"},
			pipelinesvc.JobAttemptFieldError:         {Name: "error"},
			pipelinesvc.JobAttemptFieldSyncStatsJSON: {Name: "sync_stats_json"},
		},
		LockScope: "pipeline.JobAttempts",
	}
}

func NewCatalogDiscoverysStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.CatalogDiscoverysStorage {
	return dbutil.GormEntityStorage[pipelinesvc.CatalogDiscovery, dbCatalogDiscovery, pipelinesvc.CatalogDiscoveryFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapCatalogDiscoveryQueryError,
		ConvertToInternal: convertCatalogDiscoveryToDB,
		ConvertToExternal: convertCatalogDiscoveryFromDB,
		BuildFilterExpression: func(filter *pipelinesvc.CatalogDiscoveryFilter) (clause.Expression, error) {
			return buildCatalogDiscoveryFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			pipelinesvc.CatalogDiscoveryFieldID:           {Name: "id"},
			pipelinesvc.CatalogDiscoveryFieldSourceID:     {Name: "source_id"},
			pipelinesvc.CatalogDiscoveryFieldVersion:      {Name: "version"},
			pipelinesvc.CatalogDiscoveryFieldCatalogJSON:  {Name: "catalog_json"},
			pipelinesvc.CatalogDiscoveryFieldDiscoveredAt: {Name: "discovered_at"},
		},
		LockScope: "pipeline.CatalogDiscoverys",
	}
}

func NewConfiguredCatalogsStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.ConfiguredCatalogsStorage {
	return dbutil.GormEntityStorage[pipelinesvc.ConfiguredCatalog, dbConfiguredCatalog, pipelinesvc.ConfiguredCatalogFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapConfiguredCatalogQueryError,
		ConvertToInternal: convertConfiguredCatalogToDB,
		ConvertToExternal: convertConfiguredCatalogFromDB,
		BuildFilterExpression: func(filter *pipelinesvc.ConfiguredCatalogFilter) (clause.Expression, error) {
			return buildConfiguredCatalogFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			pipelinesvc.ConfiguredCatalogFieldID:           {Name: "id"},
			pipelinesvc.ConfiguredCatalogFieldConnectionID: {Name: "connection_id"},
			pipelinesvc.ConfiguredCatalogFieldStreamsJSON:  {Name: "streams_json"},
			pipelinesvc.ConfiguredCatalogFieldCreatedAt:    {Name: "created_at"},
			pipelinesvc.ConfiguredCatalogFieldUpdatedAt:    {Name: "updated_at"},
		},
		LockScope: "pipeline.ConfiguredCatalogs",
	}
}

type JobLogsStorage struct {
	dbutil.GormEntityStorage[pipelinesvc.JobLog, dbJobLog, pipelinesvc.JobLogFilter]
}

// user code 'JobLog custom methods'
// end user code 'JobLog custom methods'
func NewJobLogsStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.JobLogsStorage {
	return &JobLogsStorage{
		GormEntityStorage: dbutil.GormEntityStorage[pipelinesvc.JobLog, dbJobLog, pipelinesvc.JobLogFilter]{
			Logger:            logger,
			DB:                db,
			DBErrorsWrapper:   wrapJobLogQueryError,
			ConvertToInternal: convertJobLogToDB,
			ConvertToExternal: convertJobLogFromDB,
			BuildFilterExpression: func(filter *pipelinesvc.JobLogFilter) (clause.Expression, error) {
				return buildJobLogFilterExpr(filter)
			},
			FieldMapping: map[any]clause.Column{
				pipelinesvc.JobLogFieldID:        {Name: "id"},
				pipelinesvc.JobLogFieldJobID:     {Name: "job_id"},
				pipelinesvc.JobLogFieldLogLine:   {Name: "log_line"},
				pipelinesvc.JobLogFieldCreatedAt: {Name: "created_at"},
			},
			LockScope: "pipeline.JobLogs",
		},
		// user code 'JobLog custom metods'
		// end user code 'JobLog custom metods'
	}
}

func NewConnectionStatesStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.ConnectionStatesStorage {
	return dbutil.GormEntityStorage[pipelinesvc.ConnectionState, dbConnectionState, pipelinesvc.ConnectionStateFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapConnectionStateQueryError,
		ConvertToInternal: convertConnectionStateToDB,
		ConvertToExternal: convertConnectionStateFromDB,
		BuildFilterExpression: func(filter *pipelinesvc.ConnectionStateFilter) (clause.Expression, error) {
			return buildConnectionStateFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			pipelinesvc.ConnectionStateFieldConnectionID: {Name: "connection_id"},
			pipelinesvc.ConnectionStateFieldStateType:    {Name: "state_type"},
			pipelinesvc.ConnectionStateFieldStateBlob:    {Name: "state_blob"},
			pipelinesvc.ConnectionStateFieldUpdatedAt:    {Name: "updated_at"},
		},
		LockScope: "pipeline.ConnectionStates",
	}
}

func NewWorkspaceSettingssStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.WorkspaceSettingssStorage {
	return dbutil.GormEntityStorage[pipelinesvc.WorkspaceSettings, dbWorkspaceSettings, pipelinesvc.WorkspaceSettingsFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapWorkspaceSettingsQueryError,
		ConvertToInternal: convertWorkspaceSettingsToDB,
		ConvertToExternal: convertWorkspaceSettingsFromDB,
		BuildFilterExpression: func(filter *pipelinesvc.WorkspaceSettingsFilter) (clause.Expression, error) {
			return buildWorkspaceSettingsFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			pipelinesvc.WorkspaceSettingsFieldWorkspaceID:         {Name: "workspace_id"},
			pipelinesvc.WorkspaceSettingsFieldMaxJobsPerWorkspace: {Name: "max_jobs_per_workspace"},
			pipelinesvc.WorkspaceSettingsFieldCreatedAt:           {Name: "created_at"},
			pipelinesvc.WorkspaceSettingsFieldUpdatedAt:           {Name: "updated_at"},
		},
		LockScope: "pipeline.WorkspaceSettingss",
	}
}

type ConnectorTasksStorage struct {
	dbutil.GormEntityStorage[pipelinesvc.ConnectorTask, dbConnectorTask, pipelinesvc.ConnectorTaskFilter]
}

// user code 'ConnectorTask custom methods'
// end user code 'ConnectorTask custom methods'
func NewConnectorTasksStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.ConnectorTasksStorage {
	return &ConnectorTasksStorage{
		GormEntityStorage: dbutil.GormEntityStorage[pipelinesvc.ConnectorTask, dbConnectorTask, pipelinesvc.ConnectorTaskFilter]{
			Logger:            logger,
			DB:                db,
			DBErrorsWrapper:   wrapConnectorTaskQueryError,
			ConvertToInternal: convertConnectorTaskToDB,
			ConvertToExternal: convertConnectorTaskFromDB,
			BuildFilterExpression: func(filter *pipelinesvc.ConnectorTaskFilter) (clause.Expression, error) {
				return buildConnectorTaskFilterExpr(filter)
			},
			FieldMapping: map[any]clause.Column{
				pipelinesvc.ConnectorTaskFieldID:           {Name: "id"},
				pipelinesvc.ConnectorTaskFieldWorkspaceID:  {Name: "workspace_id"},
				pipelinesvc.ConnectorTaskFieldTaskType:     {Name: "task_type"},
				pipelinesvc.ConnectorTaskFieldStatus:       {Name: "status"},
				pipelinesvc.ConnectorTaskFieldPayload:      {Name: "payload"},
				pipelinesvc.ConnectorTaskFieldResult:       {Name: "result"},
				pipelinesvc.ConnectorTaskFieldErrorMessage: {Name: "error_message"},
				pipelinesvc.ConnectorTaskFieldWorkerID:     {Name: "worker_id"},
				pipelinesvc.ConnectorTaskFieldCreatedAt:    {Name: "created_at"},
				pipelinesvc.ConnectorTaskFieldUpdatedAt:    {Name: "updated_at"},
				pipelinesvc.ConnectorTaskFieldCompletedAt:  {Name: "completed_at"},
			},
			LockScope: "pipeline.ConnectorTasks",
		},
		// user code 'ConnectorTask custom metods'
		// end user code 'ConnectorTask custom metods'
	}
}

func NewStreamGenerationsStorage(db *gorm.DB, logger *logging.Logger) pipelinesvc.StreamGenerationsStorage {
	return dbutil.GormEntityStorage[pipelinesvc.StreamGeneration, dbStreamGeneration, pipelinesvc.StreamGenerationFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapStreamGenerationQueryError,
		ConvertToInternal: convertStreamGenerationToDB,
		ConvertToExternal: convertStreamGenerationFromDB,
		BuildFilterExpression: func(filter *pipelinesvc.StreamGenerationFilter) (clause.Expression, error) {
			return buildStreamGenerationFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			pipelinesvc.StreamGenerationFieldConnectionID:    {Name: "connection_id"},
			pipelinesvc.StreamGenerationFieldStreamNamespace: {Name: "stream_namespace"},
			pipelinesvc.StreamGenerationFieldStreamName:      {Name: "stream_name"},
			pipelinesvc.StreamGenerationFieldGenerationID:    {Name: "generation_id"},
			pipelinesvc.StreamGenerationFieldUpdatedAt:       {Name: "updated_at"},
		},
		LockScope: "pipeline.StreamGenerations",
	}
}
