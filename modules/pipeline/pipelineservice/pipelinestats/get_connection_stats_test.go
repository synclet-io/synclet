package pipelinestats_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestats"
)

func newActiveConnection(workspaceID uuid.UUID, schedule *string) *pipelineservice.Connection {
	return &pipelineservice.Connection{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        "orders",
		Status:      pipelineservice.ConnectionStatusActive,
		Schedule:    schedule,
	}
}

func executeConnectionStats(t *testing.T, conn *pipelineservice.Connection, stats *fakeStatsStorage, timeRange pipelineservice.TimeRange) *pipelineservice.ConnectionStatsResult {
	t.Helper()

	result, err := pipelinestats.NewGetConnectionStats(stats, newFakeStorage(conn)).Execute(context.Background(), pipelineservice.GetConnectionStatsParams{
		ConnectionID: conn.ID,
		WorkspaceID:  conn.WorkspaceID,
		TimeRange:    timeRange,
	})
	require.NoError(t, err)

	return result
}

func TestGetConnectionStats_EmptyData(t *testing.T) {
	conn := newActiveConnection(uuid.New(), ptr(everyMinute))

	result := executeConnectionStats(t, conn, &fakeStatsStorage{}, pipelineservice.TimeRange24h)

	assert.Zero(t, result.TotalRecords)
	assert.Zero(t, result.AvgDurationMs)
	assert.Zero(t, result.SuccessRate)
	assert.Zero(t, result.AvgDurationDelta)
	assert.Zero(t, result.SuccessRateDelta)
	assert.Zero(t, result.TotalRecordsDelta)
	assert.Nil(t, result.LastSyncAt)
	assert.Equal(t, pipelineservice.HealthHealthy, result.Health, "scheduled but never synced")
	assert.Empty(t, result.DurationChart)
	assert.Empty(t, result.RecordsChart)
	assert.Empty(t, result.FailureBreakdown)
}

func TestGetConnectionStats_Aggregation(t *testing.T) {
	conn := newActiveConnection(uuid.New(), ptr(everyMinute))
	stats := &fakeStatsStorage{buckets: []rollupBucket{
		{connectionID: conn.ID, offset: -time.Hour, syncsSucceeded: 3, syncsFailed: 1, recordsRead: 400, totalDurationMs: 4000},
		{connectionID: conn.ID, offset: -2 * time.Hour, syncsSucceeded: 1, recordsRead: 100, totalDurationMs: 1000},
		{connectionID: conn.ID, offset: -25 * time.Hour, syncsSucceeded: 1, syncsFailed: 1, recordsRead: 250, totalDurationMs: 4000},
		{connectionID: uuid.New(), offset: -time.Hour, syncsSucceeded: 50, recordsRead: 99999, totalDurationMs: 1},
	}}

	result := executeConnectionStats(t, conn, stats, pipelineservice.TimeRange24h)

	assert.Equal(t, int64(500), result.TotalRecords)
	assert.Equal(t, int64(1000), result.AvgDurationMs)
	assert.InDelta(t, 80.0, result.SuccessRate, 1e-9)
	assert.InDelta(t, -50.0, result.AvgDurationDelta, 1e-9)
	assert.InDelta(t, 30.0, result.SuccessRateDelta, 1e-9)
	assert.InDelta(t, 100.0, result.TotalRecordsDelta, 1e-9)
}

func TestGetConnectionStats_WindowEdges(t *testing.T) {
	const day = 24 * time.Hour

	conn := newActiveConnection(uuid.New(), ptr(everyMinute))
	stats := &fakeStatsStorage{buckets: []rollupBucket{
		{connectionID: conn.ID, offset: -day, syncsSucceeded: 1, recordsRead: 10},
		{connectionID: conn.ID, offset: 0, syncsSucceeded: 1, recordsRead: 1000},
		{connectionID: conn.ID, offset: -day - time.Nanosecond, syncsFailed: 1, recordsRead: 100},
		{connectionID: conn.ID, offset: -2 * day, syncsSucceeded: 1, recordsRead: 200},
		{connectionID: conn.ID, offset: -2*day - time.Nanosecond, syncsSucceeded: 1, recordsRead: 10000},
	}}

	result := executeConnectionStats(t, conn, stats, pipelineservice.TimeRange24h)

	require.Len(t, stats.rollupWindows, 2)
	current, previous := stats.rollupWindows[0], stats.rollupWindows[1]
	assert.Equal(t, day, current.to.Sub(current.from))
	assert.Equal(t, current.from, previous.to, "previous window ends where current starts")
	assert.Equal(t, day, previous.to.Sub(previous.from))

	assert.Equal(t, int64(10), result.TotalRecords, "bucket at window start included, bucket at window end excluded")
	assert.InDelta(t, 100.0, result.SuccessRate, 1e-9)
	assert.InDelta(t, (10.0-300.0)/300.0*100, result.TotalRecordsDelta, 1e-9, "previous window covers [start-24h, start)")
	assert.InDelta(t, 50.0, result.SuccessRateDelta, 1e-9)
}

