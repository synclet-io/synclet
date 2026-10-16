package auditstorage

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/synclet-io/synclet/modules/audit/auditservice"
)

// EventPageStorage lists audit events with row-offset pagination.
type EventPageStorage struct {
	db *gorm.DB
}

func NewEventPageStorage(db *gorm.DB) *EventPageStorage {
	return &EventPageStorage{db: db}
}

var _ auditservice.EventPageStorage = (*EventPageStorage)(nil)

func (s *EventPageStorage) FindPage(ctx context.Context, flt *auditservice.AuditEventFilter, offset, limit int) ([]*auditservice.AuditEvent, error) {
	filterExpr, err := buildAuditEventFilterExpr(flt)
	if err != nil {
		return nil, fmt.Errorf("building audit event filter: %w", err)
	}

	query := s.db.WithContext(ctx).Model(&dbAuditEvent{}).
		Order(clause.OrderBy{Columns: []clause.OrderByColumn{
			{Column: clause.Column{Name: "created_at"}, Desc: true},
			{Column: clause.Column{Name: "id"}, Desc: true},
		}}).
		Offset(offset).
		Limit(limit)

	if filterExpr != nil {
		query = query.Clauses(filterExpr)
	}

	var rows []dbAuditEvent
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("listing audit events: %w", wrapAuditEventQueryError(err))
	}

	events := make([]*auditservice.AuditEvent, 0, len(rows))

	for i := range rows {
		event, err := convertAuditEventFromDB(&rows[i])
		if err != nil {
			return nil, fmt.Errorf("converting audit event: %w", err)
		}

		events = append(events, event)
	}

	return events, nil
}
