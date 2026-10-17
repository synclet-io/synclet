package pipelinestorage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	pipelinesvc "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ClaimNextScheduledJob atomically claims the next scheduled job for a worker.
// Uses SELECT FOR UPDATE SKIP LOCKED for safe concurrent claiming.
// Sets status to 'starting', assigns worker_id, started_at, and heartbeat_at.
// Returns nil, nil if no scheduled jobs are available.
func (s *JobsStorage) ClaimNextScheduledJob(ctx context.Context, workerID string) (*pipelinesvc.Job, error) {
	var row dbJob
	now := time.Now()

	result := s.DB.WithContext(ctx).
		Raw(`UPDATE pipeline.jobs SET status = ?, worker_id = ?, started_at = ?, heartbeat_at = ?
		     WHERE id = (
		       SELECT id FROM pipeline.jobs
		       WHERE status = ?
		       ORDER BY created_at ASC
		       LIMIT 1
		       FOR UPDATE SKIP LOCKED
		     )
		     RETURNING *`,
			jobStatusStarting, workerID, now, now,
			jobStatusScheduled).
		Scan(&row)
	if result.Error != nil {
		return nil, fmt.Errorf("claiming scheduled job: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, nil
	}

	return convertJobFromDB(&row)
}

// CountActiveJobs counts jobs with status in (scheduled, starting, running).
func (s *JobsStorage) CountActiveJobs(ctx context.Context) (int, error) {
	var count int

	result := s.DB.WithContext(ctx).
		Raw(`SELECT COUNT(*) FROM pipeline.jobs WHERE status IN (?, ?, ?)`,
			jobStatusScheduled, jobStatusStarting, jobStatusRunning).
		Scan(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("counting active jobs: %w", result.Error)
	}

	return count, nil
}

// UpdateIfStatus writes the full job row only while its stored status is one
// of fromStatuses. Returns false when the job is missing or in another status.
func (s *JobsStorage) UpdateIfStatus(ctx context.Context, job *pipelinesvc.Job, fromStatuses ...pipelinesvc.JobStatus) (bool, error) {
	row, err := convertJobToDB(job)
	if err != nil {
		return false, err
	}

	statuses := make([]string, 0, len(fromStatuses))
	for _, st := range fromStatuses {
		dbStatus, err := convertJobStatusToDB(st)
		if err != nil {
			return false, err
		}

		statuses = append(statuses, dbStatus)
	}

	result := s.DB.WithContext(ctx).
		Model(row).
		Where("status IN ?", statuses).
		Select("*").
		Updates(row)
	if result.Error != nil {
		return false, wrapJobQueryError(fmt.Errorf("conditionally updating job: %w", result.Error))
	}

	return result.RowsAffected > 0, nil
}

// TouchHeartbeat sets heartbeat_at on a starting or running job. Returns false
// when the job is missing or no longer active.
func (s *JobsStorage) TouchHeartbeat(ctx context.Context, id uuid.UUID, at time.Time) (bool, error) {
	result := s.DB.WithContext(ctx).
		Exec(`UPDATE pipeline.jobs SET heartbeat_at = ? WHERE id = ? AND status IN (?, ?)`,
			at, id, jobStatusStarting, jobStatusRunning)
	if result.Error != nil {
		return false, fmt.Errorf("updating job heartbeat: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

// MarkRunning sets a starting job to running. Returns false when the job is
// missing or no longer starting.
func (s *JobsStorage) MarkRunning(ctx context.Context, id uuid.UUID) (bool, error) {
	result := s.DB.WithContext(ctx).
		Exec(`UPDATE pipeline.jobs SET status = ? WHERE id = ? AND status = ?`,
			jobStatusRunning, id, jobStatusStarting)
	if result.Error != nil {
		return false, wrapJobQueryError(fmt.Errorf("marking job running: %w", result.Error))
	}

	return result.RowsAffected > 0, nil
}

// LockKeyShare holds a key share lock on the job until the surrounding
// transaction ends, so the job cannot be deleted meanwhile; status and other
// non-key updates are not blocked. Returns false when the job does not exist.
func (s *JobsStorage) LockKeyShare(ctx context.Context, id uuid.UUID) (bool, error) {
	var ids []uuid.UUID

	if err := s.DB.WithContext(ctx).
		Raw(`SELECT id FROM pipeline.jobs WHERE id = ? FOR KEY SHARE`, id).
		Scan(&ids).Error; err != nil {
		return false, wrapJobQueryError(fmt.Errorf("locking job: %w", err))
	}

	return len(ids) > 0, nil
}