func TestGetConnectionStats_QueryWindows(t *testing.T) {
	conn := newActiveConnection(uuid.New(), nil)
	stats := &fakeStatsStorage{}
	before := time.Now()

	executeConnectionStats(t, conn, stats, pipelineservice.TimeRange7d)

	after := time.Now()
	current := stats.rollupWindows[0]

	assert.WithinRange(t, current.to, before, after)
	assert.Equal(t, 7*24*time.Hour, current.to.Sub(current.from))
	assert.Equal(t, current, stats.durationWindow)
	assert.Equal(t, current, stats.recordsWindow)
	assert.Equal(t, current, stats.failedJobsWindow)
	assert.Equal(t, 20, stats.durationLimit)
}

func TestGetConnectionStats_Health(t *testing.T) {
	workspaceID := uuid.New()

	t.Run("inactive connection is disabled without looking at jobs", func(t *testing.T) {
		conn := newActiveConnection(workspaceID, ptr(everyMinute))
		conn.Status = pipelineservice.ConnectionStatusInactive
		stats := &fakeStatsStorage{lastSyncAt: ptr(minuteAligned())}

		result := executeConnectionStats(t, conn, stats, pipelineservice.TimeRange24h)

		assert.Equal(t, pipelineservice.HealthDisabled, result.Health)
		assert.False(t, stats.queriedLastJob)
	})

	t.Run("manual connection without syncs is disabled", func(t *testing.T) {
		result := executeConnectionStats(t, newActiveConnection(workspaceID, nil), &fakeStatsStorage{}, pipelineservice.TimeRange24h)

		assert.Equal(t, pipelineservice.HealthDisabled, result.Health)
	})

	t.Run("recent successful sync is healthy", func(t *testing.T) {
		lastSync := minuteAligned()
		stats := &fakeStatsStorage{
			lastSyncAt:  &lastSync,
			lastJobInfo: &pipelineservice.LastJobInfo{Status: "completed", CompletedAt: &lastSync},
		}

		result := executeConnectionStats(t, newActiveConnection(workspaceID, ptr(everyMinute)), stats, pipelineservice.TimeRange24h)

		assert.Equal(t, pipelineservice.HealthHealthy, result.Health)
		assert.Equal(t, &lastSync, result.LastSyncAt)
	})

	t.Run("last job failed is failing", func(t *testing.T) {
		stats := &fakeStatsStorage{
			lastSyncAt:  ptr(minuteAligned()),
			lastJobInfo: &pipelineservice.LastJobInfo{Status: "failed"},
		}

		result := executeConnectionStats(t, newActiveConnection(workspaceID, ptr(everyMinute)), stats, pipelineservice.TimeRange24h)

		assert.Equal(t, pipelineservice.HealthFailing, result.Health)
	})

	t.Run("missing last job info falls back to timing", func(t *testing.T) {
		stats := &fakeStatsStorage{lastSyncAt: ptr(minuteAligned())}

		result := executeConnectionStats(t, newActiveConnection(workspaceID, ptr(everyMinute)), stats, pipelineservice.TimeRange24h)

		assert.Equal(t, pipelineservice.HealthHealthy, result.Health)
	})
}

func TestGetConnectionStats_DurationChart(t *testing.T) {
	newest := time.Date(2026, 5, 17, 12, 30, 0, 0, time.UTC)
	older := time.Date(2026, 5, 16, 9, 5, 0, 0, time.UTC)
	stats := &fakeStatsStorage{durationRows: []pipelineservice.DurationChartRow{
		{CompletedAt: &newest, Status: "Failed", DurationMs: 300},
		{CompletedAt: &older, Status: "completed", DurationMs: 200},
		{Status: "running", DurationMs: 100},
	}}

	result := executeConnectionStats(t, newActiveConnection(uuid.New(), nil), stats, pipelineservice.TimeRange24h)

	assert.Equal(t, []pipelineservice.DurationChartPoint{
		{Label: "", DurationMs: 100, Status: pipelineservice.SyncStatusCompleted},
		{Label: "May 16 09:05", DurationMs: 200, Status: pipelineservice.SyncStatusCompleted},
		{Label: "May 17 12:30", DurationMs: 300, Status: pipelineservice.SyncStatusFailed},
	}, result.DurationChart)
}

