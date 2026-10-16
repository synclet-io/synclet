package pipelinestorage

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pipelinesvc "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetLogs returns log lines for a job with cursor-based pagination.
func (s *JobLogsStorage) GetLogs(ctx context.Context, jobID uuid.UUID, afterID int64, limit int) ([]pipelinesvc.JobLog, error) {
	var rows []dbJobLog

	query := s.DB.WithContext(ctx).
		Where("job_id = ? AND id > ?", jobID, afterID).
		Order("id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	return convertJobLogsFromDB(rows)
}

// InsertLines inserts log lines for one attempt of a job in a single batch.
func (s *JobLogsStorage) InsertLines(ctx context.Context, jobID uuid.UUID, attempt int, lines []string) error {
	if len(lines) == 0 {
		return nil
	}

	logs := make([]dbJobLog, len(lines))
	for i, line := range lines {
		logs[i] = dbJobLog{JobID: jobID, Attempt: attempt, LogLine: line}
	}

	return s.DB.WithContext(ctx).Omit("ID", "CreatedAt").Create(&logs).Error
}

// InsertLine inserts one log line and returns its id.
func (s *JobLogsStorage) InsertLine(ctx context.Context, jobID uuid.UUID, attempt int, line string) (int64, error) {
	var id int64

	if err := s.DB.WithContext(ctx).Raw(
		`INSERT INTO pipeline.job_logs (job_id, attempt, log_line) VALUES (?, ?, ?) RETURNING id`,
		jobID, attempt, line,
	).Scan(&id).Error; err != nil {
		return 0, fmt.Errorf("inserting job log line: %w", err)
	}

	return id, nil
}

// UpdateLine replaces the text of one log line. Returns false when the line is gone.
func (s *JobLogsStorage) UpdateLine(ctx context.Context, id int64, line string) (bool, error) {
	result := s.DB.WithContext(ctx).Exec(`UPDATE pipeline.job_logs SET log_line = ? WHERE id = ?`, line, id)
	if result.Error != nil {
		return false, fmt.Errorf("updating job log line: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

// MaxLogID returns the highest log id stored for one attempt of a job, or 0
// when the attempt has no lines.
func (s *JobLogsStorage) MaxLogID(ctx context.Context, jobID uuid.UUID, attempt int) (int64, error) {
	var id int64

	if err := s.DB.WithContext(ctx).Raw(
		`SELECT COALESCE(MAX(id), 0) FROM pipeline.job_logs WHERE job_id = ? AND attempt = ?`,
		jobID, attempt,
	).Scan(&id).Error; err != nil {
		return 0, fmt.Errorf("reading max job log id: %w", err)
	}

	return id, nil
}

// ListAttemptLogs returns the lines of one attempt with afterID < id <= upToID, in id order.
func (s *JobLogsStorage) ListAttemptLogs(ctx context.Context, jobID uuid.UUID, attempt int, afterID, upToID int64, limit int) ([]pipelinesvc.JobLog, error) {
	var rows []dbJobLog

	query := s.DB.WithContext(ctx).
		Where("job_id = ? AND attempt = ? AND id > ? AND id <= ?", jobID, attempt, afterID, upToID).
		Order("id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("listing attempt logs: %w", err)
	}

	return convertJobLogsFromDB(rows)
}

// DeleteAttemptLogs deletes the lines of one attempt with id <= upToID.
func (s *JobLogsStorage) DeleteAttemptLogs(ctx context.Context, jobID uuid.UUID, attempt int, upToID int64) error {
	if err := s.DB.WithContext(ctx).Exec(
		`DELETE FROM pipeline.job_logs WHERE job_id = ? AND attempt = ? AND id <= ?`,
		jobID, attempt, upToID,
	).Error; err != nil {
		return fmt.Errorf("deleting attempt logs: %w", err)
	}

	return nil
}

// ListUnarchivedAttempts returns, in ascending order, the attempts of a job
// that have lines in job_logs and no archived segment.
func (s *JobLogsStorage) ListUnarchivedAttempts(ctx context.Context, jobID uuid.UUID) ([]int, error) {
	var attempts []int

	if err := s.DB.WithContext(ctx).Raw(`
		SELECT DISTINCT l.attempt
		FROM pipeline.job_logs l
		LEFT JOIN pipeline.job_log_segments s ON s.job_id = l.job_id AND s.attempt = l.attempt
		WHERE l.job_id = ? AND (s.status IS NULL OR s.status <> ?)
		ORDER BY l.attempt`,
		jobID, jobLogSegmentStatusArchived,
	).Scan(&attempts).Error; err != nil {
		return nil, fmt.Errorf("listing unarchived attempts: %w", err)
	}

	return attempts, nil
}

func convertJobLogsFromDB(rows []dbJobLog) ([]pipelinesvc.JobLog, error) {
	result := make([]pipelinesvc.JobLog, len(rows))

	for i := range rows {
		converted, err := convertJobLogFromDB(&rows[i])
		if err != nil {
			return nil, err
		}

		result[i] = *converted
	}

	return result, nil
}
