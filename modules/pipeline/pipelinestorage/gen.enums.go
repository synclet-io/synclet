package pipelinestorage

import (
	fmt "fmt"

	pipelineservice "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	// user code 'imports'
	// end user code 'imports'
)

const (
	schemaChangePolicyPropagate = "propagate"
	schemaChangePolicyIgnore    = "ignore"
	schemaChangePolicyPause     = "pause"
)

func convertSchemaChangePolicyToDB(schemaChangePolicyValue pipelineservice.SchemaChangePolicy) (string, error) {
	result, ok := map[pipelineservice.SchemaChangePolicy]string{
		pipelineservice.SchemaChangePolicyPropagate: schemaChangePolicyPropagate,
		pipelineservice.SchemaChangePolicyIgnore:    schemaChangePolicyIgnore,
		pipelineservice.SchemaChangePolicyPause:     schemaChangePolicyPause,
	}[schemaChangePolicyValue]
	if !ok {
		return "", fmt.Errorf("unknown SchemaChangePolicy value: %d", schemaChangePolicyValue)
	}

	return result, nil
}

func convertSchemaChangePolicyFromDB(schemaChangePolicyValue string) (pipelineservice.SchemaChangePolicy, error) {
	result, ok := map[string]pipelineservice.SchemaChangePolicy{
		schemaChangePolicyPropagate: pipelineservice.SchemaChangePolicyPropagate,
		schemaChangePolicyIgnore:    pipelineservice.SchemaChangePolicyIgnore,
		schemaChangePolicyPause:     pipelineservice.SchemaChangePolicyPause,
	}[schemaChangePolicyValue]
	if !ok {
		return 0, fmt.Errorf("unknown SchemaChangePolicy db value: %s", schemaChangePolicyValue)
	}

	return result, nil
}

const (
	connectionStatusActive   = "active"
	connectionStatusInactive = "inactive"
	connectionStatusPaused   = "paused"
)

func convertConnectionStatusToDB(connectionStatusValue pipelineservice.ConnectionStatus) (string, error) {
	result, ok := map[pipelineservice.ConnectionStatus]string{
		pipelineservice.ConnectionStatusActive:   connectionStatusActive,
		pipelineservice.ConnectionStatusInactive: connectionStatusInactive,
		pipelineservice.ConnectionStatusPaused:   connectionStatusPaused,
	}[connectionStatusValue]
	if !ok {
		return "", fmt.Errorf("unknown ConnectionStatus value: %d", connectionStatusValue)
	}

	return result, nil
}

func convertConnectionStatusFromDB(connectionStatusValue string) (pipelineservice.ConnectionStatus, error) {
	result, ok := map[string]pipelineservice.ConnectionStatus{
		connectionStatusActive:   pipelineservice.ConnectionStatusActive,
		connectionStatusInactive: pipelineservice.ConnectionStatusInactive,
		connectionStatusPaused:   pipelineservice.ConnectionStatusPaused,
	}[connectionStatusValue]
	if !ok {
		return 0, fmt.Errorf("unknown ConnectionStatus db value: %s", connectionStatusValue)
	}

	return result, nil
}

const (
	jobStatusScheduled = "scheduled"
	jobStatusStarting  = "starting"
	jobStatusRunning   = "running"
	jobStatusCompleted = "completed"
	jobStatusFailed    = "failed"
	jobStatusCancelled = "cancelled"
)

func convertJobStatusToDB(jobStatusValue pipelineservice.JobStatus) (string, error) {
	result, ok := map[pipelineservice.JobStatus]string{
		pipelineservice.JobStatusScheduled: jobStatusScheduled,
		pipelineservice.JobStatusStarting:  jobStatusStarting,
		pipelineservice.JobStatusRunning:   jobStatusRunning,
		pipelineservice.JobStatusCompleted: jobStatusCompleted,
		pipelineservice.JobStatusFailed:    jobStatusFailed,
		pipelineservice.JobStatusCancelled: jobStatusCancelled,
	}[jobStatusValue]
	if !ok {
		return "", fmt.Errorf("unknown JobStatus value: %d", jobStatusValue)
	}

	return result, nil
}

func convertJobStatusFromDB(jobStatusValue string) (pipelineservice.JobStatus, error) {
	result, ok := map[string]pipelineservice.JobStatus{
		jobStatusScheduled: pipelineservice.JobStatusScheduled,
		jobStatusStarting:  pipelineservice.JobStatusStarting,
		jobStatusRunning:   pipelineservice.JobStatusRunning,
		jobStatusCompleted: pipelineservice.JobStatusCompleted,
		jobStatusFailed:    pipelineservice.JobStatusFailed,
		jobStatusCancelled: pipelineservice.JobStatusCancelled,
	}[jobStatusValue]
	if !ok {
		return 0, fmt.Errorf("unknown JobStatus db value: %s", jobStatusValue)
	}

	return result, nil
}

