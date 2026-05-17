package auditservice

import (
	"context"
	"encoding/json"
	"fmt"
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

// RecordAuditEvent persists a single audit event. Failure to persist is
// logged but never returned — an audit-write failure must not unwind the
// business operation that produced it.
type RecordAuditEvent struct {
	storage Storage
	logger  *logging.Logger
}

func NewRecordAuditEvent(storage Storage, logger *logging.Logger) *RecordAuditEvent {
	return &RecordAuditEvent{storage: storage, logger: logger.Named("record-audit-event")}
}

func (uc *RecordAuditEvent) Execute(ctx context.Context, params RecordAuditEventParams) {
	if !params.Action.IsValid() || !params.ResourceType.IsValid() || !params.ActorType.IsValid() {
		uc.logger.WithField("action", params.Action).WithField("resource_type", params.ResourceType).Error(ctx, "skipping audit event with invalid enum values")

		return
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

	if _, err := uc.storage.AuditEvents().Create(ctx, record); err != nil {
		uc.logger.WithError(err).
			WithField("workspace_id", params.WorkspaceID).
			WithField("resource_type", params.ResourceType).
			WithField("action", params.Action).
			Error(ctx, fmt.Sprintf("failed to persist audit event: %s", err))
	}
}
