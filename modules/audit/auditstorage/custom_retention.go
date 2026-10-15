package auditstorage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/synclet-io/synclet/modules/audit/auditservice"
)

// RetentionStorage runs the bulk queries behind audit retention cleanup.
type RetentionStorage struct {
	db *gorm.DB
}

func NewRetentionStorage(db *gorm.DB) *RetentionStorage {
	return &RetentionStorage{db: db}
}

var _ auditservice.RetentionStorage = (*RetentionStorage)(nil)

func (s *RetentionStorage) DistinctWorkspaceIDs(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID

	err := s.db.WithContext(ctx).Model(&dbAuditEvent{}).Distinct().Pluck("workspace_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("listing audit event workspace ids: %w", err)
	}

	return ids, nil
}

func (s *RetentionStorage) DeleteOlderThan(ctx context.Context, workspaceID uuid.UUID, cutoff time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Exec(
		"DELETE FROM audit.audit_events WHERE workspace_id = ? AND created_at < ?",
		workspaceID, cutoff,
	)
	if result.Error != nil {
		return 0, fmt.Errorf("deleting expired audit events: %w", result.Error)
	}

	return result.RowsAffected, nil
}
