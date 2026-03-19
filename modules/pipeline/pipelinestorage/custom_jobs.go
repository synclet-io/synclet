package pipelinestorage

import (
	"context"
	"fmt"
	"time"

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
