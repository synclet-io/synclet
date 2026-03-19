package pipelineroute

import (
	"context"

	"github.com/synclet-io/synclet/pkg/protocol"
)

// Handler defines callbacks for mode-specific side effects during message routing.
// The router owns RECORD forwarding, stats counting, LOG output, and STATE forwarding.
// Handler only receives callbacks for mode-specific side effects.
type Handler interface {
	// OnStateConfirmed is called when the destination emits a STATE message,
	// confirming all preceding records are committed.
	OnStateConfirmed(ctx context.Context, msg *protocol.AirbyteStateMessage) error

	// OnSourceControl is called when the source emits a CONTROL message.
	OnSourceControl(ctx context.Context, msg *protocol.AirbyteControlMessage) error

	// OnDestControl is called when the destination emits a CONTROL message.
	OnDestControl(ctx context.Context, msg *protocol.AirbyteControlMessage) error

	// OnSourceTrace is called when the source emits a TRACE error message.
	OnSourceTrace(ctx context.Context, msg *protocol.AirbyteTraceMessage) error

	// OnLog is called when the source or destination emits a LOG or TRACE message.
	// The line is pre-formatted text (e.g., "[src] INFO: message").
	OnLog(ctx context.Context, line string) error
}

// DefaultHandler provides no-op implementations of all Handler methods.
// Embed it in concrete handlers to only override the methods you need.
type DefaultHandler struct{}

func (DefaultHandler) OnStateConfirmed(context.Context, *protocol.AirbyteStateMessage) error {
	return nil
}
func (DefaultHandler) OnSourceControl(context.Context, *protocol.AirbyteControlMessage) error {
	return nil
}
func (DefaultHandler) OnDestControl(context.Context, *protocol.AirbyteControlMessage) error {
	return nil
}
func (DefaultHandler) OnSourceTrace(context.Context, *protocol.AirbyteTraceMessage) error {
	return nil
}
func (DefaultHandler) OnLog(context.Context, string) error { return nil }
