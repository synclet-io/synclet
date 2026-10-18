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

// withFullRefresh sets the named catalog streams to full refresh.
func withFullRefresh(catalog *protocol.ConfiguredAirbyteCatalog, names ...string) *protocol.ConfiguredAirbyteCatalog {
	for i := range catalog.Streams {
		for _, name := range names {
			if catalog.Streams[i].Stream.Name == name {
				catalog.Streams[i].SyncMode = protocol.SyncModeFullRefresh
			}
		}
	}

	return catalog
}

func TestGetSyncStateOmitsFullRefreshStreams(t *testing.T) {
	users := protocol.StreamDescriptor{Name: "users", Namespace: "public"}
	orders := protocol.StreamDescriptor{Name: "orders", Namespace: "public"}

	tests := []struct {
		name       string
		catalog    *protocol.ConfiguredAirbyteCatalog
		prefix     *string
		stateType  string
		blob       string
		wantNil    bool
		want       map[string]int
		wantShared string
		wantRaw    string
	}{
		{
			name:      "stream state with mixed modes",
			catalog:   withFullRefresh(sourceCatalog(users, orders), "users"),
			stateType: "stream",
			blob: `[{"type":"stream","stream":{"stream_descriptor":{"name":"users","namespace":"public"},"stream_state":{"cursor":7}}},` +
				`{"type":"stream","stream":{"stream_descriptor":{"name":"orders","namespace":"public"},"stream_state":{"cursor":3}}}]`,
			want: map[string]int{"public.orders": 3},
		},
		{
			name:      "stream state with every stream full refresh",
			catalog:   withFullRefresh(sourceCatalog(users), "users"),
			stateType: "stream",
			blob:      `[{"type":"stream","stream":{"stream_descriptor":{"name":"users","namespace":"public"},"stream_state":{"cursor":7}}}]`,
			wantNil:   true,
		},
		{
			name:      "global state with mixed modes",
			catalog:   withFullRefresh(sourceCatalog(users, orders), "users"),
			stateType: "global",
			blob: `[{"type":"global","global":{"shared_state":{"lsn":"0/1"},"stream_states":[` +
				`{"stream_descriptor":{"name":"users","namespace":"public"},"stream_state":{"cursor":7}},` +
				`{"stream_descriptor":{"name":"orders","namespace":"public"},"stream_state":{"cursor":3}}]}}]`,
			want:       map[string]int{"public.orders": 3},
			wantShared: `{"lsn":"0/1"}`,
		},
		{
			name:      "global state with every stream full refresh",
			catalog:   withFullRefresh(sourceCatalog(users), "users"),
			stateType: "global",
			blob: `[{"type":"global","global":{"shared_state":{"lsn":"0/1"},"stream_states":[` +
				`{"stream_descriptor":{"name":"users","namespace":"public"},"stream_state":{"cursor":7}}]}}]`,
			wantNil: true,
		},
		{
			name:       "global shared_state only with mixed modes",
			catalog:    withFullRefresh(sourceCatalog(users, orders), "users"),
			stateType:  "global",
			blob:       `[{"type":"global","global":{"shared_state":{"lsn":"0/1"},"stream_states":[]}}]`,
			want:       map[string]int{},
			wantShared: `{"lsn":"0/1"}`,
		},
		{
			name:      "global shared_state only with every stream full refresh",
			catalog:   withFullRefresh(sourceCatalog(users), "users"),
			stateType: "global",
			blob:      `[{"type":"global","global":{"shared_state":{"lsn":"0/1"},"stream_states":[]}}]`,
			wantNil:   true,
		},
		{
			name:      "global stream_states of full-refresh streams only with an incremental stream in the catalog",
			catalog:   withFullRefresh(sourceCatalog(users, orders), "users"),
			stateType: "global",
			blob: `[{"type":"global","global":{"shared_state":{"lsn":"0/1"},"stream_states":[` +
				`{"stream_descriptor":{"name":"users","namespace":"public"},"stream_state":{"cursor":7}}]}}]`,
			want:       map[string]int{},
			wantShared: `{"lsn":"0/1"}`,
		},
		{
			name:      "entries under prefixed names",
			catalog:   withFullRefresh(sourceCatalog(users, orders), "users"),
			prefix:    strPtr("pre_"),
			stateType: "stream",
			blob: `[{"type":"stream","stream":{"stream_descriptor":{"name":"pre_users","namespace":"public"},"stream_state":{"cursor":7}}},` +
				`{"type":"stream","stream":{"stream_descriptor":{"name":"pre_orders","namespace":"public"},"stream_state":{"cursor":3}}}]`,
			want: map[string]int{"public.orders": 3},
		},
		{
			name:      "legacy state with every stream full refresh",
			catalog:   withFullRefresh(sourceCatalog(users), "users"),
			stateType: "legacy",
			blob:      `[{"type":"legacy","data":{"users":{"cursor":7}}}]`,
			wantNil:   true,
		},
		{
			name:      "legacy state with mixed modes",
			catalog:   withFullRefresh(sourceCatalog(users, orders), "users"),
			stateType: "legacy",
			blob:      `[{"type":"legacy","data":{"users":{"cursor":7}}}]`,
			wantRaw:   `{"users":{"cursor":7}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStorage()
			connID := uuid.New()
			saveBlob(store, connID, tt.stateType, tt.blob)

			blob, err := pipelinestate.NewGetSyncState(store).Execute(context.Background(), pipelinestate.GetSyncStateParams{
				ConnectionID:        connID,
				SourceCatalog:       tt.catalog,
				NamespaceDefinition: pipelineservice.NamespaceDefinitionSource,
				StreamPrefix:        tt.prefix,
			})
			require.NoError(t, err)

			switch {
			case tt.wantNil:
				assert.Nil(t, blob)
			case tt.wantRaw != "":
				assert.JSONEq(t, tt.wantRaw, string(blob))
			default:
				assert.Equal(t, tt.want, cursorsByStream(t, blob))

				if tt.wantShared != "" {
					var messages []*protocol.AirbyteStateMessage
					require.NoError(t, json.Unmarshal(blob, &messages))
					require.Len(t, messages, 1)
					require.NotNil(t, messages[0].Global)
					assert.JSONEq(t, tt.wantShared, string(messages[0].Global.SharedState))
				}
			}

			assert.Equal(t, tt.blob, store.connStates.states[connID].StateBlob, "stored state is untouched")
		})
	}
}
