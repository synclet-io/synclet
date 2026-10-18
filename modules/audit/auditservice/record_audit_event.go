package auditservice

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"

	"github.com/synclet-io/synclet/pkg/auditutil"
)

// RecordAuditEventParams holds the input for a single audit event write.
// Callers compute the before/after snapshots themselves so the audit module
// stays decoupled from each consuming module's domain types.
type RecordAuditEventParams struct {
	WorkspaceID   uuid.UUID
	ActorType     ActorType
	ActorID       uuid.UUID
	ActorLabel    string
	Action        Action
	ResourceType  ResourceType
	ResourceID    uuid.UUID
	ResourceLabel string
	// Before / After may be any JSON-serialisable value; both may be nil
	// (e.g. on create, Before is nil; on delete, After is nil).
	Before any
	After  any
}

// RecordAuditEvent builds a single audit event and hands it to the outbox;
// the bus consumer persists it. Failure to enqueue is logged but never
// returned — an audit-write failure must not unwind the business operation
// that produced it.
type RecordAuditEvent struct {
	storage Storage
	users   UserDirectory
	logger  *logging.Logger
}

func NewRecordAuditEvent(storage Storage, users UserDirectory, logger *logging.Logger) *RecordAuditEvent {
	return &RecordAuditEvent{storage: storage, users: users, logger: logger.Named("record-audit-event")}
}

func (uc *RecordAuditEvent) Execute(ctx context.Context, params RecordAuditEventParams) {
	if !params.Action.IsValid() || !params.ResourceType.IsValid() || !params.ActorType.IsValid() {
		uc.logger.WithField("action", params.Action).WithField("resource_type", params.ResourceType).Error(ctx, "skipping audit event with invalid enum values")

		return
	}

	if params.ActorType == ActorTypeUser && params.ActorLabel == "" {
		params.ActorLabel = uc.userEmail(ctx, params.ActorID)
	}

	changes := auditutil.Diff(params.Before, params.After)

	changes, truncated := auditutil.TruncateDiff(changes)

	diffJSON, err := json.Marshal(changes)
	if err != nil {
		uc.logger.WithError(err).Error(ctx, "encoding audit diff failed; recording empty diff")

		diffJSON = []byte("[]")
	}

	record := &AuditEvent{
		ID:            uuid.New(),
		WorkspaceID:   params.WorkspaceID,
		ActorType:     params.ActorType,
		ActorID:       params.ActorID,
		ActorLabel:    params.ActorLabel,
		Action:        params.Action,
		ResourceType:  params.ResourceType,
		ResourceID:    params.ResourceID,
		ResourceLabel: params.ResourceLabel,
		DiffJSON:      string(diffJSON),
		DiffTruncated: truncated,
		CreatedAt:     time.Now(),
	}

	event := &AuditOutboxEvent{
		EventType: AuditOutboxEventTypeRecorded,
		Data:      &AuditEventRecordedData{Event: record},
	}

	if err := uc.storage.AuditOutboxEvents().Send(ctx, event); err != nil {
		uc.logger.WithError(err).
			WithField("workspace_id", params.WorkspaceID).
			WithField("resource_type", params.ResourceType).
			WithField("action", params.Action).
			Error(ctx, "failed to enqueue audit event")
	}
}

// Record enqueues the audit event described by params.
func (uc *RecordAuditEvent) Record(ctx context.Context, params RecordAuditEventParams) {
	uc.Execute(ctx, params)
}

// userEmail resolves a user actor's email for the event's actor label; a failed lookup
// leaves the label empty rather than dropping the event.
func (uc *RecordAuditEvent) userEmail(ctx context.Context, userID uuid.UUID) string {
	email, err := uc.users.UserEmail(ctx, userID)
	if err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Warn(ctx, "resolving audit actor email failed")

		return ""
	}

	return email
}
