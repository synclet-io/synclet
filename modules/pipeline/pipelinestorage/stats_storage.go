package pipelinestorage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	pipelinesvc "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// StatsStorage implements pipelinesvc.StatsStorage with raw SQL queries.
type StatsStorage struct {
	db *gorm.DB
}

// NewStatsStorage creates a new StatsStorage.
func NewStatsStorage(db *gorm.DB) *StatsStorage {
	return &StatsStorage{db: db}
}

var _ pipelinesvc.StatsStorage = &StatsStorage{}

// UpsertRollups computes and upserts rollup buckets from job data.
func (s *StatsStorage) UpsertRollups(ctx context.Context, bucketSize pipelinesvc.BucketSize, since time.Time, truncUnit string) error {
	// Allowlist to prevent SQL injection — truncUnit is interpolated into SQL
	// because date_trunc() doesn't accept parameterized unit arguments.
	switch truncUnit {
	case "hour", "day", "week", "month":
		// valid
	default:
		return fmt.Errorf("invalid truncUnit: %q", truncUnit)
	}

	query := fmt.Sprintf(`
		INSERT INTO pipeline.stats_rollups (
			workspace_id, connection_id, bucket_start, bucket_size,
			syncs_total, syncs_succeeded, syncs_failed,
			records_read, bytes_synced,
			total_duration_ms, avg_duration_ms
		)
		SELECT
			c.workspace_id,
			j.connection_id,
			date_trunc('%s', j.completed_at) AS bucket_start,
			$1 AS bucket_size,
			COUNT(*) AS syncs_total,
			COUNT(*) FILTER (WHERE j.status = 'completed') AS syncs_succeeded,
			COUNT(*) FILTER (WHERE j.status = 'failed') AS syncs_failed,
			COALESCE(SUM((ja.sync_stats_json->>'records_read')::bigint), 0) AS records_read,
			COALESCE(SUM((ja.sync_stats_json->>'bytes_synced')::bigint), 0) AS bytes_synced,
			COALESCE(SUM((ja.sync_stats_json->>'duration')::bigint / 1000000), 0) AS total_duration_ms,
			CASE
				WHEN COUNT(*) > 0 THEN COALESCE(SUM((ja.sync_stats_json->>'duration')::bigint / 1000000), 0) / COUNT(*)
				ELSE 0
			END AS avg_duration_ms
		FROM pipeline.jobs j
		JOIN pipeline.connections c ON c.id = j.connection_id
		LEFT JOIN pipeline.job_attempts ja ON ja.job_id = j.id AND ja.attempt_number = j.attempt
		WHERE j.status IN ('completed', 'failed')
			AND j.completed_at >= $2
			AND j.completed_at IS NOT NULL
		GROUP BY c.workspace_id, j.connection_id, date_trunc('%s', j.completed_at)
		ON CONFLICT (connection_id, bucket_start, bucket_size) DO UPDATE SET
			syncs_total = EXCLUDED.syncs_total,
			syncs_succeeded = EXCLUDED.syncs_succeeded,
			syncs_failed = EXCLUDED.syncs_failed,
			records_read = EXCLUDED.records_read,
			bytes_synced = EXCLUDED.bytes_synced,
			total_duration_ms = EXCLUDED.total_duration_ms,
			avg_duration_ms = EXCLUDED.avg_duration_ms
	`, truncUnit, truncUnit)

	dbBucketSize, err := convertBucketSizeToDB(bucketSize)
	if err != nil {
		return err
	}

	result := s.db.WithContext(ctx).Exec(query, dbBucketSize, since)

	return result.Error
}

