package pipelinestate_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// mockStorage implements a minimal pipelineservice.Storage for testing state use cases.
type mockStorage struct {
	pipelineservice.Storage
	connStates *mockConnectionStatesStorage
	streamGens *mockStreamGenerationsStorage
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		connStates: &mockConnectionStatesStorage{
			states: make(map[uuid.UUID]*pipelineservice.ConnectionState),
		},
		streamGens: &mockStreamGenerationsStorage{
			gens: make(map[string]*pipelineservice.StreamGeneration),
		},
	}
}

func (m *mockStorage) ConnectionStates() pipelineservice.ConnectionStatesStorage {
	return m.connStates
}

func (m *mockStorage) StreamGenerations() pipelineservice.StreamGenerationsStorage {
	return m.streamGens
}

// mockStreamGenerationsStorage stores StreamGeneration records in memory, keyed by "connID|ns|name".
type mockStreamGenerationsStorage struct {
	pipelineservice.StreamGenerationsStorage
	gens map[string]*pipelineservice.StreamGeneration
}

func streamGenKey(connID uuid.UUID, ns, name string) string {
	return connID.String() + "|" + ns + "|" + name
}

func (m *mockStreamGenerationsStorage) Find(_ context.Context, genFilter *pipelineservice.StreamGenerationFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.StreamGeneration, error) {
	var result []*pipelineservice.StreamGeneration

	for _, streamGen := range m.gens {
		if genFilter.ConnectionID != nil {
			connVal := genFilter.ConnectionID.(*filter.EqualsFilter[uuid.UUID]).Value
			if streamGen.ConnectionID != connVal {
				continue
			}
		}

		if genFilter.StreamNamespace != nil {
			nsVal := genFilter.StreamNamespace.(*filter.EqualsFilter[string]).Value
			if streamGen.StreamNamespace != nsVal {
				continue
			}
		}

		if genFilter.StreamName != nil {
			nameVal := genFilter.StreamName.(*filter.EqualsFilter[string]).Value
			if streamGen.StreamName != nameVal {
				continue
			}
		}

		cp := streamGen.Copy()
		result = append(result, &cp)
	}

	return result, nil
}

func (m *mockStreamGenerationsStorage) Save(_ context.Context, record *pipelineservice.StreamGeneration) (*pipelineservice.StreamGeneration, error) {
	cp := record.Copy()
	key := streamGenKey(record.ConnectionID, record.StreamNamespace, record.StreamName)
	m.gens[key] = &cp

	return &cp, nil
}

// mockConnectionStatesStorage stores ConnectionState in memory.
type mockConnectionStatesStorage struct {
	pipelineservice.ConnectionStatesStorage
	states map[uuid.UUID]*pipelineservice.ConnectionState
}

func (m *mockConnectionStatesStorage) First(_ context.Context, f *pipelineservice.ConnectionStateFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ConnectionState, error) {
	for _, s := range m.states {
		if s.ConnectionID == f.ConnectionID.(*filter.EqualsFilter[uuid.UUID]).Value {
			cp := *s

			return &cp, nil
		}
	}

	return nil, pipelineservice.ErrConnectionStateNotFound
}

func (m *mockConnectionStatesStorage) Save(_ context.Context, record *pipelineservice.ConnectionState) (*pipelineservice.ConnectionState, error) {
	cp := *record
	m.states[record.ConnectionID] = &cp

	return &cp, nil
}

func (m *mockConnectionStatesStorage) Delete(_ context.Context, f *pipelineservice.ConnectionStateFilter) error {
	for id, s := range m.states {
		if s.ConnectionID == f.ConnectionID.(*filter.EqualsFilter[uuid.UUID]).Value {
			delete(m.states, id)
		}
	}

	return nil
}

