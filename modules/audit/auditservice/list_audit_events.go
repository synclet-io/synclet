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

type ListAuditEvents struct {
	pages EventPageStorage
}

func NewListAuditEvents(pages EventPageStorage) *ListAuditEvents {
	return &ListAuditEvents{pages: pages}
}

func (uc *ListAuditEvents) Execute(ctx context.Context, params ListAuditEventsParams) ([]*AuditEvent, error) {
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

	return events, nil
}
