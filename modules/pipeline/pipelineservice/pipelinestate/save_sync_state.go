package pipelinestate

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// SaveSyncStateParams holds parameters for saving sync state.
type SaveSyncStateParams struct {
	ConnectionID uuid.UUID
	StateMessage *protocol.AirbyteStateMessage
}

// SaveSyncState upserts state for a connection based on the state message type.
type SaveSyncState struct {
	storage pipelineservice.Storage
}

// NewSaveSyncState creates a new SaveSyncState use case.
func NewSaveSyncState(storage pipelineservice.Storage) *SaveSyncState {
	return &SaveSyncState{storage: storage}
}

// Execute saves state based on the message type:
//   - STREAM: merges the per-stream state into the existing blob array, which
//     then holds STREAM entries only. A GLOBAL entry in the blob is turned into
//     one STREAM entry per element of its stream_states and its shared_state is
//     dropped; a LEGACY entry is dropped. STREAM entries already in the blob
//     override the seeded ones, and the incoming message overrides both.
//   - GLOBAL: replaces the entire blob (global messages are always complete)
//   - LEGACY: replaces the entire blob
func (uc *SaveSyncState) Execute(ctx context.Context, params SaveSyncStateParams) error {
	if params.StateMessage == nil {
		return nil
	}

	// Normalize state type from Airbyte UPPERCASE to lowercase for Postgres enum.
	params.StateMessage.Type = protocol.NormalizeStateType(params.StateMessage.Type)

	// Load existing connection state (or create new).
	existing, err := uc.storage.ConnectionStates().First(ctx, &pipelineservice.ConnectionStateFilter{
		ConnectionID: filter.Equals(params.ConnectionID),
	})
	if err != nil {
		// Not found — create new.
		existing = &pipelineservice.ConnectionState{
			ConnectionID: params.ConnectionID,
			StateType:    string(protocol.StateTypeStream),
			StateBlob:    "[]",
		}
	}

	switch params.StateMessage.Type {
	case protocol.StateTypeStream:
		if params.StateMessage.Stream == nil {
			return nil
		}

		blob, mergeErr := mergeStreamState(existing.StateBlob, params.StateMessage)
		if mergeErr != nil {
			return fmt.Errorf("merging stream state: %w", mergeErr)
		}

		existing.StateBlob = blob
		existing.StateType = string(protocol.StateTypeStream)

	case protocol.StateTypeGlobal:
		// Global state replaces entire blob — it is always a complete snapshot.
		data, marshalErr := json.Marshal([]*protocol.AirbyteStateMessage{params.StateMessage})
		if marshalErr != nil {
			return fmt.Errorf("marshaling global state: %w", marshalErr)
		}

		existing.StateBlob = string(data)
		existing.StateType = string(protocol.StateTypeGlobal)

	case protocol.StateTypeLegacy, "":
		// Legacy state replaces entire blob.
		data, marshalErr := json.Marshal([]*protocol.AirbyteStateMessage{params.StateMessage})
		if marshalErr != nil {
			return fmt.Errorf("marshaling legacy state: %w", marshalErr)
		}

		existing.StateBlob = string(data)
		existing.StateType = string(protocol.StateTypeLegacy)

	default:
		return fmt.Errorf("unknown state type: %s", params.StateMessage.Type)
	}

	existing.UpdatedAt = time.Now()

	if _, err := uc.storage.ConnectionStates().Save(ctx, existing); err != nil {
		return fmt.Errorf("saving connection state: %w", err)
	}

	return nil
}

// mergeStreamState merges a STREAM state message into the existing blob array
// and returns an array of STREAM entries only, one per (namespace, name).
func mergeStreamState(blobJSON string, msg *protocol.AirbyteStateMessage) (string, error) {
	var existing []*protocol.AirbyteStateMessage
	if blobJSON != "" && blobJSON != "[]" {
		if err := json.Unmarshal([]byte(blobJSON), &existing); err != nil {
			existing = nil
		}
	}

	var states []*protocol.AirbyteStateMessage

	upsert := func(entry *protocol.AirbyteStateMessage) {
		desc := entry.Stream.StreamDescriptor
		for i, s := range states {
			if s.Stream.StreamDescriptor == desc {
				states[i] = entry

				return
			}
		}

		states = append(states, entry)
	}

	for _, entry := range existing {
		switch protocol.NormalizeStateType(entry.Type) {
		case protocol.StateTypeStream:
			if entry.Stream != nil {
				entry.Type = protocol.StateTypeStream
				upsert(entry)
			}
		case protocol.StateTypeGlobal:
			if entry.Global != nil {
				for _, streamState := range entry.Global.StreamStates {
					upsert(&protocol.AirbyteStateMessage{Type: protocol.StateTypeStream, Stream: &streamState})
				}
			}
		}
	}

	upsert(msg)

	data, err := json.Marshal(states)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
