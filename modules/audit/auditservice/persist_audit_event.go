package auditservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
)

// PersistAuditEvent inserts an audit event delivered through the bus. Execute
// is idempotent: an event whose id is already stored is a success, so a
// redelivered message never fails. Events that can never be stored are
// rejected with a ValidationError.
//
// An event with a nil WorkspaceID is pending: a user-level event of a user
// who belongs to no workspace. Only the registration and the logins before the
// user's first membership leave that state: they move into the workspace the
// user creates or joins first, or, if that move did not happen, into the
// workspace of the user's next login. A user whose registration is no longer
// pending gets nothing moved, so the activity of a user removed from their last
// workspace never reaches a workspace they join later.
type PersistAuditEvent struct {
	storage Storage
	pending PendingEventStorage
	logger  *logging.Logger
}

func NewPersistAuditEvent(storage Storage, pending PendingEventStorage, logger *logging.Logger) *PersistAuditEvent {
	return &PersistAuditEvent{storage: storage, pending: pending, logger: logger.Named("persist-audit-event")}
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

	if userID, ok := adoptingUser(event); ok {
		if err := uc.adoptPendingEvents(ctx, userID, event); err != nil {
			return err
		}
	}

	return nil
}

// adoptingUser returns the user whose pending events event can adopt: the
// creator of a new workspace, a newly added member, or a user logging in to a
// workspace.
func adoptingUser(event *AuditEvent) (uuid.UUID, bool) {
	if event.WorkspaceID == uuid.Nil {
		return uuid.Nil, false
	}

	switch {
	case event.ResourceType == ResourceTypeWorkspace && event.Action == ActionCreate,
		event.ResourceType == ResourceTypeUser && event.Action == ActionLogin:
		if event.ActorType == ActorTypeUser && event.ActorID != uuid.Nil {
			return event.ActorID, true
		}
	case event.ResourceType == ResourceTypeWorkspaceMember && event.Action == ActionCreate:
		if event.ResourceID != uuid.Nil {
			return event.ResourceID, true
		}
	}

	return uuid.Nil, false
}

func (uc *PersistAuditEvent) adoptPendingEvents(ctx context.Context, userID uuid.UUID, event *AuditEvent) error {
	if _, err := uc.pending.AdoptPendingEvents(ctx, userID, event.WorkspaceID, event.CreatedAt); err != nil {
		uc.logger.WithError(err).WithFields(map[string]interface{}{
			"user_id":      userID.String(),
			"workspace_id": event.WorkspaceID.String(),
		}).Error(ctx, "moving pending audit events of user into workspace failed")

		return fmt.Errorf("moving pending audit events of user %s into workspace: %w", userID, err)
	}

	return nil
}