// QueryConnectionRollup returns aggregated rollup data for a connection over a time range.
func (s *StatsStorage) QueryConnectionRollup(ctx context.Context, connectionID uuid.UUID, from, to time.Time) (pipelinesvc.ConnectionRollup, error) {
	var stats pipelinesvc.ConnectionRollup

	row := s.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(syncs_total), 0),
			COALESCE(SUM(syncs_succeeded), 0),
			COALESCE(SUM(syncs_failed), 0),
			COALESCE(SUM(records_read), 0),
			COALESCE(SUM(total_duration_ms), 0)
		FROM pipeline.stats_rollups
		WHERE connection_id = ?
			AND bucket_start >= ?
			AND bucket_start < ?
	`, connectionID, from, to).Row()

	if err := row.Scan(&stats.SyncsTotal, &stats.SyncsSucceeded, &stats.SyncsFailed, &stats.RecordsRead, &stats.TotalDurationMs); err != nil {
		return stats, fmt.Errorf("scanning connection rollup: %w", err)
	}

	return stats, nil
}

// QueryLastSyncAt returns the last completed_at time for a connection.
func (s *StatsStorage) QueryLastSyncAt(ctx context.Context, connectionID uuid.UUID) (*time.Time, error) {
	var lastSyncAt *time.Time

	if err := s.db.WithContext(ctx).Raw(`
		SELECT completed_at FROM pipeline.jobs
		WHERE connection_id = ? AND completed_at IS NOT NULL
		ORDER BY completed_at DESC LIMIT 1
	`, connectionID).Scan(&lastSyncAt).Error; err != nil {
		return nil, fmt.Errorf("querying last sync at: %w", err)
	}

	return lastSyncAt, nil
}

// QueryLastJobInfo returns the status and completed_at of the most recent job for a connection.
func (s *StatsStorage) QueryLastJobInfo(ctx context.Context, connectionID uuid.UUID) (*pipelinesvc.LastJobInfo, error) {
	var row struct {
		Status      string
		CompletedAt *time.Time
	}

	if err := s.db.WithContext(ctx).Raw(`
		SELECT status, completed_at
		FROM pipeline.jobs
		WHERE connection_id = ?
		ORDER BY completed_at DESC NULLS LAST
		LIMIT 1
	`, connectionID).Scan(&row).Error; err != nil {
		return nil, fmt.Errorf("querying last job info: %w", err)
	}

	if row.Status == "" {
		return nil, nil
	}

	return &pipelinesvc.LastJobInfo{
		Status:      row.Status,
		CompletedAt: row.CompletedAt,
	}, nil
}

// QueryDurationChart returns raw duration chart rows for a connection's recent jobs.
func (s *StatsStorage) QueryDurationChart(ctx context.Context, connectionID uuid.UUID, from, to time.Time, limit int) ([]pipelinesvc.DurationChartRow, error) {
	var rows []pipelinesvc.DurationChartRow

	if err := s.db.WithContext(ctx).Raw(`
		SELECT j.completed_at, j.status,
			COALESCE((ja.sync_stats_json->>'duration')::bigint / 1000000, 0) AS duration_ms
		FROM pipeline.jobs j
		LEFT JOIN pipeline.job_attempts ja ON ja.job_id = j.id AND ja.attempt_number = j.attempt
		WHERE j.connection_id = ?
			AND j.completed_at >= ?
			AND j.completed_at < ?
			AND j.status IN ('completed', 'failed')
		ORDER BY j.completed_at DESC
		LIMIT ?
	`, connectionID, from, to, limit).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("querying duration chart: %w", err)
	}

	return rows, nil
}

// QueryRecordsChart returns raw records chart rows from rollups for a connection.
func (s *StatsStorage) QueryRecordsChart(ctx context.Context, connectionID uuid.UUID, from, to time.Time) ([]pipelinesvc.RecordsChartRow, error) {
	var rows []pipelinesvc.RecordsChartRow

	if err := s.db.WithContext(ctx).Raw(`
		SELECT bucket_start, records_read
		FROM pipeline.stats_rollups
		WHERE connection_id = ?
			AND bucket_start >= ?
			AND bucket_start < ?
		ORDER BY bucket_start ASC
	`, connectionID, from, to).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("querying records chart: %w", err)
	}

	return rows, nil
}

// QueryConnectionFailedJobs returns failed job error info for a connection in a time range.
func (s *StatsStorage) QueryConnectionFailedJobs(ctx context.Context, connectionID uuid.UUID, from, to time.Time) ([]pipelinesvc.FailedJobRow, error) {
	var rows []pipelinesvc.FailedJobRow

	if err := s.db.WithContext(ctx).Raw(`
		SELECT error, failure_reason
		FROM pipeline.jobs
		WHERE connection_id = ?
			AND status = 'failed'
			AND completed_at >= ?
			AND completed_at < ?
	`, connectionID, from, to).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("querying connection failed jobs: %w", err)
	}

	return rows, nil
}

// QueryWorkspaceRollup returns aggregated rollup totals for a workspace over a time range.
func (s *StatsStorage) QueryWorkspaceRollup(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) (pipelinesvc.RollupTotals, error) {
	var totals pipelinesvc.RollupTotals

	row := s.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(syncs_total), 0),
			COALESCE(SUM(syncs_succeeded), 0),
			COALESCE(SUM(syncs_failed), 0),
			COALESCE(SUM(records_read), 0)
		FROM pipeline.stats_rollups
		WHERE workspace_id = ?
			AND bucket_start >= ?
			AND bucket_start < ?
	`, workspaceID, from, to).Row()

	if err := row.Scan(&totals.SyncsTotal, &totals.SyncsSucceeded, &totals.SyncsFailed, &totals.RecordsRead); err != nil {
		return totals, fmt.Errorf("scanning workspace rollup: %w", err)
	}

	return totals, nil
}

