package pipelinestorage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JobRetentionStorageImpl selects jobs for job retention cleanup.
type JobRetentionStorageImpl struct {
	DB *gorm.DB
}

// NewJobRetentionStorage returns a JobRetentionStorageImpl for retention cleanup use.
func NewJobRetentionStorage(db *gorm.DB) *JobRetentionStorageImpl {
	return &JobRetentionStorageImpl{DB: db}
}

// ListExcessTerminalJobIDs returns up to limit terminal jobs of a workspace
// beyond the keepCount most recent ones, newest first.
func (s *JobRetentionStorageImpl) ListExcessTerminalJobIDs(ctx context.Context, workspaceID uuid.UUID, keepCount, limit int) ([]uuid.UUID, error) {
	var ids []uuid.UUID

	if err := s.DB.WithContext(ctx).Raw(`
		SELECT j.id FROM pipeline.jobs j
		JOIN pipeline.connections c ON c.id = j.connection_id
		WHERE c.workspace_id = ?
		  AND j.status IN (?, ?, ?)
		ORDER BY j.created_at DESC
		OFFSET ? LIMIT ?`,
		workspaceID, jobStatusCompleted, jobStatusFailed, jobStatusCancelled, keepCount, limit,
	).Scan(&ids).Error; err != nil {
		return nil, fmt.Errorf("listing excess terminal jobs: %w", err)
	}

	return ids, nil
}
