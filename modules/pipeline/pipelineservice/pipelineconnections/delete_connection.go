package pipelineconnections

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// DeleteConnectionParams holds parameters for deleting a connection.
type DeleteConnectionParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// DeleteConnection deletes a connection within a workspace.
type DeleteConnection struct {
	storage pipelineservice.Storage
}

// NewDeleteConnection creates a new DeleteConnection use case.
func NewDeleteConnection(storage pipelineservice.Storage) *DeleteConnection {
	return &DeleteConnection{storage: storage}
}

// Execute deletes the connection matching the given ID and workspace.
func (uc *DeleteConnection) Execute(ctx context.Context, params DeleteConnectionParams) error {
	err := uc.storage.Connections().Delete(ctx, &pipelineservice.ConnectionFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("deleting connection: %w", err)
	}

	return nil
}
