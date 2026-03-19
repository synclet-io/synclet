package pipelinestate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// GetSyncStateParams holds parameters for retrieving sync state.
type GetSyncStateParams struct {
	ConnectionID uuid.UUID
}

// GetSyncState retrieves the raw state blob for a connection.
// Returns the blob as json.RawMessage suitable for passing to connectors.
type GetSyncState struct {
	storage pipelineservice.Storage
}

// NewGetSyncState creates a new GetSyncState use case.
func NewGetSyncState(storage pipelineservice.Storage) *GetSyncState {
	return &GetSyncState{storage: storage}
}

// Execute returns the state blob for the connection in the format expected by
// the Airbyte connector protocol:
//   - STREAM/GLOBAL: array of AirbyteStateMessage objects with UPPERCASE type
//   - LEGACY: the raw data object (unwrapped from the storage array)
//
// Returns nil if no state exists (connector will do full refresh).
func (uc *GetSyncState) Execute(ctx context.Context, params GetSyncStateParams) (json.RawMessage, error) {
	state, err := uc.storage.ConnectionStates().First(ctx, &pipelineservice.ConnectionStateFilter{
		ConnectionID: filter.Equals(params.ConnectionID),
	})
	if err != nil {
		return nil, nil //nolint:nilerr // not-found is expected, return nil for full refresh
	}

	if state.StateBlob == "" || state.StateBlob == "[]" {
		return nil, nil
	}

	// Unmarshal the stored state messages.
	var messages []*protocol.AirbyteStateMessage
	if err := json.Unmarshal([]byte(state.StateBlob), &messages); err != nil {
		return nil, fmt.Errorf("unmarshaling state blob: %w", err)
	}
	if len(messages) == 0 {
		return nil, nil
	}

	// Legacy state must be unwrapped: connectors expect a plain JSON object
	// (the raw cursor/state data), not the array wrapper we use for storage.
	if protocol.AirbyteStateType(state.StateType) == protocol.StateTypeLegacy {
		if messages[0].Data == nil {
			return nil, nil
		}
		return messages[0].Data, nil
	}

	// Airbyte CDK expects UPPERCASE type values ("STREAM", "GLOBAL").
	// We store lowercase for the Postgres enum, so convert back before
	// returning to connectors.
	for _, msg := range messages {
		msg.Type = protocol.AirbyteStateType(strings.ToUpper(string(msg.Type)))
	}

	data, err := json.Marshal(messages)
	if err != nil {
		return nil, fmt.Errorf("marshaling state for connector: %w", err)
	}
	return data, nil
}
