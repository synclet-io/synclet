package pipelineroute

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

func strPtr(s string) *string { return &s }

func makeCatalog(streams ...protocol.ConfiguredAirbyteStream) *protocol.ConfiguredAirbyteCatalog {
	return &protocol.ConfiguredAirbyteCatalog{Streams: streams}
}

func makeStream(name, namespace string) protocol.ConfiguredAirbyteStream {
	return protocol.ConfiguredAirbyteStream{
		Stream: protocol.AirbyteStream{
			Name:      name,
			Namespace: namespace,
		},
		SyncMode:            protocol.SyncModeFullRefresh,
		DestinationSyncMode: protocol.DestinationSyncModeAppend,
	}
}

func makeRecordMsg(stream, namespace string) *protocol.AirbyteMessage {
	return &protocol.AirbyteMessage{
		Type: protocol.MessageTypeRecord,
		Record: &protocol.AirbyteRecordMessage{
			Stream:    stream,
			Namespace: namespace,
			Data:      json.RawMessage(`{}`),
		},
	}
}

func makeStreamStateMsg(streamName, namespace string) *protocol.AirbyteMessage {
	return &protocol.AirbyteMessage{
		Type: protocol.MessageTypeState,
		State: &protocol.AirbyteStateMessage{
			Type: protocol.StateTypeStream,
			Stream: &protocol.AirbyteStreamState{
				StreamDescriptor: protocol.StreamDescriptor{
					Name:      streamName,
					Namespace: namespace,
				},
				StreamState: json.RawMessage(`{"cursor":"abc"}`),
			},
		},
	}
}

func makeGlobalStateMsg(descriptors ...protocol.StreamDescriptor) *protocol.AirbyteMessage {
	states := make([]protocol.AirbyteStreamState, len(descriptors))
	for i, d := range descriptors {
		states[i] = protocol.AirbyteStreamState{
			StreamDescriptor: d,
			StreamState:      json.RawMessage(`{}`),
		}
	}

	return &protocol.AirbyteMessage{
		Type: protocol.MessageTypeState,
		State: &protocol.AirbyteStateMessage{
			Type: protocol.StateTypeGlobal,
			Global: &protocol.AirbyteGlobalState{
				StreamStates: states,
			},
		},
	}
}

func makeLegacyStateMsg() *protocol.AirbyteMessage {
	return &protocol.AirbyteMessage{
		Type: protocol.MessageTypeState,
		State: &protocol.AirbyteStateMessage{
			Type: protocol.StateTypeLegacy,
			Data: json.RawMessage(`{"position":42}`),
		},
	}
}

func TestNamespaceRewriter_Source(t *testing.T) {
	catalog := makeCatalog(
		makeStream("users", "public"),
		makeStream("orders", "sales"),
	)
	rewriter := NewNamespaceRewriter(catalog, pipelineservice.NamespaceDefinitionSource, nil, strPtr("dst_"))

	t.Run("RewriteRecord prepends prefix, keeps namespace", func(t *testing.T) {
		msg := makeRecordMsg("users", "public")
		rewriter.RewriteRecord(msg)
		assert.Equal(t, "dst_users", msg.Record.Stream)
		assert.Equal(t, "public", msg.Record.Namespace)
	})

	t.Run("RewriteState stream type prepends prefix, keeps namespace", func(t *testing.T) {
		msg := makeStreamStateMsg("orders", "sales")
		rewriter.RewriteState(msg)
		assert.Equal(t, "dst_orders", msg.State.Stream.StreamDescriptor.Name)
		assert.Equal(t, "sales", msg.State.Stream.StreamDescriptor.Namespace)
	})
}

func TestNamespaceRewriter_Destination(t *testing.T) {
	catalog := makeCatalog(
		makeStream("users", "public"),
	)
	rewriter := NewNamespaceRewriter(catalog, pipelineservice.NamespaceDefinitionDestination, nil, strPtr("pre_"))

	t.Run("RewriteRecord clears namespace, prepends prefix", func(t *testing.T) {
		msg := makeRecordMsg("users", "public")
		rewriter.RewriteRecord(msg)
		assert.Equal(t, "pre_users", msg.Record.Stream)
		assert.Empty(t, msg.Record.Namespace)
	})

	t.Run("RewriteState stream type clears namespace, prepends prefix", func(t *testing.T) {
		msg := makeStreamStateMsg("users", "public")
		rewriter.RewriteState(msg)
		assert.Equal(t, "pre_users", msg.State.Stream.StreamDescriptor.Name)
		assert.Empty(t, msg.State.Stream.StreamDescriptor.Namespace)
	})
}