func TestSaveStreamState(t *testing.T) {
	store := newMockStorage()
	uc := pipelinestate.NewSaveSyncState(store)
	connID := uuid.New()

	err := uc.Execute(context.Background(), pipelinestate.SaveSyncStateParams{
		ConnectionID: connID,
		StateMessage: &protocol.AirbyteStateMessage{
			Type: protocol.StateTypeStream,
			Stream: &protocol.AirbyteStreamState{
				StreamDescriptor: protocol.StreamDescriptor{Name: "users", Namespace: "public"},
				StreamState:      json.RawMessage(`{"cursor":"2024-01-01"}`),
			},
		},
	})
	require.NoError(t, err)

	// Verify stored state.
	state := store.connStates.states[connID]
	require.NotNil(t, state)
	assert.Equal(t, "stream", state.StateType)

	var msgs []*protocol.AirbyteStateMessage
	err = json.Unmarshal([]byte(state.StateBlob), &msgs)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, protocol.StateTypeStream, msgs[0].Type)
	assert.Equal(t, "users", msgs[0].Stream.StreamDescriptor.Name)
	assert.Equal(t, "public", msgs[0].Stream.StreamDescriptor.Namespace)
}

func TestSaveStreamStateMerge(t *testing.T) {
	store := newMockStorage()
	useCase := pipelinestate.NewSaveSyncState(store)
	connID := uuid.New()
	ctx := context.Background()

	// Save "users" stream.
	err := useCase.Execute(ctx, pipelinestate.SaveSyncStateParams{
		ConnectionID: connID,
		StateMessage: &protocol.AirbyteStateMessage{
			Type: protocol.StateTypeStream,
			Stream: &protocol.AirbyteStreamState{
				StreamDescriptor: protocol.StreamDescriptor{Name: "users", Namespace: "public"},
				StreamState:      json.RawMessage(`{"cursor":"2024-01-01"}`),
			},
		},
	})
	require.NoError(t, err)

	// Save "orders" stream.
	err = useCase.Execute(ctx, pipelinestate.SaveSyncStateParams{
		ConnectionID: connID,
		StateMessage: &protocol.AirbyteStateMessage{
			Type: protocol.StateTypeStream,
			Stream: &protocol.AirbyteStreamState{
				StreamDescriptor: protocol.StreamDescriptor{Name: "orders", Namespace: "public"},
				StreamState:      json.RawMessage(`{"cursor":"100"}`),
			},
		},
	})
	require.NoError(t, err)

	// Should have 2 entries.
	var msgs []*protocol.AirbyteStateMessage
	err = json.Unmarshal([]byte(store.connStates.states[connID].StateBlob), &msgs)
	require.NoError(t, err)
	assert.Len(t, msgs, 2)

	// Update "users" stream with new cursor.
	err = useCase.Execute(ctx, pipelinestate.SaveSyncStateParams{
		ConnectionID: connID,
		StateMessage: &protocol.AirbyteStateMessage{
			Type: protocol.StateTypeStream,
			Stream: &protocol.AirbyteStreamState{
				StreamDescriptor: protocol.StreamDescriptor{Name: "users", Namespace: "public"},
				StreamState:      json.RawMessage(`{"cursor":"2024-06-15"}`),
			},
		},
	})
	require.NoError(t, err)

	// Still 2 entries, but "users" updated.
	err = json.Unmarshal([]byte(store.connStates.states[connID].StateBlob), &msgs)
	require.NoError(t, err)
	assert.Len(t, msgs, 2)

	// Find users entry and verify updated cursor.
	for _, m := range msgs {
		if m.Stream.StreamDescriptor.Name == "users" {
			assert.JSONEq(t, `{"cursor":"2024-06-15"}`, string(m.Stream.StreamState))
		}
	}
}

