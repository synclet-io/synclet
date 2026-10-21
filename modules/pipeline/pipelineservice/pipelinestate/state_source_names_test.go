package pipelinestate_test

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/pkg/protocol"
)

func strPtr(s string) *string { return &s }

func sourceCatalog(streams ...protocol.StreamDescriptor) *protocol.ConfiguredAirbyteCatalog {
	catalog := &protocol.ConfiguredAirbyteCatalog{}
	for _, s := range streams {
		catalog.Streams = append(catalog.Streams, protocol.ConfiguredAirbyteStream{
			Stream:   protocol.AirbyteStream{Name: s.Name, Namespace: s.Namespace},
			SyncMode: protocol.SyncModeIncremental,
		})
	}

	return catalog
}

// savingHandler stores every STATE the destination confirms, as the control
// plane does for a running sync.
type savingHandler struct {
	pipelineexec.DefaultHandler

	save   *pipelinestate.SaveSyncState
	connID uuid.UUID
}

func (h *savingHandler) OnStateConfirmed(ctx context.Context, msg *protocol.AirbyteStateMessage) error {
	return h.save.Execute(ctx, pipelinestate.SaveSyncStateParams{ConnectionID: h.connID, StateMessage: msg})
}

func echoStates(stdin io.Reader, stdout *io.PipeWriter) {
	reader := protocol.NewMessageReader(stdin, protocol.SyncMessageMaxSize)
	writer := protocol.NewMessageWriter(stdout)

	defer func() { _ = stdout.Close() }()

	for {
		msg, err := reader.Read()
		if err != nil {
			return
		}

		if msg.Type == protocol.MessageTypeState {
			if err := writer.Write(msg); err != nil {
				return
			}
		}
	}
}

func runSync(t *testing.T, rewriter *pipelineexec.NamespaceRewriter, handler pipelineexec.Handler, sourceOutput string) {
	t.Helper()

	destStdinReader, destStdinWriter := io.Pipe()
	destStdoutReader, destStdoutWriter := io.Pipe()

	go echoStates(destStdinReader, destStdoutWriter)

	_, err := pipelineexec.Run(context.Background(), strings.NewReader(sourceOutput), destStdinWriter, destStdoutReader, handler, pipelineexec.RunConfig{Rewriter: rewriter}, nil)
	require.NoError(t, err)
}

// cursorsByStream reads the state handed to the source and returns each
// stream's cursor keyed by "namespace.name". It fails the test when a stream
// has more than one entry.
func cursorsByStream(t *testing.T, blob json.RawMessage) map[string]int {
	t.Helper()

	var messages []*protocol.AirbyteStateMessage
	require.NoError(t, json.Unmarshal(blob, &messages))

	cursors := map[string]int{}

	add := func(s protocol.AirbyteStreamState) {
		var state struct {
			Cursor int `json:"cursor"`
		}

		require.NoError(t, json.Unmarshal(s.StreamState, &state))

		key := s.StreamDescriptor.Namespace + "." + s.StreamDescriptor.Name
		require.NotContains(t, cursors, key, "the state holds one entry per stream")

		cursors[key] = state.Cursor
	}

	for _, msg := range messages {
		if msg.Stream != nil {
			add(*msg.Stream)
		}

		if msg.Global != nil {
			for _, s := range msg.Global.StreamStates {
				add(s)
			}
		}
	}

	return cursors
}

