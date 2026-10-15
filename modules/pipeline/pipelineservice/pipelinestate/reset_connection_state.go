package pipelinestate

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
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
	audit          pipelineservice.AuditRecorder
}

// NewResetConnectionState creates a new ResetConnectionState use case.
func NewResetConnectionState(getConnection *pipelineconnections.GetConnection, clearSyncState *ClearSyncState, audit pipelineservice.AuditRecorder) *ResetConnectionState {
	return &ResetConnectionState{
		getConnection:  getConnection,
		clearSyncState: clearSyncState,
		audit:          audit,
	}
}

// Execute verifies the connection belongs to the workspace and clears all state.
func (uc *ResetConnectionState) Execute(ctx context.Context, params ResetConnectionStateParams) error {
	conn, err := uc.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          params.ConnectionID,
		WorkspaceID: params.WorkspaceID,
	})
	if err != nil {
		return fmt.Errorf("connection not found in workspace: %w", err)
	}

	if err := uc.clearSyncState.Execute(ctx, ClearSyncStateParams{
		ConnectionID: params.ConnectionID,
	}); err != nil {
		return fmt.Errorf("clearing connection state: %w", err)
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "update",
		ResourceType:  "connection",
		ResourceID:    conn.ID,
		ResourceLabel: conn.Name,
		After:         map[string]any{"state_reset": "connection"},
	})

	return nil
}
