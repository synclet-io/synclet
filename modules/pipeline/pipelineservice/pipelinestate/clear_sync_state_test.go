package pipelinestate_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/pkg/protocol"
)

func TestClearAllState(t *testing.T) {
	store := newMockStorage()
	connID := uuid.New()

	// Pre-populate with a state.
	store.connStates.states[connID] = &pipelineservice.ConnectionState{
		ConnectionID: connID,
		StateType:    "stream",
		StateBlob:    `[{"type":"STREAM","stream":{"stream_descriptor":{"name":"users"},"stream_state":{"cursor":"100"}}}]`,
	}

	uc := pipelinestate.NewClearSyncState(store)
	err := uc.Execute(context.Background(), pipelinestate.ClearSyncStateParams{
		ConnectionID: connID,
		StreamName:   nil, // clear all
	})
	require.NoError(t, err)

	// Row should be deleted.
	_, exists := store.connStates.states[connID]
	assert.False(t, exists)
}

func TestClearAllState_IncrementsAllGenerations(t *testing.T) {
	store := newMockStorage()
	connID := uuid.New()

	// Pre-populate with connection state.
	store.connStates.states[connID] = &pipelineservice.ConnectionState{
		ConnectionID: connID,
		StateType:    "stream",
		StateBlob:    `[{"type":"STREAM","stream":{"stream_descriptor":{"name":"users"},"stream_state":{"cursor":"100"}}}]`,
	}

	// Pre-populate with two stream generation records.
	store.streamGens.gens[streamGenKey(connID, "public", "users")] = &pipelineservice.StreamGeneration{
		ConnectionID:    connID,
		StreamNamespace: "public",
		StreamName:      "users",
		GenerationID:    3,
	}
	store.streamGens.gens[streamGenKey(connID, "public", "orders")] = &pipelineservice.StreamGeneration{
		ConnectionID:    connID,
		StreamNamespace: "public",
		StreamName:      "orders",
		GenerationID:    5,
	}

	uc := pipelinestate.NewClearSyncState(store)
	err := uc.Execute(context.Background(), pipelinestate.ClearSyncStateParams{
		ConnectionID: connID,
		StreamName:   nil, // clear all
	})
	require.NoError(t, err)

	// Both generation_ids should be incremented by 1.
	usersGen := store.streamGens.gens[streamGenKey(connID, "public", "users")]
	require.NotNil(t, usersGen)
	assert.Equal(t, int64(4), usersGen.GenerationID)

	ordersGen := store.streamGens.gens[streamGenKey(connID, "public", "orders")]
	require.NotNil(t, ordersGen)
	assert.Equal(t, int64(6), ordersGen.GenerationID)
}

func TestClearAllState_NoExistingGenerations(t *testing.T) {
	store := newMockStorage()
	connID := uuid.New()

	// Pre-populate with connection state but NO generation records.
	store.connStates.states[connID] = &pipelineservice.ConnectionState{
		ConnectionID: connID,
		StateType:    "stream",
		StateBlob:    `[]`,
	}

	uc := pipelinestate.NewClearSyncState(store)
	err := uc.Execute(context.Background(), pipelinestate.ClearSyncStateParams{
		ConnectionID: connID,
		StreamName:   nil, // clear all
	})
	// Should not error -- nothing to increment.
	require.NoError(t, err)

	// No generation records should be created.
	assert.Empty(t, store.streamGens.gens)
}

