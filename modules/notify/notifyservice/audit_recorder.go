package notifyservice

import (
	"context"

	"github.com/google/uuid"
)

// AuditEvent is the wire-thin payload the notify module emits when a mutating
// operation succeeds. Decoupled from the audit module's domain types so this
// package does not depend on `modules/audit`; the audit adapter (in
// `notifyadapt`) maps this into the audit module's own event type.
type AuditEvent struct {
	WorkspaceID   uuid.UUID
	Action        string
	ResourceType  string
	ResourceID    uuid.UUID
	ResourceLabel string
	Before        any
	After         any
}

// AuditRecorder is the port the notify module uses to write audit events.
// Implementations must be best-effort — Record never returns an error, so a
// recorder failure does not unwind the business operation that produced it.
type AuditRecorder interface {
	Record(ctx context.Context, event AuditEvent)
}

// NoopAuditRecorder drops every event. Useful in tests and code paths where
// the audit feature is not yet wired.
type NoopAuditRecorder struct{}

func (NoopAuditRecorder) Record(_ context.Context, _ AuditEvent) {}
