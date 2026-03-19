package pipelineconnectors

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListConnectors returns all managed connectors for a workspace.
type ListConnectors struct {
	storage pipelineservice.Storage
}

// NewListConnectors creates a new ListConnectors use case.
func NewListConnectors(storage pipelineservice.Storage) *ListConnectors {
	return &ListConnectors{storage: storage}
}

// ListConnectorsParams holds parameters for listing connectors.
type ListConnectorsParams struct {
	WorkspaceID uuid.UUID
}

// Execute returns all managed connectors for a workspace.
func (uc *ListConnectors) Execute(ctx context.Context, params ListConnectorsParams) ([]*pipelineservice.ManagedConnector, error) {
	connectors, err := uc.storage.ManagedConnectors().Find(ctx, &pipelineservice.ManagedConnectorFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing managed connectors: %w", err)
	}

	return connectors, nil
}
