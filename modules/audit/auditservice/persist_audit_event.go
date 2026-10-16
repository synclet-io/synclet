package auditservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// PersistAuditEvent inserts an audit event delivered through the bus. Execute
// is idempotent: an event whose id is already stored is a success, so a
// redelivered message never fails. Events that can never be stored are
// rejected with a ValidationError.
type PersistAuditEvent struct {
	storage Storage
}

func NewPersistAuditEvent(storage Storage) *PersistAuditEvent {
	return &PersistAuditEvent{storage: storage}
}

func (uc *PersistAuditEvent) Execute(ctx context.Context, event *AuditEvent) error {
	if event == nil {
		return &ValidationError{Message: "audit event is nil"}
	}

	if event.ID == uuid.Nil {
		return &ValidationError{Message: "audit event id is empty"}
	}

	if !event.Action.IsValid() || !event.ResourceType.IsValid() || !event.ActorType.IsValid() {
		return &ValidationError{Message: "audit event has invalid enum values"}
	}

	_, err := uc.storage.AuditEvents().Create(ctx, event)
	if errors.Is(err, ErrAuditEventAlreadyExists) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("persisting audit event: %w", err)
	}

	return nil
}
