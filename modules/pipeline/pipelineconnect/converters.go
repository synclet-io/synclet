package pipelineconnect

import (
	"time"

	"github.com/google/uuid"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	registryv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnectors"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// Connection status

func connectionStatusToProto(s pipelineservice.ConnectionStatus) pipelinev1.ConnectionStatus {
	switch s {
	case pipelineservice.ConnectionStatusActive:
		return pipelinev1.ConnectionStatus_CONNECTION_STATUS_ACTIVE
	case pipelineservice.ConnectionStatusInactive:
		return pipelinev1.ConnectionStatus_CONNECTION_STATUS_INACTIVE
	case pipelineservice.ConnectionStatusPaused:
		return pipelinev1.ConnectionStatus_CONNECTION_STATUS_PAUSED
	default:
		return pipelinev1.ConnectionStatus_CONNECTION_STATUS_UNSPECIFIED
	}
}

// Schema change policy

func schemaChangePolicyToProto(s pipelineservice.SchemaChangePolicy) pipelinev1.SchemaChangePolicy {
	switch s {
	case pipelineservice.SchemaChangePolicyPropagate:
		return pipelinev1.SchemaChangePolicy_SCHEMA_CHANGE_POLICY_PROPAGATE
	case pipelineservice.SchemaChangePolicyIgnore:
		return pipelinev1.SchemaChangePolicy_SCHEMA_CHANGE_POLICY_IGNORE
	case pipelineservice.SchemaChangePolicyPause:
		return pipelinev1.SchemaChangePolicy_SCHEMA_CHANGE_POLICY_PAUSE
	default:
		return pipelinev1.SchemaChangePolicy_SCHEMA_CHANGE_POLICY_UNSPECIFIED
	}
}

func protoToSchemaChangePolicy(p pipelinev1.SchemaChangePolicy) pipelineservice.SchemaChangePolicy {
	switch p {
	case pipelinev1.SchemaChangePolicy_SCHEMA_CHANGE_POLICY_PROPAGATE:
		return pipelineservice.SchemaChangePolicyPropagate
	case pipelinev1.SchemaChangePolicy_SCHEMA_CHANGE_POLICY_IGNORE:
		return pipelineservice.SchemaChangePolicyIgnore
	case pipelinev1.SchemaChangePolicy_SCHEMA_CHANGE_POLICY_PAUSE:
		return pipelineservice.SchemaChangePolicyPause
	default:
		return 0
	}
}

// Job status

func jobStatusToProto(s pipelineservice.JobStatus) pipelinev1.JobStatus {
	switch s {
	case pipelineservice.JobStatusScheduled:
		return pipelinev1.JobStatus_JOB_STATUS_SCHEDULED
	case pipelineservice.JobStatusStarting:
		return pipelinev1.JobStatus_JOB_STATUS_STARTING
	case pipelineservice.JobStatusRunning:
		return pipelinev1.JobStatus_JOB_STATUS_RUNNING
	case pipelineservice.JobStatusCompleted:
		return pipelinev1.JobStatus_JOB_STATUS_COMPLETED
	case pipelineservice.JobStatusFailed:
		return pipelinev1.JobStatus_JOB_STATUS_FAILED
	case pipelineservice.JobStatusCancelled:
		return pipelinev1.JobStatus_JOB_STATUS_CANCELLED
	default:
		return pipelinev1.JobStatus_JOB_STATUS_UNSPECIFIED
	}
}

// Job type

func jobTypeToProto(t pipelineservice.JobType) pipelinev1.JobType {
	switch t {
	case pipelineservice.JobTypeSync:
		return pipelinev1.JobType_JOB_TYPE_SYNC
	case pipelineservice.JobTypeDiscover:
		return pipelinev1.JobType_JOB_TYPE_DISCOVER
	case pipelineservice.JobTypeCheck:
		return pipelinev1.JobType_JOB_TYPE_CHECK
	default:
		return pipelinev1.JobType_JOB_TYPE_UNSPECIFIED
	}
}

// Connector type

func managedConnectorTypeToProto(t pipelineservice.ConnectorType) registryv1.ConnectorType {
	switch t {
	case pipelineservice.ConnectorTypeSource:
		return registryv1.ConnectorType_CONNECTOR_TYPE_SOURCE
	case pipelineservice.ConnectorTypeDestination:
		return registryv1.ConnectorType_CONNECTOR_TYPE_DESTINATION
	default:
		return registryv1.ConnectorType_CONNECTOR_TYPE_UNSPECIFIED
	}
}

func protoToConnectorType(t registryv1.ConnectorType) pipelineservice.ConnectorType {
	switch t {
	case registryv1.ConnectorType_CONNECTOR_TYPE_SOURCE:
		return pipelineservice.ConnectorTypeSource
	case registryv1.ConnectorType_CONNECTOR_TYPE_DESTINATION:
		return pipelineservice.ConnectorTypeDestination
	default:
		return pipelineservice.ConnectorTypeSource
	}
}

func protoToSupportLevel(sl registryv1.SupportLevel) string {
	switch sl {
	case registryv1.SupportLevel_SUPPORT_LEVEL_COMMUNITY:
		return "community"
	case registryv1.SupportLevel_SUPPORT_LEVEL_CERTIFIED:
		return "certified"
	default:
		return ""
	}
}

func protoToLicense(l registryv1.License) string {
	switch l {
	case registryv1.License_LICENSE_ELV2:
		return "ELv2"
	case registryv1.License_LICENSE_MIT:
		return "MIT"
	default:
		return ""
	}
}

func protoToSourceType(st registryv1.SourceType) string {
	switch st {
	case registryv1.SourceType_SOURCE_TYPE_API:
		return "api"
	case registryv1.SourceType_SOURCE_TYPE_DATABASE:
		return "database"
	case registryv1.SourceType_SOURCE_TYPE_FILE:
		return "file"
	default:
		return ""
	}
}

func supportLevelToProto(sl pipelineservice.SupportLevel) registryv1.SupportLevel {
	switch sl {
	case pipelineservice.SupportLevelCommunity:
		return registryv1.SupportLevel_SUPPORT_LEVEL_COMMUNITY
	case pipelineservice.SupportLevelCertified:
		return registryv1.SupportLevel_SUPPORT_LEVEL_CERTIFIED
	default:
		return registryv1.SupportLevel_SUPPORT_LEVEL_UNSPECIFIED
	}
}

func releaseStageToProto(rs pipelineservice.ReleaseStage) registryv1.ReleaseStage {
	switch rs {
	case pipelineservice.ReleaseStageGenerallyAvailable:
		return registryv1.ReleaseStage_RELEASE_STAGE_GENERALLY_AVAILABLE
	case pipelineservice.ReleaseStageBeta:
		return registryv1.ReleaseStage_RELEASE_STAGE_BETA
	case pipelineservice.ReleaseStageAlpha:
		return registryv1.ReleaseStage_RELEASE_STAGE_ALPHA
	case pipelineservice.ReleaseStageCustom:
		return registryv1.ReleaseStage_RELEASE_STAGE_CUSTOM
	default:
		return registryv1.ReleaseStage_RELEASE_STAGE_UNSPECIFIED
	}
}

func licenseStringToProto(l string) registryv1.License {
	switch l {
	case "ELv2":
		return registryv1.License_LICENSE_ELV2
	case "MIT":
		return registryv1.License_LICENSE_MIT
	default:
		return registryv1.License_LICENSE_UNSPECIFIED
	}
}

func sourceTypeToProto(st pipelineservice.SourceType) registryv1.SourceType {
	switch st {
	case pipelineservice.SourceTypeAPI:
		return registryv1.SourceType_SOURCE_TYPE_API
	case pipelineservice.SourceTypeDatabase:
		return registryv1.SourceType_SOURCE_TYPE_DATABASE
	case pipelineservice.SourceTypeFile:
		return registryv1.SourceType_SOURCE_TYPE_FILE
	default:
		return registryv1.SourceType_SOURCE_TYPE_UNSPECIFIED
	}
}

// Repository converters

func repositoryToProto(repo *pipelineservice.Repository) *registryv1.Repository {
	lastSyncedAt := ""
	if repo.LastSyncedAt != nil {
		lastSyncedAt = repo.LastSyncedAt.Format(time.RFC3339)
	}

	lastError := ""
	if repo.LastError != nil {
		lastError = *repo.LastError
	}

	hasAuth := repo.AuthHeader != nil && *repo.AuthHeader != ""

	return &registryv1.Repository{
		Id:             repo.ID.String(),
		Name:           repo.Name,
		Url:            repo.URL,
		HasAuth:        hasAuth,
		Status:         repositoryStatusToProto(repo.Status),
		LastSyncedAt:   lastSyncedAt,
		ConnectorCount: int32(repo.ConnectorCount),
		LastError:      lastError,
	}
}

func repositoryStatusToProto(s pipelineservice.RepositoryStatus) registryv1.RepositoryStatus {
	switch s {
	case pipelineservice.RepositoryStatusSyncing:
		return registryv1.RepositoryStatus_REPOSITORY_STATUS_SYNCING
	case pipelineservice.RepositoryStatusSynced:
		return registryv1.RepositoryStatus_REPOSITORY_STATUS_SYNCED
	case pipelineservice.RepositoryStatusFailed:
		return registryv1.RepositoryStatus_REPOSITORY_STATUS_FAILED
	default:
		return registryv1.RepositoryStatus_REPOSITORY_STATUS_UNSPECIFIED
	}
}

// managedConnectorToProto converts a domain ManagedConnector to its proto representation.
func managedConnectorToProto(connector *pipelineservice.ManagedConnector) *registryv1.ManagedConnectorInfo {
	return &registryv1.ManagedConnectorInfo{
		Id:            connector.ID.String(),
		DockerImage:   connector.DockerImage,
		DockerTag:     connector.DockerTag,
		Name:          connector.Name,
		ConnectorType: managedConnectorTypeToProto(connector.ConnectorType),
		RepositoryId:  uuidPtrToString(connector.RepositoryID),
	}
}

// managedConnectorWithUpdateInfoToProto converts a domain ManagedConnectorWithUpdateInfo to proto.
func managedConnectorWithUpdateInfoToProto(item *pipelineconnectors.ManagedConnectorWithUpdateInfo) *registryv1.ManagedConnectorInfo {
	info := managedConnectorToProto(item.Connector)

	if item.UpdateInfo != nil {
		breakingChanges := make([]*registryv1.BreakingChange, len(item.UpdateInfo.BreakingChanges))
		for i, bc := range item.UpdateInfo.BreakingChanges {
			breakingChanges[i] = &registryv1.BreakingChange{
				Version:                   bc.Version,
				Message:                   bc.Message,
				MigrationDocumentationUrl: bc.MigrationDocumentationURL,
				UpgradeDeadline:           bc.UpgradeDeadline,
			}
		}

		info.UpdateInfo = &registryv1.UpdateInfo{
			AvailableVersion: item.UpdateInfo.AvailableVersion,
			HasUpdate:        item.UpdateInfo.HasUpdate,
			BreakingChanges:  breakingChanges,
		}
	}

	return info
}

func uuidPtrToString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}

	return id.String()
}

