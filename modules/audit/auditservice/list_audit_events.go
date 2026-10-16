package auditservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// ListAuditEventsParams filters and paginates the workspace audit log.
type ListAuditEventsParams struct {
	WorkspaceID  uuid.UUID
	ActorID      *uuid.UUID
	ResourceType *ResourceType
	ResourceID   *uuid.UUID
	Action       *Action
	Since        *time.Time
	Until        *time.Time
	Limit        int
	Offset       int
}

// ListedAuditEvent is an audit event with what is known about its actor at read time.
type ListedAuditEvent struct {
	*AuditEvent
	// ActorDeleted is true when the event's user actor no longer exists.
	ActorDeleted bool
}

type ListAuditEvents struct {
	pages EventPageStorage
	users UserDirectory
}

func NewListAuditEvents(pages EventPageStorage, users UserDirectory) *ListAuditEvents {
	return &ListAuditEvents{pages: pages, users: users}
}

func (uc *ListAuditEvents) Execute(ctx context.Context, params ListAuditEventsParams) ([]ListedAuditEvent, error) {
	eventFilter := &AuditEventFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}

	if params.ActorID != nil {
		eventFilter.ActorID = filter.Equals(*params.ActorID)
	}

	if params.ResourceType != nil {
		eventFilter.ResourceType = filter.Equals(*params.ResourceType)
	}

	if params.ResourceID != nil {
		eventFilter.ResourceID = filter.Equals(*params.ResourceID)
	}

	if params.Action != nil {
		eventFilter.Action = filter.Equals(*params.Action)
	}

	switch {
	case params.Since != nil && params.Until != nil:
		eventFilter.CreatedAt = filter.And[time.Time](
			filter.GreaterOrEquals(*params.Since),
			filter.Less(*params.Until),
		)
	case params.Since != nil:
		eventFilter.CreatedAt = filter.GreaterOrEquals(*params.Since)
	case params.Until != nil:
		eventFilter.CreatedAt = filter.Less(*params.Until)
	}

	limit := params.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	events, err := uc.pages.FindPage(ctx, eventFilter, max(params.Offset, 0), limit)
	if err != nil {
		return nil, fmt.Errorf("listing audit events: %w", err)
	}

	deletedActors, err := uc.deletedUserActors(ctx, events)
	if err != nil {
		return nil, err
	}

	listed := make([]ListedAuditEvent, 0, len(events))
	for _, event := range events {
		listed = append(listed, ListedAuditEvent{
			AuditEvent:   event,
			ActorDeleted: event.ActorType == ActorTypeUser && deletedActors[event.ActorID],
		})
	}

	return listed, nil
}

func (uc *ListAuditEvents) deletedUserActors(ctx context.Context, events []*AuditEvent) (map[uuid.UUID]bool, error) {
	deleted := make(map[uuid.UUID]bool)
	checked := make(map[uuid.UUID]bool)

	for _, event := range events {
		if event.ActorType != ActorTypeUser || checked[event.ActorID] {
			continue
		}

		checked[event.ActorID] = true

		exists, err := uc.users.UserExists(ctx, event.ActorID)
		if err != nil {
			return nil, fmt.Errorf("checking audit actor %s: %w", event.ActorID, err)
		}

		if !exists {
			deleted[event.ActorID] = true
		}
	}

	return deleted, nil
}