const (
	jobTypeSync     = "sync"
	jobTypeDiscover = "discover"
	jobTypeCheck    = "check"
)

func convertJobTypeToDB(jobTypeValue pipelineservice.JobType) (string, error) {
	result, ok := map[pipelineservice.JobType]string{
		pipelineservice.JobTypeSync:     jobTypeSync,
		pipelineservice.JobTypeDiscover: jobTypeDiscover,
		pipelineservice.JobTypeCheck:    jobTypeCheck,
	}[jobTypeValue]
	if !ok {
		return "", fmt.Errorf("unknown JobType value: %d", jobTypeValue)
	}

	return result, nil
}

func convertJobTypeFromDB(jobTypeValue string) (pipelineservice.JobType, error) {
	result, ok := map[string]pipelineservice.JobType{
		jobTypeSync:     pipelineservice.JobTypeSync,
		jobTypeDiscover: pipelineservice.JobTypeDiscover,
		jobTypeCheck:    pipelineservice.JobTypeCheck,
	}[jobTypeValue]
	if !ok {
		return 0, fmt.Errorf("unknown JobType db value: %s", jobTypeValue)
	}

	return result, nil
}

const (
	namespaceDefinitionSource      = "source"
	namespaceDefinitionDestination = "destination"
	namespaceDefinitionCustom      = "custom"
)

func convertNamespaceDefinitionToDB(namespaceDefinitionValue pipelineservice.NamespaceDefinition) (string, error) {
	result, ok := map[pipelineservice.NamespaceDefinition]string{
		pipelineservice.NamespaceDefinitionSource:      namespaceDefinitionSource,
		pipelineservice.NamespaceDefinitionDestination: namespaceDefinitionDestination,
		pipelineservice.NamespaceDefinitionCustom:      namespaceDefinitionCustom,
	}[namespaceDefinitionValue]
	if !ok {
		return "", fmt.Errorf("unknown NamespaceDefinition value: %d", namespaceDefinitionValue)
	}

	return result, nil
}

func convertNamespaceDefinitionFromDB(namespaceDefinitionValue string) (pipelineservice.NamespaceDefinition, error) {
	result, ok := map[string]pipelineservice.NamespaceDefinition{
		namespaceDefinitionSource:      pipelineservice.NamespaceDefinitionSource,
		namespaceDefinitionDestination: pipelineservice.NamespaceDefinitionDestination,
		namespaceDefinitionCustom:      pipelineservice.NamespaceDefinitionCustom,
	}[namespaceDefinitionValue]
	if !ok {
		return 0, fmt.Errorf("unknown NamespaceDefinition db value: %s", namespaceDefinitionValue)
	}

	return result, nil
}

const (
	repositoryStatusSyncing = "syncing"
	repositoryStatusSynced  = "synced"
	repositoryStatusFailed  = "failed"
)

func convertRepositoryStatusToDB(repositoryStatusValue pipelineservice.RepositoryStatus) (string, error) {
	result, ok := map[pipelineservice.RepositoryStatus]string{
		pipelineservice.RepositoryStatusSyncing: repositoryStatusSyncing,
		pipelineservice.RepositoryStatusSynced:  repositoryStatusSynced,
		pipelineservice.RepositoryStatusFailed:  repositoryStatusFailed,
	}[repositoryStatusValue]
	if !ok {
		return "", fmt.Errorf("unknown RepositoryStatus value: %d", repositoryStatusValue)
	}

	return result, nil
}

func convertRepositoryStatusFromDB(repositoryStatusValue string) (pipelineservice.RepositoryStatus, error) {
	result, ok := map[string]pipelineservice.RepositoryStatus{
		repositoryStatusSyncing: pipelineservice.RepositoryStatusSyncing,
		repositoryStatusSynced:  pipelineservice.RepositoryStatusSynced,
		repositoryStatusFailed:  pipelineservice.RepositoryStatusFailed,
	}[repositoryStatusValue]
	if !ok {
		return 0, fmt.Errorf("unknown RepositoryStatus db value: %s", repositoryStatusValue)
	}

	return result, nil
}

const (
	connectorTaskTypeCheck    = "check"
	connectorTaskTypeSpec     = "spec"
	connectorTaskTypeDiscover = "discover"
)

