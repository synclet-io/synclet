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
// - STREAM: merges the per-stream state into the existing blob array
// - GLOBAL: replaces the entire blob (global messages are always complete)
// - LEGACY: replaces the entire blob
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

// mergeStreamState merges a STREAM state message into the existing blob array.
// It finds a matching stream by (namespace, name) and replaces its state,
// or appends a new entry if no match exists.
func mergeStreamState(blobJSON string, msg *protocol.AirbyteStateMessage) (string, error) {
	var states []*protocol.AirbyteStateMessage
	if blobJSON != "" && blobJSON != "[]" {
		if err := json.Unmarshal([]byte(blobJSON), &states); err != nil {
			// If existing blob is not a valid array (e.g., state type changed), reset.
			states = nil
		}
	}

	// Find matching stream descriptor.
	targetName := msg.Stream.StreamDescriptor.Name
	targetNS := msg.Stream.StreamDescriptor.Namespace
	found := false
	for i, s := range states {
		if s.Type == protocol.StateTypeStream && s.Stream != nil &&
			s.Stream.StreamDescriptor.Name == targetName &&
			s.Stream.StreamDescriptor.Namespace == targetNS {
			states[i] = msg
			found = true
			break
		}
	}
	if !found {
		states = append(states, msg)
	}

	data, err := json.Marshal(states)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