func TestNamespaceRewriter_Custom(t *testing.T) {
	catalog := makeCatalog(
		makeStream("users", "public"),
		makeStream("orders", "sales"),
	)
	customFmt := "custom_${SOURCE_NAMESPACE}"
	rewriter := NewNamespaceRewriter(catalog, pipelineservice.NamespaceDefinitionCustom, &customFmt, strPtr("p_"))

	t.Run("RewriteRecord replaces namespace using custom format", func(t *testing.T) {
		msg := makeRecordMsg("users", "public")
		rewriter.RewriteRecord(msg)
		assert.Equal(t, "p_users", msg.Record.Stream)
		assert.Equal(t, "custom_public", msg.Record.Namespace)
	})

	t.Run("RewriteState global type rewrites all stream descriptors", func(t *testing.T) {
		msg := makeGlobalStateMsg(
			protocol.StreamDescriptor{Name: "users", Namespace: "public"},
			protocol.StreamDescriptor{Name: "orders", Namespace: "sales"},
		)
		rewriter.RewriteState(msg)
		require.Len(t, msg.State.Global.StreamStates, 2)
		assert.Equal(t, "p_users", msg.State.Global.StreamStates[0].StreamDescriptor.Name)
		assert.Equal(t, "custom_public", msg.State.Global.StreamStates[0].StreamDescriptor.Namespace)
		assert.Equal(t, "p_orders", msg.State.Global.StreamStates[1].StreamDescriptor.Name)
		assert.Equal(t, "custom_sales", msg.State.Global.StreamStates[1].StreamDescriptor.Namespace)
	})
}

func TestNamespaceRewriter_LegacyState(t *testing.T) {
	catalog := makeCatalog(makeStream("users", "public"))
	rewriter := NewNamespaceRewriter(catalog, pipelineservice.NamespaceDefinitionDestination, nil, strPtr("p_"))

	msg := makeLegacyStateMsg()
	rewriter.RewriteState(msg) // Should be a no-op, no panic.
	assert.Equal(t, protocol.StateTypeLegacy, msg.State.Type)
	assert.Nil(t, msg.State.Stream)
	assert.Nil(t, msg.State.Global)
}

func TestNamespaceRewriter_NilPointers(t *testing.T) {
	catalog := makeCatalog(makeStream("users", "public"))
	rewriter := NewNamespaceRewriter(catalog, pipelineservice.NamespaceDefinitionSource, nil, nil)

	t.Run("nil record is no-op", func(t *testing.T) {
		msg := &protocol.AirbyteMessage{Type: protocol.MessageTypeRecord}
		rewriter.RewriteRecord(msg) // No panic.
	})

	t.Run("nil state is no-op", func(t *testing.T) {
		msg := &protocol.AirbyteMessage{Type: protocol.MessageTypeState}
		rewriter.RewriteState(msg) // No panic.
	})
}

func TestNamespaceRewriter_NoPrefix(t *testing.T) {
	catalog := makeCatalog(makeStream("users", "public"))
	rewriter := NewNamespaceRewriter(catalog, pipelineservice.NamespaceDefinitionDestination, nil, nil)

	msg := makeRecordMsg("users", "public")
	rewriter.RewriteRecord(msg)
	assert.Equal(t, "users", msg.Record.Stream) // No prefix applied.
	assert.Empty(t, msg.Record.Namespace)       // Namespace cleared for Destination.
}

func TestNamespaceRewriter_FallbackForUnknownStream(t *testing.T) {
	// Stream not in catalog should still get prefix/namespace rewriting.
	catalog := makeCatalog(makeStream("users", "public"))
	rewriter := NewNamespaceRewriter(catalog, pipelineservice.NamespaceDefinitionDestination, nil, strPtr("x_"))

	msg := makeRecordMsg("unknown_stream", "some_ns")
	rewriter.RewriteRecord(msg)
	assert.Equal(t, "x_unknown_stream", msg.Record.Stream)
	assert.Empty(t, msg.Record.Namespace) // Destination clears namespace.
}
