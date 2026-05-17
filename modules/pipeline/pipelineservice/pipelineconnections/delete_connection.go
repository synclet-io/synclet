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
	audit   pipelineservice.AuditRecorder
}

// NewDeleteConnection creates a new DeleteConnection use case.
func NewDeleteConnection(storage pipelineservice.Storage, audit pipelineservice.AuditRecorder) *DeleteConnection {
	return &DeleteConnection{storage: storage, audit: audit}
}

// Execute deletes the connection matching the given ID and workspace.
func (uc *DeleteConnection) Execute(ctx context.Context, params DeleteConnectionParams) error {
	// Load the connection first so the audit record can capture its name and
	// pre-delete shape. Workspace-scoped lookup also doubles as the access
	// check.
	existing, lookupErr := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})

	err := uc.storage.Connections().Delete(ctx, &pipelineservice.ConnectionFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("deleting connection: %w", err)
	}

	if lookupErr == nil && existing != nil {
		uc.audit.Record(ctx, pipelineservice.AuditEvent{
			WorkspaceID:   params.WorkspaceID,
			Action:        "delete",
			ResourceType:  "connection",
			ResourceID:    existing.ID,
			ResourceLabel: existing.Name,
			Before:        connectionToAuditPayload(existing),
			After:         nil,
		})
	}

	return nil
}
