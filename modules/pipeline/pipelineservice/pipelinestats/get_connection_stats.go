package pipelinestats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// parseSyncStatus converts a raw DB status string to a SyncStatus enum.
func parseSyncStatus(s string) pipelineservice.SyncStatus {
	switch strings.ToLower(s) {
	case "completed":
		return pipelineservice.SyncStatusCompleted
	case "failed":
		return pipelineservice.SyncStatusFailed
	default:
		return pipelineservice.SyncStatusCompleted
	}
}

// GetConnectionStats retrieves per-connection stats from rollups and job data.
type GetConnectionStats struct {
	statsStorage pipelineservice.StatsStorage
	storage      pipelineservice.Storage
}

// NewGetConnectionStats creates a new GetConnectionStats use case.
func NewGetConnectionStats(statsStorage pipelineservice.StatsStorage, storage pipelineservice.Storage) *GetConnectionStats {
	return &GetConnectionStats{statsStorage: statsStorage, storage: storage}
}

// Execute computes connection stats for the given time range.
func (uc *GetConnectionStats) Execute(ctx context.Context, params pipelineservice.GetConnectionStatsParams) (*pipelineservice.ConnectionStatsResult, error) {
	duration := params.TimeRange.Duration()
	now := time.Now()
	currentStart := now.Add(-duration)
	previousStart := currentStart.Add(-duration)

	result := &pipelineservice.ConnectionStatsResult{}

	// Verify connection belongs to workspace before fetching any stats.
	conn, err := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
		ID:          filter.Equals(params.ConnectionID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting connection: %w", err)
	}

	// Current period stats from rollups.
	currentStats, err := uc.statsStorage.QueryConnectionRollup(ctx, params.ConnectionID, currentStart, now)
	if err != nil {
		return nil, fmt.Errorf("querying current stats: %w", err)
	}

	result.TotalRecords = currentStats.RecordsRead
	if currentStats.SyncsTotal > 0 {
		result.AvgDurationMs = currentStats.TotalDurationMs / currentStats.SyncsTotal
		result.SuccessRate = float64(currentStats.SyncsSucceeded) / float64(currentStats.SyncsTotal) * 100
	}

	// Previous period for deltas.
	previousStats, err := uc.statsStorage.QueryConnectionRollup(ctx, params.ConnectionID, previousStart, currentStart)
	if err != nil {
		return nil, fmt.Errorf("querying previous stats: %w", err)
	}

	var prevAvgDuration int64
	if previousStats.SyncsTotal > 0 {
		prevAvgDuration = previousStats.TotalDurationMs / previousStats.SyncsTotal
	}
	result.AvgDurationDelta = computeDelta(float64(prevAvgDuration), float64(result.AvgDurationMs))

	var prevSuccessRate float64
	if previousStats.SyncsTotal > 0 {
		prevSuccessRate = float64(previousStats.SyncsSucceeded) / float64(previousStats.SyncsTotal) * 100
	}
	result.SuccessRateDelta = result.SuccessRate - prevSuccessRate
	result.TotalRecordsDelta = computeDelta(float64(previousStats.RecordsRead), float64(currentStats.RecordsRead))

	// Last sync time.
	lastSyncAt, err := uc.statsStorage.QueryLastSyncAt(ctx, params.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("querying last sync time: %w", err)
	}
	result.LastSyncAt = lastSyncAt

	// Health badge.
	schedule := ""
	if conn.Schedule != nil {
		schedule = *conn.Schedule
	}

	if conn.Status != pipelineservice.ConnectionStatusActive {
		result.Health = pipelineservice.HealthDisabled
	} else if lastSyncAt != nil {
		// Get last job status.
		lastJobInfo, err := uc.statsStorage.QueryLastJobInfo(ctx, params.ConnectionID)
		if err != nil {
			return nil, fmt.Errorf("querying last job info: %w", err)
		}
		lastStatus := ""
		if lastJobInfo != nil {
			lastStatus = lastJobInfo.Status
		}
		result.Health = ComputeHealthStatus(lastStatus, schedule, *lastSyncAt, now)
	} else if schedule != "" {
		result.Health = pipelineservice.HealthHealthy
	} else {
		result.Health = pipelineservice.HealthDisabled
	}

	// Duration chart: last 20 jobs with duration.
	durationRows, err := uc.statsStorage.QueryDurationChart(ctx, params.ConnectionID, currentStart, now, 20)
	if err != nil {
		return nil, fmt.Errorf("querying duration chart: %w", err)
	}
	// Reverse to chronological order and map to presentation types.
	points := make([]pipelineservice.DurationChartPoint, len(durationRows))
	for i, r := range durationRows {
		label := ""
		if r.CompletedAt != nil {
			label = r.CompletedAt.Format("Jan 2 15:04")
		}
		points[len(durationRows)-1-i] = pipelineservice.DurationChartPoint{
			Label:      label,
			DurationMs: r.DurationMs,
			Status:     parseSyncStatus(r.Status),
		}
	}
	result.DurationChart = points

	// Records chart: time-bucketed from rollups.
	recordRows, err := uc.statsStorage.QueryRecordsChart(ctx, params.ConnectionID, currentStart, now)
	if err != nil {
		return nil, fmt.Errorf("querying records chart: %w", err)
	}
	recPoints := make([]pipelineservice.RecordsChartPoint, len(recordRows))
	for i, r := range recordRows {
		recPoints[i] = pipelineservice.RecordsChartPoint{
			Label:       formatBucketLabel(r.BucketStart, params.TimeRange),
			RecordsRead: r.RecordsRead,
		}
	}
	result.RecordsChart = recPoints

	// Failure breakdown.
	failedJobs, err := uc.statsStorage.QueryConnectionFailedJobs(ctx, params.ConnectionID, currentStart, now)
	if err != nil {
		return nil, fmt.Errorf("querying failure breakdown: %w", err)
	}
	result.FailureBreakdown = categorizeFailedJobs(failedJobs)

	return result, nil
}

// categorizeFailedJobs groups failed jobs by failure category.
func categorizeFailedJobs(failedJobs []pipelineservice.FailedJobRow) []pipelineservice.FailureCategoryItem {
	counts := make(map[pipelineservice.FailureCategory]int32)
	for _, fj := range failedJobs {
		errStr := ""
		if fj.FailureReason != nil {
			errStr = *fj.FailureReason
		} else if fj.Error != nil {
			errStr = *fj.Error
		}
		category := CategorizeFailure(errStr)
		counts[category]++
	}

	items := make([]pipelineservice.FailureCategoryItem, 0, len(counts))
	for cat, count := range counts {
		items = append(items, pipelineservice.FailureCategoryItem{Category: cat, Count: count})
	}

	return items
}

// formatBucketLabel formats a bucket timestamp as a chart label.
func formatBucketLabel(t time.Time, timeRange pipelineservice.TimeRange) string {
	if timeRange == pipelineservice.TimeRange24h {
		return t.Format("15:04")
	}
	return t.Format("Jan 2")
}