func TestGetConnectionStats_RecordsChartLabels(t *testing.T) {
	bucket := time.Date(2026, 5, 17, 14, 0, 0, 0, time.UTC)
	rows := []pipelineservice.RecordsChartRow{{BucketStart: bucket, RecordsRead: 42}}

	hourly := executeConnectionStats(t, newActiveConnection(uuid.New(), nil), &fakeStatsStorage{recordRows: rows}, pipelineservice.TimeRange24h)
	daily := executeConnectionStats(t, newActiveConnection(uuid.New(), nil), &fakeStatsStorage{recordRows: rows}, pipelineservice.TimeRange30d)

	assert.Equal(t, []pipelineservice.RecordsChartPoint{{Label: "14:00", RecordsRead: 42}}, hourly.RecordsChart)
	assert.Equal(t, []pipelineservice.RecordsChartPoint{{Label: "May 17", RecordsRead: 42}}, daily.RecordsChart)
}

func TestGetConnectionStats_FailureBreakdown(t *testing.T) {
	stats := &fakeStatsStorage{failedJobs: []pipelineservice.FailedJobRow{
		{FailureReason: ptr("context deadline exceeded"), Error: ptr("docker died")},
		{Error: ptr("operation timeout")},
		{Error: ptr("container OOM killed")},
		{},
	}}

	result := executeConnectionStats(t, newActiveConnection(uuid.New(), nil), stats, pipelineservice.TimeRange24h)

	assert.ElementsMatch(t, []pipelineservice.FailureCategoryItem{
		{Category: pipelineservice.FailureCategoryTimeout, Count: 2},
		{Category: pipelineservice.FailureCategoryOOM, Count: 1},
		{Category: pipelineservice.FailureCategoryUnknown, Count: 1},
	}, result.FailureBreakdown)
}

func TestGetConnectionStats_ForeignConnection(t *testing.T) {
	conn := newActiveConnection(uuid.New(), nil)
	stats := &fakeStatsStorage{}

	_, err := pipelinestats.NewGetConnectionStats(stats, newFakeStorage(conn)).Execute(context.Background(), pipelineservice.GetConnectionStatsParams{
		ConnectionID: conn.ID,
		WorkspaceID:  uuid.New(),
	})

	require.ErrorIs(t, err, pipelineservice.ErrConnectionNotFound)
	assert.Empty(t, stats.rollupWindows)
}

func TestGetConnectionStats_StorageErrors(t *testing.T) {
	methods := []string{
		"QueryConnectionRollup",
		"QueryConnectionRollup:previous",
		"QueryLastSyncAt",
		"QueryLastJobInfo",
		"QueryDurationChart",
		"QueryRecordsChart",
		"QueryConnectionFailedJobs",
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			conn := newActiveConnection(uuid.New(), ptr(everyMinute))
			stats := &fakeStatsStorage{
				lastSyncAt: ptr(minuteAligned()),
				errs:       map[string]error{method: errStorage},
			}

			result, err := pipelinestats.NewGetConnectionStats(stats, newFakeStorage(conn)).Execute(context.Background(), pipelineservice.GetConnectionStatsParams{
				ConnectionID: conn.ID,
				WorkspaceID:  conn.WorkspaceID,
			})

			require.ErrorIs(t, err, errStorage)
			assert.Nil(t, result)
		})
	}
}

func TestGetConnectionStats_BucketSize(t *testing.T) {
	cases := map[pipelineservice.TimeRange]pipelineservice.BucketSize{
		pipelineservice.TimeRange24h: pipelineservice.BucketSizeHourly,
		pipelineservice.TimeRange7d:  pipelineservice.BucketSizeDaily,
		pipelineservice.TimeRange30d: pipelineservice.BucketSizeDaily,
	}

	for timeRange, want := range cases {
		t.Run(string(timeRange), func(t *testing.T) {
			stats := &fakeStatsStorage{}
			executeConnectionStats(t, newActiveConnection(uuid.New(), nil), stats, timeRange)

			assert.Equal(t, map[string][]pipelineservice.BucketSize{
				"QueryConnectionRollup": {want, want},
				"QueryRecordsChart":     {want},
			}, stats.bucketSizes)
		})
	}
}
