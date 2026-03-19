package pipelinecatalog

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// GetDiscoveredCatalogForConnectionParams holds parameters for getting a discovered catalog via connection.
type GetDiscoveredCatalogForConnectionParams struct {
	ConnectionID uuid.UUID
	WorkspaceID  uuid.UUID
}

// GetDiscoveredCatalogForConnection resolves the source from a connection and reads the cached catalog.
type GetDiscoveredCatalogForConnection struct {
	getConnection *pipelineconnections.GetConnection
	storage       pipelineservice.Storage
}

// NewGetDiscoveredCatalogForConnection creates a new GetDiscoveredCatalogForConnection use case.
func NewGetDiscoveredCatalogForConnection(
	getConnection *pipelineconnections.GetConnection,
	storage pipelineservice.Storage,
) *GetDiscoveredCatalogForConnection {
	return &GetDiscoveredCatalogForConnection{
		getConnection: getConnection,
		storage:       storage,
	}
}

// Execute resolves connection -> source -> read cached catalog from CatalogDiscovery.
func (uc *GetDiscoveredCatalogForConnection) Execute(ctx context.Context, params GetDiscoveredCatalogForConnectionParams) (*protocol.AirbyteCatalog, error) {
	conn, err := uc.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          params.ConnectionID,
		WorkspaceID: params.WorkspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("getting connection: %w", err)
	}

	// Read cached catalog instead of running Docker.
	latest, err := uc.storage.CatalogDiscoverys().First(ctx, &pipelineservice.CatalogDiscoveryFilter{
		SourceID: filter.Equals(conn.SourceID),
	}, dbutil.WithOrder(pipelineservice.CatalogDiscoveryFieldVersion, dbutil.OrderDirDesc))
	if err != nil {
		return nil, fmt.Errorf("no cached catalog for source: %w", err)
	}

	var catalog protocol.AirbyteCatalog
	if err := json.Unmarshal([]byte(latest.CatalogJSON), &catalog); err != nil {
		return nil, fmt.Errorf("parsing cached catalog: %w", err)
	}

	return &catalog, nil
}
