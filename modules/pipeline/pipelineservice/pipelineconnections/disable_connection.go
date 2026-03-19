package pipelineconnections

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// DisableConnectionParams holds parameters for disabling a connection.
type DisableConnectionParams struct {
	ConnectionID uuid.UUID
	WorkspaceID  uuid.UUID
}

// DisableConnection verifies workspace ownership and disables a connection.
type DisableConnection struct {
	getConnection          *GetConnection
	updateConnectionStatus *UpdateConnectionStatus
}

// NewDisableConnection creates a new DisableConnection use case.
func NewDisableConnection(getConnection *GetConnection, updateConnectionStatus *UpdateConnectionStatus) *DisableConnection {
	return &DisableConnection{
		getConnection:          getConnection,
		updateConnectionStatus: updateConnectionStatus,
	}
}

// Execute verifies the connection belongs to the workspace and sets status to inactive.
func (uc *DisableConnection) Execute(ctx context.Context, params DisableConnectionParams) (*pipelineservice.Connection, error) {
	if _, err := uc.getConnection.Execute(ctx, GetConnectionParams{
		ID:          params.ConnectionID,
		WorkspaceID: params.WorkspaceID,
	}); err != nil {
		return nil, fmt.Errorf("connection not found in workspace: %w", err)
	}

	conn, err := uc.updateConnectionStatus.Execute(ctx, UpdateConnectionStatusParams{
		ID:     params.ConnectionID,
		Status: pipelineservice.ConnectionStatusInactive,
	})
	if err != nil {
		return nil, fmt.Errorf("disabling connection: %w", err)
	}

	return conn, nil
}
