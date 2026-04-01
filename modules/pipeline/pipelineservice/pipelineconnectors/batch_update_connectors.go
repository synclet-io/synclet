package pipelineconnectors

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// BatchUpdateConnectors updates explicitly requested managed connectors in a workspace.
type BatchUpdateConnectors struct {
	updateConnector *UpdateManagedConnector
	storage         pipelineservice.Storage
}

// NewBatchUpdateConnectors creates a new BatchUpdateConnectors use case.
func NewBatchUpdateConnectors(updateConnector *UpdateManagedConnector, storage pipelineservice.Storage) *BatchUpdateConnectors {
	return &BatchUpdateConnectors{updateConnector: updateConnector, storage: storage}
}

// BatchUpdateConnectorsParams holds parameters for batch updating connectors.
type BatchUpdateConnectorsParams struct {
	WorkspaceID  uuid.UUID
	ConnectorIDs []uuid.UUID
}

// BatchUpdateConnectorsResult holds the result of a batch update operation.
type BatchUpdateConnectorsResult struct {
	UpdatedCount      int
	UpdatedConnectors []*pipelineservice.ManagedConnector
}

// Execute updates only the explicitly requested managed connectors.
// If ConnectorIDs is empty, returns immediately with zero updates.
func (uc *BatchUpdateConnectors) Execute(ctx context.Context, params BatchUpdateConnectorsParams) (*BatchUpdateConnectorsResult, error) {
	if len(params.ConnectorIDs) == 0 {
		return &BatchUpdateConnectorsResult{}, nil
	}

	// Find only the requested managed connectors in this workspace.
	connectors, err := uc.storage.ManagedConnectors().Find(ctx, &pipelineservice.ManagedConnectorFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
		ID:          filter.In(params.ConnectorIDs...),
	})
	if err != nil {
		return nil, fmt.Errorf("listing managed connectors: %w", err)
	}

	var updated []*pipelineservice.ManagedConnector

	for _, connector := range connectors {
		if connector.RepositoryID == nil {
			continue // Skip custom connectors.
		}

		// Update via the single update use case.
		result, err := uc.updateConnector.Execute(ctx, UpdateManagedConnectorParams{
			ConnectorID: connector.ID,
			WorkspaceID: params.WorkspaceID,
		})
		if err != nil {
			continue // Skip failures, continue with others.
		}

		updated = append(updated, result)
	}

	return &BatchUpdateConnectorsResult{
		UpdatedCount:      len(updated),
		UpdatedConnectors: updated,
	}, nil
}
