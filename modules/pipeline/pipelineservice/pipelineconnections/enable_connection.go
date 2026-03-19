package pipelineconnections

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// EnableConnectionParams holds parameters for enabling a connection.
type EnableConnectionParams struct {
	ConnectionID uuid.UUID
	WorkspaceID  uuid.UUID
}

// EnableConnection verifies workspace ownership and enables a connection.
type EnableConnection struct {
	getConnection          *GetConnection
	updateConnectionStatus *UpdateConnectionStatus
}

// NewEnableConnection creates a new EnableConnection use case.
func NewEnableConnection(getConnection *GetConnection, updateConnectionStatus *UpdateConnectionStatus) *EnableConnection {
	return &EnableConnection{
		getConnection:          getConnection,
		updateConnectionStatus: updateConnectionStatus,
	}
}

// Execute verifies the connection belongs to the workspace and sets status to active.
func (uc *EnableConnection) Execute(ctx context.Context, params EnableConnectionParams) (*pipelineservice.Connection, error) {
	if _, err := uc.getConnection.Execute(ctx, GetConnectionParams{
		ID:          params.ConnectionID,
		WorkspaceID: params.WorkspaceID,
	}); err != nil {
		return nil, fmt.Errorf("connection not found in workspace: %w", err)
	}

	conn, err := uc.updateConnectionStatus.Execute(ctx, UpdateConnectionStatusParams{
		ID:     params.ConnectionID,
		Status: pipelineservice.ConnectionStatusActive,
	})
	if err != nil {
		return nil, fmt.Errorf("enabling connection: %w", err)
	}

	return conn, nil
}
