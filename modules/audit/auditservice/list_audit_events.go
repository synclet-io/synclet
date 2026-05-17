package auditservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/saturn4er/boilerplate-go/lib/pagination"
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
	storage Storage
}

func NewListAuditEvents(storage Storage) *ListAuditEvents {
	return &ListAuditEvents{storage: storage}
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

	if params.Since != nil {
		eventFilter.CreatedAt = filter.GreaterOrEquals(*params.Since)
	}

	if params.Until != nil && params.Since == nil {
		eventFilter.CreatedAt = filter.Less(*params.Until)
	}

	limit := params.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	page := params.Offset/limit + 1

	events, err := uc.storage.AuditEvents().Find(ctx, eventFilter,
		dbutil.WithOrder(AuditEventFieldCreatedAt, dbutil.OrderDirDesc),
		dbutil.WithPagination(&pagination.Pagination{Page: page, PerPage: limit}),
	)
	if err != nil {
		return nil, fmt.Errorf("listing audit events: %w", err)
	}

	return events, nil
}
