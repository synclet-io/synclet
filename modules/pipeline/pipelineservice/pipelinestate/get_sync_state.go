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

	// SourceCatalog and the naming settings below describe the connection the
	// state is read for. When SourceCatalog is set, a stored descriptor that
	// matches no catalog stream but is the destination name exactly one catalog
	// stream gets under these settings is handed back under that stream's name.
	SourceCatalog         *protocol.ConfiguredAirbyteCatalog
	NamespaceDefinition   pipelineservice.NamespaceDefinition
	CustomNamespaceFormat *string
	StreamPrefix          *string
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
// When SourceCatalog is set, a stream configured as full refresh gets no state:
// its STREAM entries and its GLOBAL stream_states are left out, and GLOBAL and
// LEGACY state are left out when every catalog stream is full refresh. The stored state is not changed.
//
// Returns nil if no state is left (connector will do full refresh).
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
		if messages[0].Data == nil || (params.SourceCatalog != nil && !hasIncremental(params.SourceCatalog)) {
			return nil, nil
		}

		return messages[0].Data, nil
	}

	if params.SourceCatalog != nil {
		messages = omitFullRefresh(restoreSourceNames(messages, params), params.SourceCatalog)
		if len(messages) == 0 {
			return nil, nil
		}
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

// restoreSourceNames hands state stored under destination names back under the
// source catalog's names. A descriptor that is itself a catalog stream is never
// mapped, and a mapped entry is dropped when the blob also holds an entry under
// that stream's own name.
func restoreSourceNames(messages []*protocol.AirbyteStateMessage, params GetSyncStateParams) []*protocol.AirbyteStateMessage {
	rewritten := &protocol.ConfiguredAirbyteCatalog{
		Streams: append([]protocol.ConfiguredAirbyteStream(nil), params.SourceCatalog.Streams...),
	}
	pipelineservice.ApplyNamespaceAndPrefix(rewritten, params.NamespaceDefinition, params.CustomNamespaceFormat, params.StreamPrefix)

	source := make(map[protocol.StreamDescriptor]bool, len(params.SourceCatalog.Streams))
	toSource := make(map[protocol.StreamDescriptor]protocol.StreamDescriptor, len(params.SourceCatalog.Streams))
	ambiguous := make(map[protocol.StreamDescriptor]bool)

	for i, stream := range params.SourceCatalog.Streams {
		src := protocol.StreamDescriptor{Name: stream.Stream.Name, Namespace: stream.Stream.Namespace}
		dst := protocol.StreamDescriptor{Name: rewritten.Streams[i].Stream.Name, Namespace: rewritten.Streams[i].Stream.Namespace}

		source[src] = true

		if _, taken := toSource[dst]; taken {
			ambiguous[dst] = true
		}

		toSource[dst] = src
	}

	// restore returns the descriptor to hand to the source and whether the
	// entry is kept: entries that collide with an entry under the stream's own
	// name are not.
	restore := func(desc protocol.StreamDescriptor, present map[protocol.StreamDescriptor]bool) (protocol.StreamDescriptor, bool) {
		if source[desc] || ambiguous[desc] {
			return desc, true
		}

		src, ok := toSource[desc]
		if !ok {
			return desc, true
		}

		return src, !present[src]
	}

	present := make(map[protocol.StreamDescriptor]bool)

	for _, msg := range messages {
		if msg.Stream != nil && source[msg.Stream.StreamDescriptor] {
			present[msg.Stream.StreamDescriptor] = true
		}
	}

	restored := make([]*protocol.AirbyteStateMessage, 0, len(messages))

	for _, msg := range messages {
		switch {
		case msg.Stream != nil:
			desc, keep := restore(msg.Stream.StreamDescriptor, present)
			if !keep {
				continue
			}

			msg.Stream.StreamDescriptor = desc
		case msg.Global != nil:
			globalPresent := make(map[protocol.StreamDescriptor]bool)

			for _, streamState := range msg.Global.StreamStates {
				if source[streamState.StreamDescriptor] {
					globalPresent[streamState.StreamDescriptor] = true
				}
			}

			streamStates := make([]protocol.AirbyteStreamState, 0, len(msg.Global.StreamStates))

			for _, streamState := range msg.Global.StreamStates {
				desc, keep := restore(streamState.StreamDescriptor, globalPresent)
				if !keep {
					continue
				}

				streamState.StreamDescriptor = desc
				streamStates = append(streamStates, streamState)
			}

			msg.Global.StreamStates = streamStates
		}

		restored = append(restored, msg)
	}

	return restored
}

func hasIncremental(catalog *protocol.ConfiguredAirbyteCatalog) bool {
	for _, stream := range catalog.Streams {
		if stream.SyncMode != protocol.SyncModeFullRefresh {
			return true
		}
	}

	return false
}

// omitFullRefresh drops the state of every catalog stream configured as full
// refresh. Descriptors that match no catalog stream are kept. A GLOBAL message
// keeps its shared_state while the catalog has an incremental stream and is
// dropped when every stream is full refresh.
func omitFullRefresh(messages []*protocol.AirbyteStateMessage, catalog *protocol.ConfiguredAirbyteCatalog) []*protocol.AirbyteStateMessage {
	fullRefresh := make(map[protocol.StreamDescriptor]bool, len(catalog.Streams))

	for _, stream := range catalog.Streams {
		if stream.SyncMode == protocol.SyncModeFullRefresh {
			fullRefresh[protocol.StreamDescriptor{Name: stream.Stream.Name, Namespace: stream.Stream.Namespace}] = true
		}
	}

	if len(fullRefresh) == 0 {
		return messages
	}

	incremental := hasIncremental(catalog)
	kept := make([]*protocol.AirbyteStateMessage, 0, len(messages))

	for _, msg := range messages {
		switch {
		case msg.Stream != nil:
			if fullRefresh[msg.Stream.StreamDescriptor] {
				continue
			}
		case msg.Global != nil:
			if !incremental {
				continue
			}

			streamStates := make([]protocol.AirbyteStreamState, 0, len(msg.Global.StreamStates))

			for _, streamState := range msg.Global.StreamStates {
				if !fullRefresh[streamState.StreamDescriptor] {
					streamStates = append(streamStates, streamState)
				}
			}

			msg.Global.StreamStates = streamStates
		}

		kept = append(kept, msg)
	}

	return kept
}
