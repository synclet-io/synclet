package pipelineconnectors

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// UpdateManagedConnector updates a managed connector's docker_tag and spec
// to the latest from its repository connector. No Docker pull needed (D-09).
type UpdateManagedConnector struct {
	storage pipelineservice.Storage
}

// NewUpdateManagedConnector creates a new UpdateManagedConnector use case.
func NewUpdateManagedConnector(storage pipelineservice.Storage) *UpdateManagedConnector {
	return &UpdateManagedConnector{storage: storage}
}

// UpdateManagedConnectorParams holds parameters for updating a managed connector.
type UpdateManagedConnectorParams struct {
	ConnectorID uuid.UUID
	WorkspaceID uuid.UUID
}

// Execute loads the managed connector, finds the latest version from its repository,
// and updates the connector's docker_tag and spec.
func (uc *UpdateManagedConnector) Execute(ctx context.Context, params UpdateManagedConnectorParams) (*pipelineservice.ManagedConnector, error) {
	// 1. Load managed connector (workspace-scoped).
	connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(params.ConnectorID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading managed connector: %w", err)
	}

	// 2. Verify it has a repository_id (custom connectors cannot be updated this way).
	if connector.RepositoryID == nil {
		return nil, pipelineservice.ErrConnectorNotLinked
	}

	// 3. Find matching repository_connector by docker_image within the same repository.
	repoConn, err := uc.storage.RepositoryConnectors().First(ctx, &pipelineservice.RepositoryConnectorFilter{
		RepositoryID:     filter.Equals(*connector.RepositoryID),
		DockerRepository: filter.Equals(connector.DockerImage),
	})
	if err != nil {
		return nil, fmt.Errorf("loading repository connector: %w", err)
	}

	// 4. Update managed connector.
	connector.DockerTag = repoConn.DockerImageTag

	spec := repoConn.Spec
	if spec == "" {
		spec = "{}"
	}

	connector.Spec = spec
	connector.UpdatedAt = time.Now()

	updated, err := uc.storage.ManagedConnectors().Update(ctx, connector)
	if err != nil {
		return nil, fmt.Errorf("updating managed connector: %w", err)
	}

	return updated, nil
}