func convertConnectorTaskTypeToDB(connectorTaskTypeValue pipelineservice.ConnectorTaskType) (string, error) {
	result, ok := map[pipelineservice.ConnectorTaskType]string{
		pipelineservice.ConnectorTaskTypeCheck:    connectorTaskTypeCheck,
		pipelineservice.ConnectorTaskTypeSpec:     connectorTaskTypeSpec,
		pipelineservice.ConnectorTaskTypeDiscover: connectorTaskTypeDiscover,
	}[connectorTaskTypeValue]
	if !ok {
		return "", fmt.Errorf("unknown ConnectorTaskType value: %d", connectorTaskTypeValue)
	}

	return result, nil
}

func convertConnectorTaskTypeFromDB(connectorTaskTypeValue string) (pipelineservice.ConnectorTaskType, error) {
	result, ok := map[string]pipelineservice.ConnectorTaskType{
		connectorTaskTypeCheck:    pipelineservice.ConnectorTaskTypeCheck,
		connectorTaskTypeSpec:     pipelineservice.ConnectorTaskTypeSpec,
		connectorTaskTypeDiscover: pipelineservice.ConnectorTaskTypeDiscover,
	}[connectorTaskTypeValue]
	if !ok {
		return 0, fmt.Errorf("unknown ConnectorTaskType db value: %s", connectorTaskTypeValue)
	}

	return result, nil
}

const (
	connectorTaskStatusPending   = "pending"
	connectorTaskStatusRunning   = "running"
	connectorTaskStatusCompleted = "completed"
	connectorTaskStatusFailed    = "failed"
)

func convertConnectorTaskStatusToDB(connectorTaskStatusValue pipelineservice.ConnectorTaskStatus) (string, error) {
	result, ok := map[pipelineservice.ConnectorTaskStatus]string{
		pipelineservice.ConnectorTaskStatusPending:   connectorTaskStatusPending,
		pipelineservice.ConnectorTaskStatusRunning:   connectorTaskStatusRunning,
		pipelineservice.ConnectorTaskStatusCompleted: connectorTaskStatusCompleted,
		pipelineservice.ConnectorTaskStatusFailed:    connectorTaskStatusFailed,
	}[connectorTaskStatusValue]
	if !ok {
		return "", fmt.Errorf("unknown ConnectorTaskStatus value: %d", connectorTaskStatusValue)
	}

	return result, nil
}

func convertConnectorTaskStatusFromDB(connectorTaskStatusValue string) (pipelineservice.ConnectorTaskStatus, error) {
	result, ok := map[string]pipelineservice.ConnectorTaskStatus{
		connectorTaskStatusPending:   pipelineservice.ConnectorTaskStatusPending,
		connectorTaskStatusRunning:   pipelineservice.ConnectorTaskStatusRunning,
		connectorTaskStatusCompleted: pipelineservice.ConnectorTaskStatusCompleted,
		connectorTaskStatusFailed:    pipelineservice.ConnectorTaskStatusFailed,
	}[connectorTaskStatusValue]
	if !ok {
		return 0, fmt.Errorf("unknown ConnectorTaskStatus db value: %s", connectorTaskStatusValue)
	}

	return result, nil
}

const (
	connectorTypeSource      = "source"
	connectorTypeDestination = "destination"
)

func convertConnectorTypeToDB(connectorTypeValue pipelineservice.ConnectorType) (string, error) {
	result, ok := map[pipelineservice.ConnectorType]string{
		pipelineservice.ConnectorTypeSource:      connectorTypeSource,
		pipelineservice.ConnectorTypeDestination: connectorTypeDestination,
	}[connectorTypeValue]
	if !ok {
		return "", fmt.Errorf("unknown ConnectorType value: %d", connectorTypeValue)
	}

	return result, nil
}

func convertConnectorTypeFromDB(connectorTypeValue string) (pipelineservice.ConnectorType, error) {
	result, ok := map[string]pipelineservice.ConnectorType{
		connectorTypeSource:      pipelineservice.ConnectorTypeSource,
		connectorTypeDestination: pipelineservice.ConnectorTypeDestination,
	}[connectorTypeValue]
	if !ok {
		return 0, fmt.Errorf("unknown ConnectorType db value: %s", connectorTypeValue)
	}

	return result, nil
}

const (
	supportLevelCommunity = "community"
	supportLevelCertified = "certified"
	supportLevelUnknown   = "unknown"
)

