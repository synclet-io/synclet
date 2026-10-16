package pipelinestorage

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pipelinesvc "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// LockCurrent returns the segment of the job's current attempt, creating it
// when missing, and holds its row lock until the surrounding transaction ends.
// Returns ErrJobNotFound when the job does not exist.
func (s *JobLogSegmentsStorage) LockCurrent(ctx context.Context, jobID uuid.UUID) (*pipelinesvc.JobLogSegment, error) {
	var rows []dbJobLogSegment

	if err := s.DB.WithContext(ctx).Raw(`
		INSERT INTO pipeline.job_log_segments (job_id, attempt)
		SELECT id, attempt FROM pipeline.jobs WHERE id = ?
		ON CONFLICT (job_id, attempt) DO UPDATE SET job_id = EXCLUDED.job_id
		RETURNING *`, jobID,
	).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("locking current job log segment: %w", err)
	}

	if len(rows) == 0 {
		return nil, pipelinesvc.ErrJobNotFound
	}

	return convertJobLogSegmentFromDB(&rows[0])
}

// Lock returns the segment of one attempt, creating it when missing, and holds
// its row lock until the surrounding transaction ends.
func (s *JobLogSegmentsStorage) Lock(ctx context.Context, jobID uuid.UUID, attempt int) (*pipelinesvc.JobLogSegment, error) {
	var row dbJobLogSegment

	if err := s.DB.WithContext(ctx).Raw(`
		INSERT INTO pipeline.job_log_segments (job_id, attempt)
		VALUES (?, ?)
		ON CONFLICT (job_id, attempt) DO UPDATE SET job_id = EXCLUDED.job_id
		RETURNING *`, jobID, attempt,
	).Scan(&row).Error; err != nil {
		return nil, fmt.Errorf("locking job log segment: %w", err)
	}

	return convertJobLogSegmentFromDB(&row)
}

// MarkArchived stores the archive fields of an open segment and sets it
// archived. Returns false when the segment is missing or already archived.
func (s *JobLogSegmentsStorage) MarkArchived(ctx context.Context, segment *pipelinesvc.JobLogSegment) (bool, error) {
	result := s.DB.WithContext(ctx).Exec(`
		UPDATE pipeline.job_log_segments
		SET status = ?, archive_key = ?, size_bytes = ?, first_log_id = ?, last_log_id = ?, archived_at = ?
		WHERE job_id = ? AND attempt = ? AND status = ?`,
		jobLogSegmentStatusArchived, segment.ArchiveKey, segment.SizeBytes, segment.FirstLogID, segment.LastLogID, segment.ArchivedAt,
		segment.JobID, segment.Attempt, jobLogSegmentStatusOpen,
	)
	if result.Error != nil {
		return false, fmt.Errorf("marking job log segment archived: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

// ListWorkspaceArchiveKeys returns the archive keys of every archived segment
// of the workspace's jobs.
func (s *JobLogSegmentsStorage) ListWorkspaceArchiveKeys(ctx context.Context, workspaceID uuid.UUID) ([]string, error) {
	var keys []string

	if err := s.DB.WithContext(ctx).Raw(`
		SELECT s.archive_key
		FROM pipeline.job_log_segments s
		JOIN pipeline.jobs j ON j.id = s.job_id
		JOIN pipeline.connections c ON c.id = j.connection_id
		WHERE c.workspace_id = ? AND s.archive_key IS NOT NULL`, workspaceID,
	).Scan(&keys).Error; err != nil {
		return nil, fmt.Errorf("listing workspace log archive keys: %w", err)
	}

	return keys, nil
}
