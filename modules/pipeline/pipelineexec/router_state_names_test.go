package pipelineexec

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

type confirmedStates struct {
	DefaultHandler

	mu     sync.Mutex
	states []*protocol.AirbyteStateMessage
}

func (h *confirmedStates) OnStateConfirmed(_ context.Context, msg *protocol.AirbyteStateMessage) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.states = append(h.states, msg)

	return nil
}

// echoDestination echoes every STATE it reads and records the descriptors it saw.
func echoDestination(stdin io.Reader, stdout *io.PipeWriter, seen *[]protocol.StreamDescriptor) {
	reader := protocol.NewMessageReader(stdin, protocol.SyncMessageMaxSize)
	writer := protocol.NewMessageWriter(stdout)

	defer func() { _ = stdout.Close() }()

	for {
		msg, err := reader.Read()
		if err != nil {
			return
		}

		if msg.Type != protocol.MessageTypeState {
			continue
		}

		if msg.State.Stream != nil {
			*seen = append(*seen, msg.State.Stream.StreamDescriptor)
		}

		if msg.State.Global != nil {
			for _, s := range msg.State.Global.StreamStates {
				*seen = append(*seen, s.StreamDescriptor)
			}
		}

		if err := writer.Write(msg); err != nil {
			return
		}
	}
}

func TestRun_ConfirmedStateCarriesSourceNames(t *testing.T) {
	catalog := makeCatalog(makeStream("users", "public"), makeStream("orders", "public"))
	rewriter := NewNamespaceRewriter(catalog, pipelineservice.NamespaceDefinitionCustom, strPtr("raw_${SOURCE_NAMESPACE}"), strPtr("pre_"))

	source := `{"type":"STATE","state":{"type":"STREAM","stream":{"stream_descriptor":{"name":"users","namespace":"public"},"stream_state":{"cursor":7}}}}` + "\n" +
		`{"type":"STATE","state":{"type":"GLOBAL","global":{"shared_state":{"lsn":"0/1"},"stream_states":[{"stream_descriptor":{"name":"users","namespace":"public"},"stream_state":{"cursor":8}},{"stream_descriptor":{"name":"orders","namespace":"public"},"stream_state":{"cursor":9}}]}}}` + "\n"

	destStdinReader, destStdinWriter := io.Pipe()
	destStdoutReader, destStdoutWriter := io.Pipe()

	var seen []protocol.StreamDescriptor

	destDone := make(chan struct{})

	go func() {
		echoDestination(destStdinReader, destStdoutWriter, &seen)
		close(destDone)
	}()

	handler := &confirmedStates{}

	_, err := Run(context.Background(), strings.NewReader(source), destStdinWriter, destStdoutReader, handler, RunConfig{Rewriter: rewriter}, nil)
	require.NoError(t, err)
	<-destDone

	users := protocol.StreamDescriptor{Name: "users", Namespace: "public"}
	orders := protocol.StreamDescriptor{Name: "orders", Namespace: "public"}

	assert.Equal(t, []protocol.StreamDescriptor{
		{Name: "pre_users", Namespace: "raw_public"},
		{Name: "pre_users", Namespace: "raw_public"},
		{Name: "pre_orders", Namespace: "raw_public"},
	}, seen, "destination gets the rewritten names")

	require.Len(t, handler.states, 2)

	streamState := handler.states[0]
	require.NotNil(t, streamState.Stream)
	assert.Equal(t, users, streamState.Stream.StreamDescriptor)
	assert.JSONEq(t, `{"cursor":7}`, string(streamState.Stream.StreamState))

	globalState := handler.states[1]
	require.NotNil(t, globalState.Global)
	require.Len(t, globalState.Global.StreamStates, 2)
	assert.Equal(t, users, globalState.Global.StreamStates[0].StreamDescriptor)
	assert.Equal(t, orders, globalState.Global.StreamStates[1].StreamDescriptor)
	assert.JSONEq(t, `{"lsn":"0/1"}`, string(globalState.Global.SharedState))
}

func TestNamespaceRewriter_RestoreState(t *testing.T) {
	t.Run("maps destination names back to source names", func(t *testing.T) {
		rewriter := NewNamespaceRewriter(makeCatalog(makeStream("users", "public")), pipelineservice.NamespaceDefinitionSource, nil, strPtr("pre_"))

		state := makeStreamStateMsg("pre_users", "public").State
		rewriter.RestoreState(state)

		assert.Equal(t, protocol.StreamDescriptor{Name: "users", Namespace: "public"}, state.Stream.StreamDescriptor)
	})

	t.Run("leaves a destination name shared by two source streams as it is", func(t *testing.T) {
		catalog := makeCatalog(makeStream("users", "a"), makeStream("users", "b"))
		rewriter := NewNamespaceRewriter(catalog, pipelineservice.NamespaceDefinitionDestination, nil, nil)

		state := makeGlobalStateMsg(protocol.StreamDescriptor{Name: "users"}).State
		rewriter.RestoreState(state)

		assert.Equal(t, protocol.StreamDescriptor{Name: "users"}, state.Global.StreamStates[0].StreamDescriptor)
	})

	t.Run("leaves a name outside the catalog as it is", func(t *testing.T) {
		rewriter := NewNamespaceRewriter(makeCatalog(makeStream("users", "public")), pipelineservice.NamespaceDefinitionSource, nil, strPtr("pre_"))

		state := makeStreamStateMsg("pre_unknown", "public").State
		rewriter.RestoreState(state)

		assert.Equal(t, protocol.StreamDescriptor{Name: "pre_unknown", Namespace: "public"}, state.Stream.StreamDescriptor)
	})

	t.Run("ignores legacy and nil state", func(t *testing.T) {
		rewriter := NewNamespaceRewriter(makeCatalog(makeStream("users", "public")), pipelineservice.NamespaceDefinitionSource, nil, strPtr("pre_"))

		legacy := makeLegacyStateMsg().State
		rewriter.RestoreState(legacy)
		rewriter.RestoreState(nil)

		assert.Nil(t, legacy.Stream)
	})
}