func convertSupportLevelToDB(supportLevelValue pipelineservice.SupportLevel) (string, error) {
	result, ok := map[pipelineservice.SupportLevel]string{
		pipelineservice.SupportLevelCommunity: supportLevelCommunity,
		pipelineservice.SupportLevelCertified: supportLevelCertified,
		pipelineservice.SupportLevelUnknown:   supportLevelUnknown,
	}[supportLevelValue]
	if !ok {
		return "", fmt.Errorf("unknown SupportLevel value: %d", supportLevelValue)
	}

	return result, nil
}

func convertSupportLevelFromDB(supportLevelValue string) (pipelineservice.SupportLevel, error) {
	result, ok := map[string]pipelineservice.SupportLevel{
		supportLevelCommunity: pipelineservice.SupportLevelCommunity,
		supportLevelCertified: pipelineservice.SupportLevelCertified,
		supportLevelUnknown:   pipelineservice.SupportLevelUnknown,
	}[supportLevelValue]
	if !ok {
		return 0, fmt.Errorf("unknown SupportLevel db value: %s", supportLevelValue)
	}

	return result, nil
}

const (
	sourceTypeAPI      = "api"
	sourceTypeDatabase = "database"
	sourceTypeFile     = "file"
	sourceTypeUnknown  = "unknown"
)

func convertSourceTypeToDB(sourceTypeValue pipelineservice.SourceType) (string, error) {
	result, ok := map[pipelineservice.SourceType]string{
		pipelineservice.SourceTypeAPI:      sourceTypeAPI,
		pipelineservice.SourceTypeDatabase: sourceTypeDatabase,
		pipelineservice.SourceTypeFile:     sourceTypeFile,
		pipelineservice.SourceTypeUnknown:  sourceTypeUnknown,
	}[sourceTypeValue]
	if !ok {
		return "", fmt.Errorf("unknown SourceType value: %d", sourceTypeValue)
	}

	return result, nil
}

func convertSourceTypeFromDB(sourceTypeValue string) (pipelineservice.SourceType, error) {
	result, ok := map[string]pipelineservice.SourceType{
		sourceTypeAPI:      pipelineservice.SourceTypeAPI,
		sourceTypeDatabase: pipelineservice.SourceTypeDatabase,
		sourceTypeFile:     pipelineservice.SourceTypeFile,
		sourceTypeUnknown:  pipelineservice.SourceTypeUnknown,
	}[sourceTypeValue]
	if !ok {
		return 0, fmt.Errorf("unknown SourceType db value: %s", sourceTypeValue)
	}

	return result, nil
}

const (
	releaseStageGenerallyAvailable = "generally_available"
	releaseStageBeta               = "beta"
	releaseStageAlpha              = "alpha"
	releaseStageCustom             = "custom"
	releaseStageUnknown            = "unknown"
)

func convertReleaseStageToDB(releaseStageValue pipelineservice.ReleaseStage) (string, error) {
	result, ok := map[pipelineservice.ReleaseStage]string{
		pipelineservice.ReleaseStageGenerallyAvailable: releaseStageGenerallyAvailable,
		pipelineservice.ReleaseStageBeta:               releaseStageBeta,
		pipelineservice.ReleaseStageAlpha:              releaseStageAlpha,
		pipelineservice.ReleaseStageCustom:             releaseStageCustom,
		pipelineservice.ReleaseStageUnknown:            releaseStageUnknown,
	}[releaseStageValue]
	if !ok {
		return "", fmt.Errorf("unknown ReleaseStage value: %d", releaseStageValue)
	}

	return result, nil
}

func convertReleaseStageFromDB(releaseStageValue string) (pipelineservice.ReleaseStage, error) {
	result, ok := map[string]pipelineservice.ReleaseStage{
		releaseStageGenerallyAvailable: pipelineservice.ReleaseStageGenerallyAvailable,
		releaseStageBeta:               pipelineservice.ReleaseStageBeta,
		releaseStageAlpha:              pipelineservice.ReleaseStageAlpha,
		releaseStageCustom:             pipelineservice.ReleaseStageCustom,
		releaseStageUnknown:            pipelineservice.ReleaseStageUnknown,
	}[releaseStageValue]
	if !ok {
		return 0, fmt.Errorf("unknown ReleaseStage db value: %s", releaseStageValue)
	}

	return result, nil
}

const (
	bucketSizeHourly = "hourly"
	bucketSizeDaily  = "daily"
)

func convertBucketSizeToDB(bucketSizeValue pipelineservice.BucketSize) (string, error) {
	result, ok := map[pipelineservice.BucketSize]string{
		pipelineservice.BucketSizeHourly: bucketSizeHourly,
		pipelineservice.BucketSizeDaily:  bucketSizeDaily,
	}[bucketSizeValue]
	if !ok {
		return "", fmt.Errorf("unknown BucketSize value: %d", bucketSizeValue)
	}

	return result, nil
}

