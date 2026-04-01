package pipelineconnectors

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// DeleteConnector removes a managed connector.
type DeleteConnector struct {
	storage pipelineservice.Storage
}

// NewDeleteConnector creates a new DeleteConnector use case.
func NewDeleteConnector(storage pipelineservice.Storage) *DeleteConnector {
	return &DeleteConnector{storage: storage}
}

// DeleteConnectorParams holds parameters for deleting a connector.
type DeleteConnectorParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// Execute removes a managed connector by ID, scoped to workspace.
func (uc *DeleteConnector) Execute(ctx context.Context, params DeleteConnectorParams) error {
	connector, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("getting managed connector: %w", err)
	}

	var dependentCount int
	var dependentType string

	switch {
	case connector.ConnectorType.IsSource():
		dependentCount, err = uc.storage.Sources().Count(ctx, &pipelineservice.SourceFilter{
			ManagedConnectorID: filter.Equals(params.ID),
		})
		dependentType = "source(s)"
	case connector.ConnectorType.IsDestination():
		dependentCount, err = uc.storage.Destinations().Count(ctx, &pipelineservice.DestinationFilter{
			ManagedConnectorID: filter.Equals(params.ID),
		})
		dependentType = "destination(s)"
	}

	if err != nil {
		return fmt.Errorf("counting dependent resources: %w", err)
	}

	if dependentCount > 0 {
		return &pipelineservice.ValidationError{
			Message: fmt.Sprintf("cannot delete connector: used by %d %s", dependentCount, dependentType),
		}
	}

	if err := uc.storage.ManagedConnectors().Delete(ctx, &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}); err != nil {
		return fmt.Errorf("deleting managed connector: %w", err)
	}

	return nil
}
