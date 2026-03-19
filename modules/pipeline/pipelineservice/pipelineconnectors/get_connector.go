package pipelineconnectors

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetConnector retrieves a managed connector by ID and workspace.
type GetConnector struct {
	storage pipelineservice.Storage
}

// NewGetConnector creates a new GetConnector use case.
func NewGetConnector(storage pipelineservice.Storage) *GetConnector {
	return &GetConnector{storage: storage}
}

// GetConnectorParams holds parameters for getting a connector.
type GetConnectorParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// Execute retrieves a managed connector by ID, scoped to a workspace.
// WorkspaceID is required to prevent cross-workspace access (IDOR).
func (uc *GetConnector) Execute(ctx context.Context, params GetConnectorParams) (*pipelineservice.ManagedConnector, error) {
	f := &pipelineservice.ManagedConnectorFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}

	mc, err := uc.storage.ManagedConnectors().First(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("getting managed connector: %w", err)
	}

	return mc, nil
}
