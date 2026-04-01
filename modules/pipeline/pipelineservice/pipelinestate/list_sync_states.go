package pipelinestate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// ListSyncStatesParams holds parameters for listing sync states.
type ListSyncStatesParams struct {
	ConnectionID uuid.UUID
}

// ListSyncStatesResult holds the result of listing states, including per-stream
// breakdown for UI display and the overall state type.
type ListSyncStatesResult struct {
	StateType    string
	StreamStates []StreamStateEntry
}

// StreamStateEntry represents a single stream's state for UI display.
type StreamStateEntry struct {
	StreamNamespace string
	StreamName      string
	StateData       json.RawMessage
}

// ListSyncStates retrieves per-stream state entries from the connection state blob.
// For STREAM type: extracts each stream's state from the array.
// For GLOBAL type: extracts shared_state and per-stream states from global.
// For LEGACY type: returns a single entry with the raw blob.
type ListSyncStates struct {
	storage pipelineservice.Storage
}

// NewListSyncStates creates a new ListSyncStates use case.
func NewListSyncStates(storage pipelineservice.Storage) *ListSyncStates {
	return &ListSyncStates{storage: storage}
}

// Execute returns stream state entries for UI display.
func (uc *ListSyncStates) Execute(ctx context.Context, params ListSyncStatesParams) (*ListSyncStatesResult, error) {
	state, err := uc.storage.ConnectionStates().First(ctx, &pipelineservice.ConnectionStateFilter{
		ConnectionID: filter.Equals(params.ConnectionID),
	})
	if err != nil {
		return &ListSyncStatesResult{StateType: string(protocol.StateTypeStream)}, nil //nolint:nilerr // not-found is expected, return empty result
	}

	if state.StateBlob == "" || state.StateBlob == "[]" {
		return &ListSyncStatesResult{StateType: state.StateType}, nil
	}

	var msgs []*protocol.AirbyteStateMessage
	if err := json.Unmarshal([]byte(state.StateBlob), &msgs); err != nil {
		return nil, fmt.Errorf("parsing state blob: %w", err)
	}

	result := &ListSyncStatesResult{StateType: state.StateType}

	switch protocol.AirbyteStateType(state.StateType) {
	case protocol.StateTypeStream:
		for _, msg := range msgs {
			if msg.Stream == nil {
				continue
			}

			result.StreamStates = append(result.StreamStates, StreamStateEntry{
				StreamNamespace: msg.Stream.StreamDescriptor.Namespace,
				StreamName:      msg.Stream.StreamDescriptor.Name,
				StateData:       msg.Stream.StreamState,
			})
		}

	case protocol.StateTypeGlobal:
		if len(msgs) > 0 && msgs[0].Global != nil {
			for _, ss := range msgs[0].Global.StreamStates {
				result.StreamStates = append(result.StreamStates, StreamStateEntry{
					StreamNamespace: ss.StreamDescriptor.Namespace,
					StreamName:      ss.StreamDescriptor.Name,
					StateData:       ss.StreamState,
				})
			}
		}

	case protocol.StateTypeLegacy:
		if len(msgs) > 0 && msgs[0].Data != nil {
			result.StreamStates = append(result.StreamStates, StreamStateEntry{
				StreamName: "__legacy__",
				StateData:  msgs[0].Data,
			})
		}
	}

	return result, nil
}