func TestClearSingleStream_IncrementsGeneration(t *testing.T) {
	store := newMockStorage()
	connID := uuid.New()

	// Pre-populate with state containing two streams.
	blob, _ := json.Marshal([]*protocol.AirbyteStateMessage{
		{
			Type: protocol.StateTypeStream,
			Stream: &protocol.AirbyteStreamState{
				StreamDescriptor: protocol.StreamDescriptor{Name: "users", Namespace: "public"},
				StreamState:      json.RawMessage(`{"cursor":"100"}`),
			},
		},
	})
	store.connStates.states[connID] = &pipelineservice.ConnectionState{
		ConnectionID: connID,
		StateType:    "stream",
		StateBlob:    string(blob),
	}

	// Pre-populate with existing generation record.
	store.streamGens.gens[streamGenKey(connID, "public", "users")] = &pipelineservice.StreamGeneration{
		ConnectionID:    connID,
		StreamNamespace: "public",
		StreamName:      "users",
		GenerationID:    7,
	}

	uc := pipelinestate.NewClearSyncState(store)
	streamName := "users"
	streamNS := "public"
	err := uc.Execute(context.Background(), pipelinestate.ClearSyncStateParams{
		ConnectionID:    connID,
		StreamName:      &streamName,
		StreamNamespace: &streamNS,
	})
	require.NoError(t, err)

	// Generation should be incremented by 1.
	usersGen := store.streamGens.gens[streamGenKey(connID, "public", "users")]
	require.NotNil(t, usersGen)
	assert.Equal(t, int64(8), usersGen.GenerationID)
}

func TestClearSingleStream_NoExistingGeneration_CreatesRecord(t *testing.T) {
	store := newMockStorage()
	connID := uuid.New()

	// Pre-populate with state but NO generation record for this stream.
	blob, _ := json.Marshal([]*protocol.AirbyteStateMessage{
		{
			Type: protocol.StateTypeStream,
			Stream: &protocol.AirbyteStreamState{
				StreamDescriptor: protocol.StreamDescriptor{Name: "users", Namespace: "public"},
				StreamState:      json.RawMessage(`{"cursor":"100"}`),
			},
		},
	})
	store.connStates.states[connID] = &pipelineservice.ConnectionState{
		ConnectionID: connID,
		StateType:    "stream",
		StateBlob:    string(blob),
	}

	uc := pipelinestate.NewClearSyncState(store)
	streamName := "users"
	streamNS := "public"
	err := uc.Execute(context.Background(), pipelinestate.ClearSyncStateParams{
		ConnectionID:    connID,
		StreamName:      &streamName,
		StreamNamespace: &streamNS,
	})
	require.NoError(t, err)

	// New generation record should be created with generation_id=1.
	usersGen := store.streamGens.gens[streamGenKey(connID, "public", "users")]
	require.NotNil(t, usersGen)
	assert.Equal(t, int64(1), usersGen.GenerationID)
}

func TestClearSingleStream(t *testing.T) {
	store := newMockStorage()
	connID := uuid.New()

	// Pre-populate with two streams.
	blob, _ := json.Marshal([]*protocol.AirbyteStateMessage{
		{
			Type: protocol.StateTypeStream,
			Stream: &protocol.AirbyteStreamState{
				StreamDescriptor: protocol.StreamDescriptor{Name: "users", Namespace: "public"},
				StreamState:      json.RawMessage(`{"cursor":"100"}`),
			},
		},
		{
			Type: protocol.StateTypeStream,
			Stream: &protocol.AirbyteStreamState{
				StreamDescriptor: protocol.StreamDescriptor{Name: "orders", Namespace: "public"},
				StreamState:      json.RawMessage(`{"cursor":"200"}`),
			},
		},
	})
	store.connStates.states[connID] = &pipelineservice.ConnectionState{
		ConnectionID: connID,
		StateType:    "stream",
		StateBlob:    string(blob),
	}

	uc := pipelinestate.NewClearSyncState(store)
	streamName := "users"
	streamNS := "public"
	err := uc.Execute(context.Background(), pipelinestate.ClearSyncStateParams{
		ConnectionID:    connID,
		StreamName:      &streamName,
		StreamNamespace: &streamNS,
	})
	require.NoError(t, err)

	// Row should still exist with only "orders".
	state, exists := store.connStates.states[connID]
	require.True(t, exists)

	var msgs []*protocol.AirbyteStateMessage
	err = json.Unmarshal([]byte(state.StateBlob), &msgs)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "orders", msgs[0].Stream.StreamDescriptor.Name)
}
