package pipelinestorage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JobRetentionStorageImpl provides bulk deletion for job retention cleanup.
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

// DeleteJobs deletes the given jobs. CASCADE removes their attempts, logs and
// log segments.
func (s *JobRetentionStorageImpl) DeleteJobs(ctx context.Context, ids []uuid.UUID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	result := s.DB.WithContext(ctx).Exec(`DELETE FROM pipeline.jobs WHERE id IN ?`, ids)
	if result.Error != nil {
		return 0, fmt.Errorf("deleting jobs: %w", result.Error)
	}

	return result.RowsAffected, nil
}
