package pipelineconnect

import (
	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// protoConnectorTypeToDomain converts an executor proto ConnectorType to the domain enum.
func protoConnectorTypeToDomain(t executorv1.ConnectorType) pipelineservice.ConnectorType {
	switch t {
	case executorv1.ConnectorType_CONNECTOR_TYPE_SOURCE:
		return pipelineservice.ConnectorTypeSource
	case executorv1.ConnectorType_CONNECTOR_TYPE_DESTINATION:
		return pipelineservice.ConnectorTypeDestination
	default:
		return pipelineservice.ConnectorTypeSource
	}
}

// domainJobTypeToProto converts a domain JobType to the executor proto enum.
func domainJobTypeToProto(t pipelineservice.JobType) executorv1.JobType {
	switch t {
	case pipelineservice.JobTypeSync:
		return executorv1.JobType_JOB_TYPE_SYNC
	default:
		return executorv1.JobType_JOB_TYPE_UNSPECIFIED
	}
}

// domainNamespaceDefToProto converts a domain NamespaceDefinition to the executor proto enum.
func domainNamespaceDefToProto(n pipelineservice.NamespaceDefinition) executorv1.NamespaceDefinition {
	switch n {
	case pipelineservice.NamespaceDefinitionSource:
		return executorv1.NamespaceDefinition_NAMESPACE_DEFINITION_SOURCE
	case pipelineservice.NamespaceDefinitionDestination:
		return executorv1.NamespaceDefinition_NAMESPACE_DEFINITION_DESTINATION
	case pipelineservice.NamespaceDefinitionCustom:
		return executorv1.NamespaceDefinition_NAMESPACE_DEFINITION_CUSTOM
	default:
		return executorv1.NamespaceDefinition_NAMESPACE_DEFINITION_UNSPECIFIED
	}
}

// domainTaskTypeToProto converts a domain ConnectorTaskType to the pipeline proto enum.
func domainTaskTypeToProto(t pipelineservice.ConnectorTaskType) pipelinev1.ConnectorTaskType {
	switch t {
	case pipelineservice.ConnectorTaskTypeCheck:
		return pipelinev1.ConnectorTaskType_CONNECTOR_TASK_TYPE_CHECK
	case pipelineservice.ConnectorTaskTypeSpec:
		return pipelinev1.ConnectorTaskType_CONNECTOR_TASK_TYPE_SPEC
	case pipelineservice.ConnectorTaskTypeDiscover:
		return pipelinev1.ConnectorTaskType_CONNECTOR_TASK_TYPE_DISCOVER
	default:
		return pipelinev1.ConnectorTaskType_CONNECTOR_TASK_TYPE_UNSPECIFIED
	}
}
