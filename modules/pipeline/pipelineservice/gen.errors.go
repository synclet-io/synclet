package pipelineservice

import (
	fmt "fmt"
	// user code 'imports'
	// end user code 'imports'
)

type NotFoundError string

func (n NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", string(n))
}

type AlreadyExistsError string

func (a AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s already exists", string(a))
}

const (
	ErrManagedConnectorNotFound      = NotFoundError("ManagedConnector")
	ErrManagedConnectorAlreadyExists = AlreadyExistsError("ManagedConnector")
)
const (
	ErrRepositoryNotFound      = NotFoundError("Repository")
	ErrRepositoryAlreadyExists = AlreadyExistsError("Repository")
)
const (
	ErrRepositoryConnectorNotFound      = NotFoundError("RepositoryConnector")
	ErrRepositoryConnectorAlreadyExists = AlreadyExistsError("RepositoryConnector")
)
const (
	ErrSourceNotFound      = NotFoundError("Source")
	ErrSourceAlreadyExists = AlreadyExistsError("Source")
)
const (
	ErrDestinationNotFound      = NotFoundError("Destination")
	ErrDestinationAlreadyExists = AlreadyExistsError("Destination")
)
const (
	ErrConnectionNotFound      = NotFoundError("Connection")
	ErrConnectionAlreadyExists = AlreadyExistsError("Connection")
)
const (
	ErrJobNotFound      = NotFoundError("Job")
	ErrJobAlreadyExists = AlreadyExistsError("Job")
)
const (
	ErrJobAttemptNotFound      = NotFoundError("JobAttempt")
	ErrJobAttemptAlreadyExists = AlreadyExistsError("JobAttempt")
)
const (
	ErrCatalogDiscoveryNotFound      = NotFoundError("CatalogDiscovery")
	ErrCatalogDiscoveryAlreadyExists = AlreadyExistsError("CatalogDiscovery")
)
const (
	ErrConfiguredCatalogNotFound      = NotFoundError("ConfiguredCatalog")
	ErrConfiguredCatalogAlreadyExists = AlreadyExistsError("ConfiguredCatalog")
)
const (
	ErrJobLogNotFound      = NotFoundError("JobLog")
	ErrJobLogAlreadyExists = AlreadyExistsError("JobLog")
)
const (
	ErrConnectionStateNotFound      = NotFoundError("ConnectionState")
	ErrConnectionStateAlreadyExists = AlreadyExistsError("ConnectionState")
)
const (
	ErrCheckPayloadNotFound      = NotFoundError("CheckPayload")
	ErrCheckPayloadAlreadyExists = AlreadyExistsError("CheckPayload")
)
const (
	ErrSpecPayloadNotFound      = NotFoundError("SpecPayload")
	ErrSpecPayloadAlreadyExists = AlreadyExistsError("SpecPayload")
)
const (
	ErrDiscoverPayloadNotFound      = NotFoundError("DiscoverPayload")
	ErrDiscoverPayloadAlreadyExists = AlreadyExistsError("DiscoverPayload")
)
const (
	ErrCheckResultNotFound      = NotFoundError("CheckResult")
	ErrCheckResultAlreadyExists = AlreadyExistsError("CheckResult")
)
const (
	ErrSpecResultNotFound      = NotFoundError("SpecResult")
	ErrSpecResultAlreadyExists = AlreadyExistsError("SpecResult")
)
const (
	ErrDiscoverResultNotFound      = NotFoundError("DiscoverResult")
	ErrDiscoverResultAlreadyExists = AlreadyExistsError("DiscoverResult")
)
const (
	ErrWorkspaceSettingsNotFound      = NotFoundError("WorkspaceSettings")
	ErrWorkspaceSettingsAlreadyExists = AlreadyExistsError("WorkspaceSettings")
)
const (
	ErrConnectorTaskNotFound      = NotFoundError("ConnectorTask")
	ErrConnectorTaskAlreadyExists = AlreadyExistsError("ConnectorTask")
)
const (
	ErrStreamGenerationNotFound      = NotFoundError("StreamGeneration")
	ErrStreamGenerationAlreadyExists = AlreadyExistsError("StreamGeneration")
)
