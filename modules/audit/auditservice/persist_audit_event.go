package auditservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// PersistAuditEvent inserts an audit event delivered through the bus. Execute
// is idempotent: an event whose id is already stored is a success, so a
// redelivered message never fails. Events that can never be stored are
// rejected with a ValidationError.
//
// An event with a nil WorkspaceID is pending: a user-level event of a user
// who belongs to no workspace yet. When a user creates or joins a workspace,
// their pending events move into it.
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
	if err != nil && !errors.Is(err, ErrAuditEventAlreadyExists) {
		return fmt.Errorf("persisting audit event: %w", err)
	}

	if userID, ok := joinedUser(event); ok {
		if err := uc.adoptPendingEvents(ctx, userID, event.WorkspaceID); err != nil {
			return err
		}
	}

	return nil
}

// joinedUser returns the user that event adds to its workspace: the creator
// of a new workspace or a newly added member.
func joinedUser(event *AuditEvent) (uuid.UUID, bool) {
	if event.WorkspaceID == uuid.Nil || event.Action != ActionCreate {
		return uuid.Nil, false
	}

	switch event.ResourceType {
	case ResourceTypeWorkspace:
		if event.ActorType == ActorTypeUser && event.ActorID != uuid.Nil {
			return event.ActorID, true
		}
	case ResourceTypeWorkspaceMember:
		if event.ResourceID != uuid.Nil {
			return event.ResourceID, true
		}
	}

	return uuid.Nil, false
}

func (uc *PersistAuditEvent) adoptPendingEvents(ctx context.Context, userID, workspaceID uuid.UUID) error {
	pending, err := uc.storage.AuditEvents().Find(ctx, &AuditEventFilter{
		WorkspaceID: filter.Equals(uuid.Nil),
		ActorType:   filter.Equals(ActorTypeUser),
		ActorID:     filter.Equals(userID),
	})
	if err != nil {
		return fmt.Errorf("finding pending audit events: %w", err)
	}

	for _, event := range pending {
		event.WorkspaceID = workspaceID

		if _, err := uc.storage.AuditEvents().Update(ctx, event); err != nil {
			return fmt.Errorf("moving pending audit event into workspace: %w", err)
		}
	}

	return nil
}
