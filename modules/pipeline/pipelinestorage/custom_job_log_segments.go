package pipelinestorage

import (
	"context"
	"database/sql"
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

// LockArchiveKeys locks the given jobs and their log segments until the
// surrounding transaction ends and returns the archive key of every segment:
// the stored key of an archived segment and the key an archiver writes for an
// open one, whose object may exist when marking it failed. While the locks are
// held no segment of these jobs can be created and no object written for them,
// so the keys stay complete until the jobs are deleted.
func (s *JobLogSegmentsStorage) LockArchiveKeys(ctx context.Context, jobIDs []uuid.UUID) ([]string, error) {
	if len(jobIDs) == 0 {
		return nil, nil
	}

	if err := s.DB.WithContext(ctx).Exec(`
		SELECT id FROM pipeline.jobs WHERE id IN ? ORDER BY id FOR UPDATE`, jobIDs,
	).Error; err != nil {
		return nil, fmt.Errorf("locking jobs: %w", err)
	}

	return s.lockedKeys(ctx, `j.id IN ?`, jobIDs)
}

// LockWorkspaceArchiveKeys is LockArchiveKeys for every job of the workspace.
func (s *JobLogSegmentsStorage) LockWorkspaceArchiveKeys(ctx context.Context, workspaceID uuid.UUID) ([]string, error) {
	if err := s.DB.WithContext(ctx).Exec(`
		SELECT j.id FROM pipeline.jobs j
		JOIN pipeline.connections c ON c.id = j.connection_id
		WHERE c.workspace_id = ?
		ORDER BY j.id
		FOR UPDATE OF j`, workspaceID,
	).Error; err != nil {
		return nil, fmt.Errorf("locking workspace jobs: %w", err)
	}

	return s.lockedKeys(ctx, `c.workspace_id = ?`, workspaceID)
}

type lockedSegment struct {
	WorkspaceID  uuid.UUID
	ConnectionID uuid.UUID
	JobID        uuid.UUID
	Attempt      int
	ArchiveKey   sql.NullString
}

// lockedKeys locks the segments of the jobs matching where and returns their
// keys. The query does not filter on archive columns: a row being archived
// concurrently is filtered on its old version, and only rows that match are
// waited for and returned in their committed state.
func (s *JobLogSegmentsStorage) lockedKeys(ctx context.Context, where string, args ...any) ([]string, error) {
	var rows []lockedSegment

	if err := s.DB.WithContext(ctx).Raw(`
		SELECT c.workspace_id, j.connection_id, s.job_id, s.attempt, s.archive_key
		FROM pipeline.job_log_segments s
		JOIN pipeline.jobs j ON j.id = s.job_id
		JOIN pipeline.connections c ON c.id = j.connection_id
		WHERE `+where+`
		ORDER BY s.job_id, s.attempt
		FOR UPDATE OF s`, args...,
	).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("locking job log segments: %w", err)
	}

	keys := make([]string, len(rows))

	for i, row := range rows {
		keys[i] = pipelinesvc.LogArchiveKey(row.WorkspaceID, row.ConnectionID, row.JobID, row.Attempt)
		if row.ArchiveKey.Valid {
			keys[i] = row.ArchiveKey.String
		}
	}

	return keys, nil
}