func TestSaveGlobalState(t *testing.T) {
	store := newMockStorage()
	uc := pipelinestate.NewSaveSyncState(store)
	connID := uuid.New()

	err := uc.Execute(context.Background(), pipelinestate.SaveSyncStateParams{
		ConnectionID: connID,
		StateMessage: &protocol.AirbyteStateMessage{
			Type: protocol.StateTypeGlobal,
			Global: &protocol.AirbyteGlobalState{
				SharedState: json.RawMessage(`{"cdc_offset":"abc123"}`),
				StreamStates: []protocol.AirbyteStreamState{
					{
						StreamDescriptor: protocol.StreamDescriptor{Name: "users"},
						StreamState:      json.RawMessage(`{"cursor":"100"}`),
					},
				},
			},
		},
	})
	require.NoError(t, err)

	state := store.connStates.states[connID]
	require.NotNil(t, state)
	assert.Equal(t, "global", state.StateType)

	var msgs []*protocol.AirbyteStateMessage
	err = json.Unmarshal([]byte(state.StateBlob), &msgs)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, protocol.StateTypeGlobal, msgs[0].Type)
	assert.NotNil(t, msgs[0].Global)
	assert.JSONEq(t, `{"cdc_offset":"abc123"}`, string(msgs[0].Global.SharedState))
}

func TestSaveLegacyState(t *testing.T) {
	store := newMockStorage()
	uc := pipelinestate.NewSaveSyncState(store)
	connID := uuid.New()

	err := uc.Execute(context.Background(), pipelinestate.SaveSyncStateParams{
		ConnectionID: connID,
		StateMessage: &protocol.AirbyteStateMessage{
			Type: protocol.StateTypeLegacy,
			Data: json.RawMessage(`{"position":42}`),
		},
	})
	require.NoError(t, err)

	state := store.connStates.states[connID]
	require.NotNil(t, state)
	assert.Equal(t, "legacy", state.StateType)

	var msgs []*protocol.AirbyteStateMessage
	err = json.Unmarshal([]byte(state.StateBlob), &msgs)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, protocol.StateTypeLegacy, msgs[0].Type)
	assert.JSONEq(t, `{"position":42}`, string(msgs[0].Data))
}

func TestStateFormat(t *testing.T) {
	store := newMockStorage()
	useCase := pipelinestate.NewSaveSyncState(store)
	connID := uuid.New()
	ctx := context.Background()

	// Save two STREAM states.
	for _, stream := range []struct {
		name, ns, cursor string
	}{
		{"users", "public", `{"cursor":"2024-01-01"}`},
		{"orders", "", `{"cursor":"500"}`},
	} {
		err := useCase.Execute(ctx, pipelinestate.SaveSyncStateParams{
			ConnectionID: connID,
			StateMessage: &protocol.AirbyteStateMessage{
				Type: protocol.StateTypeStream,
				Stream: &protocol.AirbyteStreamState{
					StreamDescriptor: protocol.StreamDescriptor{Name: stream.name, Namespace: stream.ns},
					StreamState:      json.RawMessage(stream.cursor),
				},
			},
		})
		require.NoError(t, err)
	}

	// Verify the blob is a valid JSON array of AirbyteStateMessage (Airbyte protocol format).
	blob := store.connStates.states[connID].StateBlob
	var msgs []*protocol.AirbyteStateMessage
	err := json.Unmarshal([]byte(blob), &msgs)
	require.NoError(t, err)
	require.Len(t, msgs, 2)

	// Each entry has type, stream.stream_descriptor, stream.stream_state.
	for _, m := range msgs {
		assert.Equal(t, protocol.StateTypeStream, m.Type)
		assert.NotNil(t, m.Stream)
		assert.NotEmpty(t, m.Stream.StreamDescriptor.Name)
		assert.NotEmpty(t, m.Stream.StreamState)
	}

	// Verify JSON field names match Airbyte protocol.
	var raw []map[string]json.RawMessage
	err = json.Unmarshal([]byte(blob), &raw)
	require.NoError(t, err)

	for _, entry := range raw {
		assert.Contains(t, entry, "type")
		assert.Contains(t, entry, "stream")

		var streamObj map[string]json.RawMessage
		err = json.Unmarshal(entry["stream"], &streamObj)
		require.NoError(t, err)
		assert.Contains(t, streamObj, "stream_descriptor")
		assert.Contains(t, streamObj, "stream_state")
	}
}