// QueryTopConnections returns top connections by records synced for a workspace.
func (s *StatsStorage) QueryTopConnections(ctx context.Context, workspaceID uuid.UUID, from, to time.Time, limit int) ([]pipelinesvc.TopConnectionRow, error) {
	var rows []pipelinesvc.TopConnectionRow

	if err := s.db.WithContext(ctx).Raw(`
		SELECT
			connection_id,
			SUM(records_read) AS records_synced,
			SUM(bytes_synced) AS bytes_synced
		FROM pipeline.stats_rollups
		WHERE workspace_id = ?
			AND bucket_start >= ?
			AND bucket_start < ?
		GROUP BY connection_id
		ORDER BY records_synced DESC
		LIMIT ?
	`, workspaceID, from, to, limit).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("querying top connections: %w", err)
	}

	return rows, nil
}

// QueryConnectionLastCompletedAt returns the last completed job time for a connection.
func (s *StatsStorage) QueryConnectionLastCompletedAt(ctx context.Context, connectionID uuid.UUID) (*time.Time, error) {
	var lastCompletedAt *time.Time

	if err := s.db.WithContext(ctx).Raw(`
		SELECT completed_at FROM pipeline.jobs
		WHERE connection_id = ? AND status = 'completed'
		ORDER BY completed_at DESC LIMIT 1
	`, connectionID).Scan(&lastCompletedAt).Error; err != nil {
		return nil, fmt.Errorf("querying connection last completed at: %w", err)
	}

	return lastCompletedAt, nil
}

