package authservice

import (
	"context"

	"github.com/google/uuid"
)

// AuditEvent is the wire-thin payload the auth module emits when a mutating
// or session operation succeeds. Decoupled from the audit module's domain
// types so this package does not depend on `modules/audit`; the adapter (in
// `authadapt`) maps this into the audit module's own event type.
//
// ActorID identifies the acting user when the request context has no
// authenticated user yet (login, registration). A zero WorkspaceID marks a
// user-level event that belongs to every workspace the actor is a member of.
type AuditEvent struct {
	WorkspaceID   uuid.UUID
	ActorID       uuid.UUID
	ActorLabel    string
	Action        string
	ResourceType  string
	ResourceID    uuid.UUID
	ResourceLabel string
	Before        any
	After         any
}

// AuditRecorder is the port the auth module uses to write audit events.
// Implementations must be best-effort — Record never returns an error.
type AuditRecorder interface {
	Record(ctx context.Context, event AuditEvent)
}

// NoopAuditRecorder drops every event. Used in tests and code paths where
// the audit feature is not yet wired.
type NoopAuditRecorder struct{}

func (NoopAuditRecorder) Record(_ context.Context, _ AuditEvent) {}
