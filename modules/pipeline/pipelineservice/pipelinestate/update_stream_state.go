package pipelinestate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// UpdateStreamStateParams holds parameters for updating a specific stream's state.
type UpdateStreamStateParams struct {
	ConnectionID    uuid.UUID
	WorkspaceID     uuid.UUID
	StreamName      string
	StreamNamespace string
	StateData       string
}

// UpdateStreamState verifies workspace ownership and updates a stream's state.
type UpdateStreamState struct {
	getConnection *pipelineconnections.GetConnection
	saveSyncState *SaveSyncState
	audit         pipelineservice.AuditRecorder
}

// NewUpdateStreamState creates a new UpdateStreamState use case.
func NewUpdateStreamState(getConnection *pipelineconnections.GetConnection, saveSyncState *SaveSyncState, audit pipelineservice.AuditRecorder) *UpdateStreamState {
	return &UpdateStreamState{
		getConnection: getConnection,
		saveSyncState: saveSyncState,
		audit:         audit,
	}
}

// Execute verifies the connection, validates the JSON state data, and saves it.
func (uc *UpdateStreamState) Execute(ctx context.Context, params UpdateStreamStateParams) error {
	conn, err := uc.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          params.ConnectionID,
		WorkspaceID: params.WorkspaceID,
	})
	if err != nil {
		return fmt.Errorf("connection not found in workspace: %w", err)
	}

	if !json.Valid([]byte(params.StateData)) {
		return pipelineservice.ErrStateDataInvalidJSON
	}

	msg := &protocol.AirbyteStateMessage{
		Type: protocol.StateTypeStream,
		Stream: &protocol.AirbyteStreamState{
			StreamDescriptor: protocol.StreamDescriptor{
				Name:      params.StreamName,
				Namespace: params.StreamNamespace,
			},
			StreamState: json.RawMessage(params.StateData),
		},
	}

	if err := uc.saveSyncState.Execute(ctx, SaveSyncStateParams{
		ConnectionID: params.ConnectionID,
		StateMessage: msg,
	}); err != nil {
		return fmt.Errorf("saving stream state: %w", err)
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
		After:         map[string]any{"stream_state_set": stream},
	})

	return nil
}
