package pipelineconnections

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
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

// Execute deletes the connection matching the given ID and workspace. It
// returns a not-found error when no such connection exists, and
// ErrConnectionInWorkflow while the connection is a step of a workflow.
func (uc *DeleteConnection) Execute(ctx context.Context, params DeleteConnectionParams) error {
	var existing *pipelineservice.Connection

	// The row lock makes a concurrent step insert, which takes a key-share
	// lock on the connection, either finish before the count or wait for the
	// delete.
	if err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		conn, err := tx.Connections().First(ctx, &pipelineservice.ConnectionFilter{
			ID:          filter.Equals(params.ID),
			WorkspaceID: filter.Equals(params.WorkspaceID),
		}, dbutil.WithForUpdate())
		if err != nil {
			return fmt.Errorf("getting connection: %w", err)
		}

		steps, err := tx.WorkflowSteps().Count(ctx, &pipelineservice.WorkflowStepFilter{
			ConnectionID: filter.Equals(conn.ID),
		})
		if err != nil {
			return fmt.Errorf("checking workflow steps of connection: %w", err)
		}

		if steps > 0 {
			return pipelineservice.ErrConnectionInWorkflow
		}

		if err := tx.Connections().Delete(ctx, &pipelineservice.ConnectionFilter{
			ID:          filter.Equals(conn.ID),
			WorkspaceID: filter.Equals(params.WorkspaceID),
		}); err != nil {
			return fmt.Errorf("deleting connection: %w", err)
		}

		existing = conn

		return nil
	}); err != nil {
		return err
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "delete",
		ResourceType:  "connection",
		ResourceID:    existing.ID,
		ResourceLabel: existing.Name,
		Before:        connectionToAuditPayload(existing),
		After:         nil,
	})

	return nil
}