// QueryConnectionSparkline returns recent rollup records_read values for a connection.
func (s *StatsStorage) QueryConnectionSparkline(ctx context.Context, connectionID uuid.UUID, from, to time.Time, limit int) ([]int64, error) {
	var sparkline []int64

	if err := s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(records_read, 0) FROM pipeline.stats_rollups
		WHERE connection_id = ?
			AND bucket_start >= ?
			AND bucket_start < ?
		ORDER BY bucket_start DESC
		LIMIT ?
	`, connectionID, from, to, limit).Scan(&sparkline).Error; err != nil {
		return nil, fmt.Errorf("querying connection sparkline: %w", err)
	}

	return sparkline, nil
}

// QueryWorkspaceFailedJobs returns failed job error info for a workspace in a time range.
func (s *StatsStorage) QueryWorkspaceFailedJobs(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]pipelinesvc.FailedJobRow, error) {
	var rows []pipelinesvc.FailedJobRow

	if err := s.db.WithContext(ctx).Raw(`
		SELECT j.error, j.failure_reason
		FROM pipeline.jobs j
		JOIN pipeline.connections c ON c.id = j.connection_id
		WHERE c.workspace_id = ?
			AND j.status = 'failed'
			AND j.completed_at >= ?
			AND j.completed_at < ?
	`, workspaceID, from, to).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("querying workspace failed jobs: %w", err)
	}

	return rows, nil
}

// QueryLastJobInfoBatch returns the most recent job info for each of the given connection IDs
// in a single query, eliminating N+1 queries in workspace stats.
func (s *StatsStorage) QueryLastJobInfoBatch(ctx context.Context, connectionIDs []uuid.UUID) (map[uuid.UUID]*pipelinesvc.LastJobInfo, error) {
	if len(connectionIDs) == 0 {
		return make(map[uuid.UUID]*pipelinesvc.LastJobInfo), nil
	}

	var rows []struct {
		ConnectionID uuid.UUID
		Status       string
		CompletedAt  *time.Time
	}

	if err := s.db.WithContext(ctx).Raw(`
		SELECT DISTINCT ON (connection_id)
			connection_id, status, completed_at
		FROM pipeline.jobs
		WHERE connection_id IN ?
		ORDER BY connection_id, completed_at DESC NULLS LAST
	`, connectionIDs).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("querying last job info batch: %w", err)
	}

	result := make(map[uuid.UUID]*pipelinesvc.LastJobInfo, len(rows))
	for _, r := range rows {
		result[r.ConnectionID] = &pipelinesvc.LastJobInfo{
			Status:      r.Status,
			CompletedAt: r.CompletedAt,
		}
	}

	return result, nil
}

// QueryConnectionLastCompletedAtBatch returns the last completed job time for each of the given
// connection IDs in a single query.
func (s *StatsStorage) QueryConnectionLastCompletedAtBatch(ctx context.Context, connectionIDs []uuid.UUID) (map[uuid.UUID]*time.Time, error) {
	if len(connectionIDs) == 0 {
		return make(map[uuid.UUID]*time.Time), nil
	}

	var rows []struct {
		ConnectionID uuid.UUID
		CompletedAt  time.Time
	}

	if err := s.db.WithContext(ctx).Raw(`
		SELECT DISTINCT ON (connection_id)
			connection_id, completed_at
		FROM pipeline.jobs
		WHERE connection_id IN ?
			AND status = 'completed'
			AND completed_at IS NOT NULL
		ORDER BY connection_id, completed_at DESC
	`, connectionIDs).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("querying connection last completed at batch: %w", err)
	}

	result := make(map[uuid.UUID]*time.Time, len(rows))
	for _, r := range rows {
		t := r.CompletedAt
		result[r.ConnectionID] = &t
	}

	return result, nil
}

// QueryConnectionSparklineBatch returns recent rollup records_read values for each of the given
// connection IDs in a single query, with up to `limit` values per connection.
func (s *StatsStorage) QueryConnectionSparklineBatch(ctx context.Context, connectionIDs []uuid.UUID, from, to time.Time, limit int) (map[uuid.UUID][]int64, error) {
	if len(connectionIDs) == 0 {
		return make(map[uuid.UUID][]int64), nil
	}

	var rows []struct {
		ConnectionID uuid.UUID
		RecordsRead  int64
	}

	// Use a window function to rank rollup buckets per connection and limit to top N.
	if err := s.db.WithContext(ctx).Raw(`
		SELECT connection_id, records_read FROM (
			SELECT connection_id, COALESCE(records_read, 0) AS records_read,
				ROW_NUMBER() OVER (PARTITION BY connection_id ORDER BY bucket_start DESC) AS rn
			FROM pipeline.stats_rollups
			WHERE connection_id IN ?
				AND bucket_start >= ?
				AND bucket_start < ?
		) sub
		WHERE rn <= ?
		ORDER BY connection_id, rn ASC
	`, connectionIDs, from, to, limit).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("querying connection sparkline batch: %w", err)
	}

	result := make(map[uuid.UUID][]int64, len(connectionIDs))
	for _, r := range rows {
		result[r.ConnectionID] = append(result[r.ConnectionID], r.RecordsRead)
	}

	return result, nil
}

// QuerySyncTimeline returns time-bucketed sync data for timeline charts.
func (s *StatsStorage) QuerySyncTimeline(ctx context.Context, workspaceID uuid.UUID, bucketSize pipelinesvc.BucketSize, from, to time.Time, connectionID *uuid.UUID) ([]pipelinesvc.TimelineRow, error) {
	query := `
		SELECT
			bucket_start,
			COALESCE(SUM(syncs_succeeded), 0) AS syncs_succeeded,
			COALESCE(SUM(syncs_failed), 0) AS syncs_failed,
			COALESCE(SUM(records_read), 0) AS records_read,
			COALESCE(SUM(bytes_synced), 0) AS bytes_synced,
			CASE
				WHEN SUM(syncs_total) > 0 THEN (COALESCE(SUM(total_duration_ms), 0) / SUM(syncs_total))::bigint
				ELSE 0
			END AS avg_duration_ms
		FROM pipeline.stats_rollups
		WHERE workspace_id = ?
			AND bucket_size = ?
			AND bucket_start >= ?
			AND bucket_start < ?
	`

	dbBucketSize, err := convertBucketSizeToDB(bucketSize)
	if err != nil {
		return nil, err
	}

	args := []interface{}{workspaceID, dbBucketSize, from, to}

	if connectionID != nil {
		query += ` AND connection_id = ?`
		args = append(args, *connectionID)
	}

	query += ` GROUP BY bucket_start ORDER BY bucket_start ASC`

	var rows []pipelinesvc.TimelineRow
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("querying sync timeline: %w", err)
	}

	return rows, nil
}
