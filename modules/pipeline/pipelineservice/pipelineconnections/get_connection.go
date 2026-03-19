package pipelineconnections

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetConnectionParams holds parameters for getting a connection.
type GetConnectionParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// GetConnection retrieves a connection by ID within a workspace.
type GetConnection struct {
	storage pipelineservice.Storage
}

// NewGetConnection creates a new GetConnection use case.
func NewGetConnection(storage pipelineservice.Storage) *GetConnection {
	return &GetConnection{storage: storage}
}

// Execute returns the connection matching the given ID and workspace.
func (uc *GetConnection) Execute(ctx context.Context, params GetConnectionParams) (*pipelineservice.Connection, error) {
	conn, err := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting connection: %w", err)
	}

	return conn, nil
}
