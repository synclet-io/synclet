package pipelinestorage

import (
	"context"

	"github.com/google/uuid"

	pipelinesvc "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// AppendLog inserts a single log line for a job.
func (s *JobLogsStorage) AppendLog(ctx context.Context, jobID uuid.UUID, line string) error {
	return s.DB.WithContext(ctx).Exec(
		`INSERT INTO pipeline.job_logs (job_id, log_line) VALUES (?, ?)`,
		jobID, line,
	).Error
}

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

// BatchAppendLogs inserts multiple log lines for a job in a single batch.
func (s *JobLogsStorage) BatchAppendLogs(ctx context.Context, jobID uuid.UUID, lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	logs := make([]dbJobLog, len(lines))
	for i, line := range lines {
		logs[i] = dbJobLog{JobID: jobID, LogLine: line}
	}
	return s.DB.WithContext(ctx).Create(&logs).Error
}
