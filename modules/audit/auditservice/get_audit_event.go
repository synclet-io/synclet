package auditservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

type GetAuditEventParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

type GetAuditEvent struct {
	storage Storage
	users   UserDirectory
}

func NewGetAuditEvent(storage Storage, users UserDirectory) *GetAuditEvent {
	return &GetAuditEvent{storage: storage, users: users}
}

func (uc *GetAuditEvent) Execute(ctx context.Context, params GetAuditEventParams) (*ListedAuditEvent, error) {
	event, err := uc.storage.AuditEvents().First(ctx, &AuditEventFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting audit event: %w", err)
	}

	deleted, err := deletedUserActors(ctx, uc.users, []*AuditEvent{event})
	if err != nil {
		return nil, err
	}

	return &ListedAuditEvent{
		AuditEvent:   event,
		ActorDeleted: event.ActorType == ActorTypeUser && deleted[event.ActorID],
	}, nil
}
