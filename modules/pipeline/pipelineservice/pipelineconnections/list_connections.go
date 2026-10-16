package pipelineconnections

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListConnectionsParams holds parameters for listing connections.
type ListConnectionsParams struct {
	WorkspaceID   uuid.UUID
	SourceID      *uuid.UUID
	DestinationID *uuid.UUID
}

// ListConnections lists all connections within a workspace.
type ListConnections struct {
	storage pipelineservice.Storage
}

// NewListConnections creates a new ListConnections use case.
func NewListConnections(storage pipelineservice.Storage) *ListConnections {
	return &ListConnections{storage: storage}
}

// Execute returns the workspace's connections, narrowed to the given source
// and destination when set.
func (uc *ListConnections) Execute(ctx context.Context, params ListConnectionsParams) ([]*pipelineservice.Connection, error) {
	connFilter := &pipelineservice.ConnectionFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}
	if params.SourceID != nil {
		connFilter.SourceID = filter.Equals(*params.SourceID)
	}

	if params.DestinationID != nil {
		connFilter.DestinationID = filter.Equals(*params.DestinationID)
	}

	conns, err := uc.storage.Connections().Find(ctx, connFilter)
	if err != nil {
		return nil, fmt.Errorf("listing connections: %w", err)
	}

	return conns, nil
}