func convertBucketSizeFromDB(bucketSizeValue string) (pipelineservice.BucketSize, error) {
	result, ok := map[string]pipelineservice.BucketSize{
		bucketSizeHourly: pipelineservice.BucketSizeHourly,
		bucketSizeDaily:  pipelineservice.BucketSizeDaily,
	}[bucketSizeValue]
	if !ok {
		return 0, fmt.Errorf("unknown BucketSize db value: %s", bucketSizeValue)
	}

	return result, nil
}

const (
	healthHealthy  = "healthy"
	healthWarning  = "warning"
	healthFailing  = "failing"
	healthDisabled = "disabled"
)

func convertHealthToDB(healthValue pipelineservice.Health) (string, error) {
	result, ok := map[pipelineservice.Health]string{
		pipelineservice.HealthHealthy:  healthHealthy,
		pipelineservice.HealthWarning:  healthWarning,
		pipelineservice.HealthFailing:  healthFailing,
		pipelineservice.HealthDisabled: healthDisabled,
	}[healthValue]
	if !ok {
		return "", fmt.Errorf("unknown Health value: %d", healthValue)
	}

	return result, nil
}

func convertHealthFromDB(healthValue string) (pipelineservice.Health, error) {
	result, ok := map[string]pipelineservice.Health{
		healthHealthy:  pipelineservice.HealthHealthy,
		healthWarning:  pipelineservice.HealthWarning,
		healthFailing:  pipelineservice.HealthFailing,
		healthDisabled: pipelineservice.HealthDisabled,
	}[healthValue]
	if !ok {
		return 0, fmt.Errorf("unknown Health db value: %s", healthValue)
	}

	return result, nil
}

const (
	failureCategoryTimeout        = "timeout"
	failureCategoryOOM            = "oom"
	failureCategoryConnector      = "connector"
	failureCategoryInfrastructure = "infrastructure"
	failureCategoryUnknown        = "unknown"
)

func convertFailureCategoryToDB(failureCategoryValue pipelineservice.FailureCategory) (string, error) {
	result, ok := map[pipelineservice.FailureCategory]string{
		pipelineservice.FailureCategoryTimeout:        failureCategoryTimeout,
		pipelineservice.FailureCategoryOOM:            failureCategoryOOM,
		pipelineservice.FailureCategoryConnector:      failureCategoryConnector,
		pipelineservice.FailureCategoryInfrastructure: failureCategoryInfrastructure,
		pipelineservice.FailureCategoryUnknown:        failureCategoryUnknown,
	}[failureCategoryValue]
	if !ok {
		return "", fmt.Errorf("unknown FailureCategory value: %d", failureCategoryValue)
	}

	return result, nil
}

func convertFailureCategoryFromDB(failureCategoryValue string) (pipelineservice.FailureCategory, error) {
	result, ok := map[string]pipelineservice.FailureCategory{
		failureCategoryTimeout:        pipelineservice.FailureCategoryTimeout,
		failureCategoryOOM:            pipelineservice.FailureCategoryOOM,
		failureCategoryConnector:      pipelineservice.FailureCategoryConnector,
		failureCategoryInfrastructure: pipelineservice.FailureCategoryInfrastructure,
		failureCategoryUnknown:        pipelineservice.FailureCategoryUnknown,
	}[failureCategoryValue]
	if !ok {
		return 0, fmt.Errorf("unknown FailureCategory db value: %s", failureCategoryValue)
	}

	return result, nil
}

const (
	syncStatusCompleted = "completed"
	syncStatusFailed    = "failed"
)

func convertSyncStatusToDB(syncStatusValue pipelineservice.SyncStatus) (string, error) {
	result, ok := map[pipelineservice.SyncStatus]string{
		pipelineservice.SyncStatusCompleted: syncStatusCompleted,
		pipelineservice.SyncStatusFailed:    syncStatusFailed,
	}[syncStatusValue]
	if !ok {
		return "", fmt.Errorf("unknown SyncStatus value: %d", syncStatusValue)
	}

	return result, nil
}

func convertSyncStatusFromDB(syncStatusValue string) (pipelineservice.SyncStatus, error) {
	result, ok := map[string]pipelineservice.SyncStatus{
		syncStatusCompleted: pipelineservice.SyncStatusCompleted,
		syncStatusFailed:    pipelineservice.SyncStatusFailed,
	}[syncStatusValue]
	if !ok {
		return 0, fmt.Errorf("unknown SyncStatus db value: %s", syncStatusValue)
	}

	return result, nil
}