func TestSecondSyncGetsStateUnderSourceNames(t *testing.T) {
	users := protocol.StreamDescriptor{Name: "users", Namespace: "public"}
	orders := protocol.StreamDescriptor{Name: "orders", Namespace: "public"}

	tests := []struct {
		name   string
		source string
	}{
		{
			name: "stream state",
			source: `{"type":"STATE","state":{"type":"STREAM","stream":{"stream_descriptor":{"name":"users","namespace":"public"},"stream_state":{"cursor":7}}}}` + "\n" +
				`{"type":"STATE","state":{"type":"STREAM","stream":{"stream_descriptor":{"name":"orders","namespace":"public"},"stream_state":{"cursor":3}}}}` + "\n",
		},
		{
			name: "global state",
			source: `{"type":"STATE","state":{"type":"GLOBAL","global":{"shared_state":{"lsn":"0/1"},"stream_states":[` +
				`{"stream_descriptor":{"name":"users","namespace":"public"},"stream_state":{"cursor":7}},` +
				`{"stream_descriptor":{"name":"orders","namespace":"public"},"stream_state":{"cursor":3}}]}}}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStorage()
			connID := uuid.New()
			catalog := sourceCatalog(users, orders)
			rewriter := pipelineexec.NewNamespaceRewriter(catalog, pipelineservice.NamespaceDefinitionCustom, strPtr("raw_${SOURCE_NAMESPACE}"), strPtr("pre_"))

			runSync(t, rewriter, &savingHandler{save: pipelinestate.NewSaveSyncState(store), connID: connID}, tt.source)

			blob, err := pipelinestate.NewGetSyncState(store).Execute(context.Background(), pipelinestate.GetSyncStateParams{
				ConnectionID:          connID,
				SourceCatalog:         catalog,
				NamespaceDefinition:   pipelineservice.NamespaceDefinitionCustom,
				CustomNamespaceFormat: strPtr("raw_${SOURCE_NAMESPACE}"),
				StreamPrefix:          strPtr("pre_"),
			})
			require.NoError(t, err)

			assert.Equal(t, map[string]int{"public.users": 7, "public.orders": 3}, cursorsByStream(t, blob))
		})
	}
}

func saveBlob(store *mockStorage, connID uuid.UUID, stateType, blob string) {
	store.connStates.states[connID] = &pipelineservice.ConnectionState{ConnectionID: connID, StateType: stateType, StateBlob: blob}
}

func TestGetSyncStateRestoresRewrittenNames(t *testing.T) {
	users := protocol.StreamDescriptor{Name: "users", Namespace: "public"}
	orders := protocol.StreamDescriptor{Name: "orders", Namespace: "public"}

	tests := []struct {
		name      string
		catalog   *protocol.ConfiguredAirbyteCatalog
		nsDef     pipelineservice.NamespaceDefinition
		format    *string
		prefix    *string
		stateType string
		blob      string
		want      map[string]int
	}{
		{
			name:      "stream entries under prefixed and custom namespace names",
			catalog:   sourceCatalog(users, orders),
			nsDef:     pipelineservice.NamespaceDefinitionCustom,
			format:    strPtr("raw_${SOURCE_NAMESPACE}"),
			prefix:    strPtr("pre_"),
			stateType: "stream",
			blob: `[{"type":"stream","stream":{"stream_descriptor":{"name":"pre_users","namespace":"raw_public"},"stream_state":{"cursor":7}}},` +
				`{"type":"stream","stream":{"stream_descriptor":{"name":"pre_orders","namespace":"raw_public"},"stream_state":{"cursor":3}}}]`,
			want: map[string]int{"public.users": 7, "public.orders": 3},
		},
		{
			name:      "global stream_states under prefixed names",
			catalog:   sourceCatalog(users, orders),
			nsDef:     pipelineservice.NamespaceDefinitionSource,
			prefix:    strPtr("pre_"),
			stateType: "global",
			blob: `[{"type":"global","global":{"shared_state":{"lsn":"0/1"},"stream_states":[` +
				`{"stream_descriptor":{"name":"pre_users","namespace":"public"},"stream_state":{"cursor":7}},` +
				`{"stream_descriptor":{"name":"pre_orders","namespace":"public"},"stream_state":{"cursor":3}}]}}]`,
			want: map[string]int{"public.users": 7, "public.orders": 3},
		},
		{
			name:      "entry under the source name wins over the rewritten one",
			catalog:   sourceCatalog(users),
			nsDef:     pipelineservice.NamespaceDefinitionSource,
			prefix:    strPtr("pre_"),
			stateType: "stream",
			blob: `[{"type":"stream","stream":{"stream_descriptor":{"name":"pre_users","namespace":"public"},"stream_state":{"cursor":7}}},` +
				`{"type":"stream","stream":{"stream_descriptor":{"name":"users","namespace":"public"},"stream_state":{"cursor":9}}}]`,
			want: map[string]int{"public.users": 9},
		},
		{
			name:      "global entry under the source name wins over the rewritten one",
			catalog:   sourceCatalog(users),
			nsDef:     pipelineservice.NamespaceDefinitionSource,
			prefix:    strPtr("pre_"),
			stateType: "global",
			blob: `[{"type":"global","global":{"shared_state":{"lsn":"0/1"},"stream_states":[` +
				`{"stream_descriptor":{"name":"pre_users","namespace":"public"},"stream_state":{"cursor":7}},` +
				`{"stream_descriptor":{"name":"users","namespace":"public"},"stream_state":{"cursor":9}}]}}]`,
			want: map[string]int{"public.users": 9},
		},
		{
			name:      "a source stream named like another stream's rewritten name keeps its name",
			catalog:   sourceCatalog(users, protocol.StreamDescriptor{Name: "pre_users", Namespace: "public"}),
			nsDef:     pipelineservice.NamespaceDefinitionSource,
			prefix:    strPtr("pre_"),
			stateType: "stream",
			blob:      `[{"type":"stream","stream":{"stream_descriptor":{"name":"pre_users","namespace":"public"},"stream_state":{"cursor":7}}}]`,
			want:      map[string]int{"public.pre_users": 7},
		},
		{
			name:      "names rewritten under an earlier prefix are kept",
			catalog:   sourceCatalog(users),
			nsDef:     pipelineservice.NamespaceDefinitionSource,
			prefix:    strPtr("new_"),
			stateType: "stream",
			blob:      `[{"type":"stream","stream":{"stream_descriptor":{"name":"old_users","namespace":"public"},"stream_state":{"cursor":7}}}]`,
			want:      map[string]int{"public.old_users": 7},
		},
		{
			name:      "a destination name shared by two source streams is kept",
			catalog:   sourceCatalog(protocol.StreamDescriptor{Name: "users", Namespace: "a"}, protocol.StreamDescriptor{Name: "users", Namespace: "b"}),
			nsDef:     pipelineservice.NamespaceDefinitionDestination,
			stateType: "stream",
			blob:      `[{"type":"stream","stream":{"stream_descriptor":{"name":"users"},"stream_state":{"cursor":7}}}]`,
			want:      map[string]int{".users": 7},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStorage()
			connID := uuid.New()
			saveBlob(store, connID, tt.stateType, tt.blob)

			blob, err := pipelinestate.NewGetSyncState(store).Execute(context.Background(), pipelinestate.GetSyncStateParams{
				ConnectionID:          connID,
				SourceCatalog:         tt.catalog,
				NamespaceDefinition:   tt.nsDef,
				CustomNamespaceFormat: tt.format,
				StreamPrefix:          tt.prefix,
			})
			require.NoError(t, err)

			assert.Equal(t, tt.want, cursorsByStream(t, blob))
		})
	}
}
