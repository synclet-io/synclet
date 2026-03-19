package pipelinestate

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

// ResetConnectionStateParams holds parameters for resetting all state for a connection.
type ResetConnectionStateParams struct {
	ConnectionID uuid.UUID
	WorkspaceID  uuid.UUID
}

// ResetConnectionState verifies workspace ownership and clears all state for a connection.
type ResetConnectionState struct {
	getConnection  *pipelineconnections.GetConnection
	clearSyncState *ClearSyncState
}

// NewResetConnectionState creates a new ResetConnectionState use case.
func NewResetConnectionState(getConnection *pipelineconnections.GetConnection, clearSyncState *ClearSyncState) *ResetConnectionState {
	return &ResetConnectionState{
		getConnection:  getConnection,
		clearSyncState: clearSyncState,
	}
}

// Execute verifies the connection belongs to the workspace and clears all state.
func (uc *ResetConnectionState) Execute(ctx context.Context, params ResetConnectionStateParams) error {
	if _, err := uc.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          params.ConnectionID,
		WorkspaceID: params.WorkspaceID,
	}); err != nil {
		return fmt.Errorf("connection not found in workspace: %w", err)
	}

	if err := uc.clearSyncState.Execute(ctx, ClearSyncStateParams{
		ConnectionID: params.ConnectionID,
	}); err != nil {
		return fmt.Errorf("clearing connection state: %w", err)
	}

	return nil
}
