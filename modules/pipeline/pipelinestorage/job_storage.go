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

// DeleteOldestTerminalJobs deletes terminal jobs exceeding keepCount for a workspace.
// Uses subquery with OFFSET to keep the N most recent terminal jobs.
// CASCADE handles job_attempts and job_logs deletion automatically.
func (s *JobRetentionStorageImpl) DeleteOldestTerminalJobs(ctx context.Context, workspaceID uuid.UUID, keepCount int) (int64, error) {
	result := s.DB.WithContext(ctx).Exec(`
		DELETE FROM pipeline.jobs
		WHERE id IN (
			SELECT j.id FROM pipeline.jobs j
			JOIN pipeline.connections c ON c.id = j.connection_id
			WHERE c.workspace_id = ?
			  AND j.status IN (?, ?, ?)
			ORDER BY j.created_at DESC
			OFFSET ?
		)`, workspaceID, jobStatusCompleted, jobStatusFailed, jobStatusCancelled, keepCount)
	if result.Error != nil {
		return 0, fmt.Errorf("deleting old jobs: %w", result.Error)
	}

	return result.RowsAffected, nil
}
