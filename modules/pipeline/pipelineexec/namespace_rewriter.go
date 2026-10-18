package pipelineexec

import (
	"log/slog"
	"strings"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// streamKey identifies a stream by its original namespace and name.
type streamKey struct {
	namespace string
	name      string
}

// NamespaceRewriter rewrites namespace and stream name in RECORD and STATE
// messages to match the destination catalog's namespace/prefix settings, and
// maps the STATE the destination confirms back to the source catalog's names.
// It pre-computes both mappings from the source catalog so lookups are O(1).
type NamespaceRewriter struct {
	mapping      map[streamKey]streamKey
	reverse      map[streamKey]streamKey
	nsDef        pipelineservice.NamespaceDefinition
	customFormat *string
	prefix       *string
}

// NewNamespaceRewriter builds a rewriter from the source catalog and connection
// namespace settings. The mapping is computed once and reused for all messages.
func NewNamespaceRewriter(
	catalog *protocol.ConfiguredAirbyteCatalog,
	nsDef pipelineservice.NamespaceDefinition,
	customFormat *string,
	prefix *string,
) *NamespaceRewriter {
	rewriter := &NamespaceRewriter{
		mapping:      make(map[streamKey]streamKey, len(catalog.Streams)),
		reverse:      make(map[streamKey]streamKey, len(catalog.Streams)),
		nsDef:        nsDef,
		customFormat: customFormat,
		prefix:       prefix,
	}

	for _, configuredStream := range catalog.Streams {
		srcKey := streamKey{
			namespace: configuredStream.Stream.Namespace,
			name:      configuredStream.Stream.Name,
		}
		rewriter.mapping[srcKey] = streamKey{
			namespace: rewriter.rewriteNamespace(configuredStream.Stream.Namespace),
			name:      rewriter.rewriteName(configuredStream.Stream.Name),
		}
	}

	// A destination key shared by several source streams has no single source
	// key, so it is left out of the reverse mapping.
	ambiguous := make(map[streamKey]bool)

	for srcKey, dstKey := range rewriter.mapping {
		if _, taken := rewriter.reverse[dstKey]; taken {
			ambiguous[dstKey] = true
		}

		rewriter.reverse[dstKey] = srcKey
	}

	for dstKey := range ambiguous {
		delete(rewriter.reverse, dstKey)
	}

	return rewriter
}

// RewriteRecord rewrites the stream name and namespace on a RECORD message.
// No-op if msg.Record is nil.
func (r *NamespaceRewriter) RewriteRecord(msg *protocol.AirbyteMessage) {
	if msg.Record == nil {
		return
	}

	src := streamKey{namespace: msg.Record.Namespace, name: msg.Record.Stream}
	if target, ok := r.mapping[src]; ok {
		msg.Record.Stream = target.name
		msg.Record.Namespace = target.namespace

		return
	}

	slog.Warn("rewriter: mapping miss, using fallback", "namespace", src.namespace, "name", src.name)
	// Fallback for streams not in catalog.
	msg.Record.Stream = r.rewriteName(msg.Record.Stream)
	msg.Record.Namespace = r.rewriteNamespace(msg.Record.Namespace)
}

// RewriteState rewrites stream descriptors in a STATE message.
// Handles stream, global, and legacy (no-op) state types.
// No-op if msg.State is nil.
func (r *NamespaceRewriter) RewriteState(msg *protocol.AirbyteMessage) {
	if msg.State == nil {
		return
	}

	switch protocol.NormalizeStateType(msg.State.Type) {
	case protocol.StateTypeStream:
		if msg.State.Stream != nil {
			r.rewriteDescriptor(&msg.State.Stream.StreamDescriptor)
		}
	case protocol.StateTypeGlobal:
		if msg.State.Global != nil {
			for i := range msg.State.Global.StreamStates {
				r.rewriteDescriptor(&msg.State.Global.StreamStates[i].StreamDescriptor)
			}
		}
		// Legacy state has no stream descriptors -- no-op.
	}
}

// RestoreState maps the stream descriptors of a STATE message confirmed by the
// destination back to the source catalog's names and namespaces, for STREAM and
// GLOBAL state. A descriptor that no single catalog stream maps to is kept.
func (r *NamespaceRewriter) RestoreState(state *protocol.AirbyteStateMessage) {
	if state == nil {
		return
	}

	switch protocol.NormalizeStateType(state.Type) {
	case protocol.StateTypeStream:
		if state.Stream != nil {
			r.restoreDescriptor(&state.Stream.StreamDescriptor)
		}
	case protocol.StateTypeGlobal:
		if state.Global != nil {
			for i := range state.Global.StreamStates {
				r.restoreDescriptor(&state.Global.StreamStates[i].StreamDescriptor)
			}
		}
	}
}

func (r *NamespaceRewriter) restoreDescriptor(desc *protocol.StreamDescriptor) {
	src, ok := r.reverse[streamKey{namespace: desc.Namespace, name: desc.Name}]
	if !ok {
		slog.Warn("rewriter: no source stream for confirmed state", "namespace", desc.Namespace, "name", desc.Name)

		return
	}

	desc.Name = src.name
	desc.Namespace = src.namespace
}

// RewriteTrace rewrites stream descriptors in TRACE messages (stream_status,
// error) so the destination can match them against its catalog.
// No-op if msg.Trace is nil.
func (r *NamespaceRewriter) RewriteTrace(msg *protocol.AirbyteMessage) {
	if msg.Trace == nil {
		return
	}

	switch msg.Trace.Type {
	case protocol.TraceTypeStreamStatus:
		if msg.Trace.StreamStatus != nil {
			r.rewriteDescriptor(&msg.Trace.StreamStatus.StreamDescriptor)
		}
	case protocol.TraceTypeError:
		if msg.Trace.Error != nil && msg.Trace.Error.StreamDescriptor != nil {
			r.rewriteDescriptor(msg.Trace.Error.StreamDescriptor)
		}
	}
}

// rewriteDescriptor applies the namespace/prefix mapping to a StreamDescriptor.
func (r *NamespaceRewriter) rewriteDescriptor(desc *protocol.StreamDescriptor) {
	src := streamKey{namespace: desc.Namespace, name: desc.Name}
	if target, ok := r.mapping[src]; ok {
		desc.Name = target.name
		desc.Namespace = target.namespace

		return
	}

	// Fallback for descriptors not in catalog.
	desc.Name = r.rewriteName(desc.Name)
	desc.Namespace = r.rewriteNamespace(desc.Namespace)
}

// rewriteName prepends the stream prefix if configured.
func (r *NamespaceRewriter) rewriteName(name string) string {
	if r.prefix != nil && *r.prefix != "" {
		return *r.prefix + name
	}

	return name
}

// rewriteNamespace applies namespace rewriting per the namespace definition.
func (r *NamespaceRewriter) rewriteNamespace(originalNamespace string) string {
	switch r.nsDef {
	case pipelineservice.NamespaceDefinitionSource:
		return originalNamespace
	case pipelineservice.NamespaceDefinitionDestination:
		return ""
	case pipelineservice.NamespaceDefinitionCustom:
		if r.customFormat != nil {
			return strings.ReplaceAll(*r.customFormat, "${SOURCE_NAMESPACE}", originalNamespace)
		}

		return originalNamespace
	default:
		return originalNamespace
	}
}
