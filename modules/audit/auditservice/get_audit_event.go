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
}

func NewGetAuditEvent(storage Storage) *GetAuditEvent {
	return &GetAuditEvent{storage: storage}
}

func (uc *GetAuditEvent) Execute(ctx context.Context, params GetAuditEventParams) (*AuditEvent, error) {
	event, err := uc.storage.AuditEvents().First(ctx, &AuditEventFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting audit event: %w", err)
	}

	return event, nil
}