// Sync modes

func protoToSyncMode(m pipelinev1.SyncMode) protocol.SyncMode {
	switch m {
	case pipelinev1.SyncMode_SYNC_MODE_FULL_REFRESH:
		return protocol.SyncModeFullRefresh
	case pipelinev1.SyncMode_SYNC_MODE_INCREMENTAL:
		return protocol.SyncModeIncremental
	default:
		return protocol.SyncModeFullRefresh
	}
}

// Namespace definition

func namespaceDefinitionToProto(n pipelineservice.NamespaceDefinition) pipelinev1.NamespaceDefinition {
	switch n {
	case pipelineservice.NamespaceDefinitionSource:
		return pipelinev1.NamespaceDefinition_NAMESPACE_DEFINITION_SOURCE
	case pipelineservice.NamespaceDefinitionDestination:
		return pipelinev1.NamespaceDefinition_NAMESPACE_DEFINITION_DESTINATION
	case pipelineservice.NamespaceDefinitionCustom:
		return pipelinev1.NamespaceDefinition_NAMESPACE_DEFINITION_CUSTOM
	default:
		return pipelinev1.NamespaceDefinition_NAMESPACE_DEFINITION_UNSPECIFIED
	}
}

func protoToNamespaceDefinition(p pipelinev1.NamespaceDefinition) pipelineservice.NamespaceDefinition {
	switch p {
	case pipelinev1.NamespaceDefinition_NAMESPACE_DEFINITION_SOURCE:
		return pipelineservice.NamespaceDefinitionSource
	case pipelinev1.NamespaceDefinition_NAMESPACE_DEFINITION_DESTINATION:
		return pipelineservice.NamespaceDefinitionDestination
	case pipelinev1.NamespaceDefinition_NAMESPACE_DEFINITION_CUSTOM:
		return pipelineservice.NamespaceDefinitionCustom
	default:
		return pipelineservice.NamespaceDefinitionSource
	}
}

func protoToDestinationSyncMode(m pipelinev1.DestinationSyncMode) protocol.DestinationSyncMode {
	switch m {
	case pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_OVERWRITE:
		return protocol.DestinationSyncModeOverwrite
	case pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_APPEND:
		return protocol.DestinationSyncModeAppend
	case pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_APPEND_DEDUP:
		return protocol.DestinationSyncModeAppendDedup
	default:
		return protocol.DestinationSyncModeOverwrite
	}
}
