package pipelinestate

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

// ResetStreamStateParams holds parameters for resetting a specific stream's state.
type ResetStreamStateParams struct {
	ConnectionID    uuid.UUID
	WorkspaceID     uuid.UUID
	StreamName      string
	StreamNamespace string
}

// ResetStreamState verifies workspace ownership and clears state for a specific stream.
type ResetStreamState struct {
	getConnection  *pipelineconnections.GetConnection
	clearSyncState *ClearSyncState
	audit          pipelineservice.AuditRecorder
}

// NewResetStreamState creates a new ResetStreamState use case.
func NewResetStreamState(getConnection *pipelineconnections.GetConnection, clearSyncState *ClearSyncState, audit pipelineservice.AuditRecorder) *ResetStreamState {
	return &ResetStreamState{
		getConnection:  getConnection,
		clearSyncState: clearSyncState,
		audit:          audit,
	}
}

// Execute verifies the connection belongs to the workspace and clears the stream's state.
func (uc *ResetStreamState) Execute(ctx context.Context, params ResetStreamStateParams) error {
	conn, err := uc.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          params.ConnectionID,
		WorkspaceID: params.WorkspaceID,
	})
	if err != nil {
		return fmt.Errorf("connection not found in workspace: %w", err)
	}

	if err := uc.clearSyncState.Execute(ctx, ClearSyncStateParams{
		ConnectionID:    params.ConnectionID,
		StreamNamespace: &params.StreamNamespace,
		StreamName:      &params.StreamName,
	}); err != nil {
		return fmt.Errorf("clearing stream state: %w", err)
	}

	stream := params.StreamName
	if params.StreamNamespace != "" {
		stream = params.StreamNamespace + "." + params.StreamName
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "update",
		ResourceType:  "connection",
		ResourceID:    conn.ID,
		ResourceLabel: conn.Name,
		After:         map[string]any{"state_reset": stream},
	})

	return nil
}
