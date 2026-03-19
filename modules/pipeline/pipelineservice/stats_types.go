package pipelineservice

import (
	"time"

	"github.com/google/uuid"
)

// TimeRange represents a time range for stats queries.
type TimeRange string

const (
	TimeRange24h TimeRange = "24h"
	TimeRange7d  TimeRange = "7d"
	TimeRange30d TimeRange = "30d"
)

// Duration returns the time.Duration corresponding to the TimeRange.
func (tr TimeRange) Duration() time.Duration {
	switch tr {
	case TimeRange7d:
		return 7 * 24 * time.Hour
	case TimeRange30d:
		return 30 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

// GetConnectionStatsParams holds parameters for the connection stats query.
type GetConnectionStatsParams struct {
	ConnectionID uuid.UUID
	WorkspaceID  uuid.UUID
	TimeRange    TimeRange
}

// GetWorkspaceStatsParams holds parameters for the workspace stats query.
type GetWorkspaceStatsParams struct {
	WorkspaceID uuid.UUID
	TimeRange   TimeRange
}

// GetSyncTimelineParams holds parameters for the sync timeline query.
type GetSyncTimelineParams struct {
	WorkspaceID  uuid.UUID
	TimeRange    TimeRange
	ConnectionID *uuid.UUID // optional, nil for workspace-level
}

// ConnectionStatsResult holds the computed per-connection statistics.
type ConnectionStatsResult struct {
	AvgDurationMs int64
	SuccessRate   float64
	TotalRecords  int64
	LastSyncAt    *time.Time
	Health        Health

	AvgDurationDelta  float64
	SuccessRateDelta  float64
	TotalRecordsDelta float64

	DurationChart    []DurationChartPoint
	RecordsChart     []RecordsChartPoint
	FailureBreakdown []FailureCategoryItem
}

// DurationChartPoint represents a single job's duration for the duration chart.
type DurationChartPoint struct {
	Label      string
	DurationMs int64
	Status     SyncStatus
}

// RecordsChartPoint represents time-bucketed records data.
type RecordsChartPoint struct {
	Label       string
	RecordsRead int64
}

// WorkspaceStatsResult holds the computed workspace-level statistics.
type WorkspaceStatsResult struct {
	TotalSyncs        int64
	SuccessRate       float64
	RecordsSynced     int64
	ActiveConnections int32
	FailedSyncs       int64

	TotalSyncsDelta    float64
	SuccessRateDelta   float64
	RecordsSyncedDelta float64
	FailedSyncsDelta   float64

	ConnectionHealths []ConnectionHealthItem
	TopConnections    []TopConnectionItem
	FailureBreakdown  []FailureCategoryItem
}

// ConnectionHealthItem represents a single connection's health status.
type ConnectionHealthItem struct {
	ConnectionID   uuid.UUID
	ConnectionName string
	Health         Health
	LastSyncAt     *time.Time
}

// TopConnectionItem represents a top connection by volume.
type TopConnectionItem struct {
	ConnectionID    uuid.UUID
	ConnectionName  string
	RecordsSynced   int64
	BytesSynced     int64
	LastSyncAt      *time.Time
	SparklineValues []int64
}

// FailureCategoryItem represents a failure category with its count.
type FailureCategoryItem struct {
	Category FailureCategory
	Count    int32
}

// SyncTimelineResult holds time-bucketed sync data for charts.
type SyncTimelineResult struct {
	Points     []TimelinePointItem
	Throughput []ThroughputPointItem
	Durations  []DurationPointItem
}

// TimelinePointItem represents succeeded/failed counts per time bucket.
type TimelinePointItem struct {
	Label     string
	Succeeded int32
	Failed    int32
}

// ThroughputPointItem represents records/bytes per time bucket.
type ThroughputPointItem struct {
	Label       string
	RecordsRead int64
	BytesSynced int64
}

// DurationPointItem represents average duration per time bucket.
type DurationPointItem struct {
	Label         string
	AvgDurationMs int64
}

// ConnectionRollup holds aggregated rollup data for a connection over a time range.
type ConnectionRollup struct {
	SyncsTotal      int64
	SyncsSucceeded  int64
	SyncsFailed     int64
	RecordsRead     int64
	TotalDurationMs int64
}

// RollupTotals holds workspace-level rollup totals.
type RollupTotals struct {
	SyncsTotal     int64
	SyncsSucceeded int64
	SyncsFailed    int64
	RecordsRead    int64
}

// LastJobInfo holds status and completion time of a job.
type LastJobInfo struct {
	Status      string
	CompletedAt *time.Time
}

// TopConnectionRow holds raw top connection data from storage.
type TopConnectionRow struct {
	ConnectionID  uuid.UUID
	RecordsSynced int64
	BytesSynced   int64
}

// FailedJobRow holds error info from a failed job.
type FailedJobRow struct {
	Error         *string
	FailureReason *string
}

// TimelineRow holds a single time-bucketed row from the timeline query.
type TimelineRow struct {
	BucketStart    time.Time
	SyncsSucceeded int32
	SyncsFailed    int32
	RecordsRead    int64
	BytesSynced    int64
	AvgDurationMs  int64
}

// DurationChartRow holds raw duration chart data from storage.
type DurationChartRow struct {
	CompletedAt *time.Time
	Status      string
	DurationMs  int64
}

// RecordsChartRow holds raw records chart data from storage.
type RecordsChartRow struct {
	BucketStart time.Time
	RecordsRead int64
}
