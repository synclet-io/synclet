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

func executeWorkspaceStats(t *testing.T, workspaceID uuid.UUID, storage *fakeStorage, stats *fakeStatsStorage) *pipelineservice.WorkspaceStatsResult {
	t.Helper()

	result, err := pipelinestats.NewGetWorkspaceStats(stats, storage, newLogger()).Execute(context.Background(), pipelineservice.GetWorkspaceStatsParams{
		WorkspaceID: workspaceID,
		TimeRange:   pipelineservice.TimeRange24h,
	})
	require.NoError(t, err)

	return result
}

func TestGetWorkspaceStats_EmptyWorkspace(t *testing.T) {
	result := executeWorkspaceStats(t, uuid.New(), newFakeStorage(), &fakeStatsStorage{})

	assert.Zero(t, result.TotalSyncs)
	assert.Zero(t, result.FailedSyncs)
	assert.Zero(t, result.RecordsSynced)
	assert.Zero(t, result.SuccessRate)
	assert.Zero(t, result.ActiveConnections)
	assert.Zero(t, result.TotalSyncsDelta)
	assert.Zero(t, result.FailedSyncsDelta)
	assert.Zero(t, result.RecordsSyncedDelta)
	assert.Zero(t, result.SuccessRateDelta)
	assert.Empty(t, result.ConnectionHealths)
	assert.Empty(t, result.TopConnections)
	assert.Empty(t, result.FailureBreakdown)
}

func TestGetWorkspaceStats_SingleConnection(t *testing.T) {
	workspaceID := uuid.New()
	conn := newActiveConnection(workspaceID, ptr(everyMinute))
	lastSync := minuteAligned()
	stats := &fakeStatsStorage{
		buckets: []rollupBucket{
			{connectionID: conn.ID, offset: -time.Hour, syncsSucceeded: 2, recordsRead: 300},
		},
		lastJobInfos: map[uuid.UUID]*pipelineservice.LastJobInfo{
			conn.ID: {Status: "completed", CompletedAt: &lastSync},
		},
	}

	result := executeWorkspaceStats(t, workspaceID, newFakeStorage(conn), stats)

	assert.Equal(t, int64(2), result.TotalSyncs)
	assert.Equal(t, int64(300), result.RecordsSynced)
	assert.InDelta(t, 100.0, result.SuccessRate, 1e-9)
	assert.InDelta(t, 100.0, result.TotalSyncsDelta, 1e-9, "growth from zero is reported as 100%")
	assert.InDelta(t, 100.0, result.SuccessRateDelta, 1e-9)
	assert.Equal(t, int32(1), result.ActiveConnections)
	assert.Equal(t, []pipelineservice.ConnectionHealthItem{
		{ConnectionID: conn.ID, ConnectionName: conn.Name, Health: pipelineservice.HealthHealthy, LastSyncAt: &lastSync},
	}, result.ConnectionHealths)
}

func TestGetWorkspaceStats_MixedConnections(t *testing.T) {
	workspaceID := uuid.New()
	lastSync := minuteAligned()

	healthy := newActiveConnection(workspaceID, ptr(everyMinute))
	failing := newActiveConnection(workspaceID, ptr(everyMinute))
	neverSynced := newActiveConnection(workspaceID, ptr(everyMinute))
	manual := newActiveConnection(workspaceID, nil)
	paused := newActiveConnection(workspaceID, ptr(everyMinute))
	paused.Status = pipelineservice.ConnectionStatusInactive
	foreign := newActiveConnection(uuid.New(), ptr(everyMinute))

	stats := &fakeStatsStorage{
		buckets: []rollupBucket{
			{connectionID: healthy.ID, offset: -time.Hour, syncsSucceeded: 6, recordsRead: 600},
			{connectionID: failing.ID, offset: -2 * time.Hour, syncsSucceeded: 1, syncsFailed: 3, recordsRead: 50},
			{connectionID: healthy.ID, offset: -30 * time.Hour, syncsSucceeded: 4, recordsRead: 400},
			{connectionID: failing.ID, offset: -30 * time.Hour, syncsFailed: 1},
		},
		lastJobInfos: map[uuid.UUID]*pipelineservice.LastJobInfo{
			healthy.ID: {Status: "completed", CompletedAt: &lastSync},
			failing.ID: {Status: "failed", CompletedAt: &lastSync},
			paused.ID:  {Status: "completed", CompletedAt: &lastSync},
		},
	}

	result := executeWorkspaceStats(t, workspaceID, newFakeStorage(healthy, failing, neverSynced, manual, paused, foreign), stats)

	assert.Equal(t, int64(10), result.TotalSyncs)
	assert.Equal(t, int64(3), result.FailedSyncs)
	assert.Equal(t, int64(650), result.RecordsSynced)
	assert.InDelta(t, 70.0, result.SuccessRate, 1e-9)
	assert.InDelta(t, 100.0, result.TotalSyncsDelta, 1e-9)
	assert.InDelta(t, 200.0, result.FailedSyncsDelta, 1e-9)
	assert.InDelta(t, 62.5, result.RecordsSyncedDelta, 1e-9)
	assert.InDelta(t, 70.0-80.0, result.SuccessRateDelta, 1e-9)
	assert.Equal(t, int32(4), result.ActiveConnections)

	assert.ElementsMatch(t, []uuid.UUID{healthy.ID, failing.ID, neverSynced.ID, manual.ID, paused.ID}, stats.lastJobInfoBatch)

	healths := make(map[uuid.UUID]pipelineservice.Health, len(result.ConnectionHealths))
	for _, item := range result.ConnectionHealths {
		healths[item.ConnectionID] = item.Health
	}

	assert.Equal(t, map[uuid.UUID]pipelineservice.Health{
		healthy.ID:     pipelineservice.HealthHealthy,
		failing.ID:     pipelineservice.HealthFailing,
		neverSynced.ID: pipelineservice.HealthHealthy,
		manual.ID:      pipelineservice.HealthHealthy,
		paused.ID:      pipelineservice.HealthDisabled,
	}, healths)
}

func TestGetWorkspaceStats_WindowEdges(t *testing.T) {
	const day = 24 * time.Hour

	connID := uuid.New()
	stats := &fakeStatsStorage{buckets: []rollupBucket{
		{connectionID: connID, offset: -day, syncsSucceeded: 1},
		{connectionID: connID, offset: 0, syncsSucceeded: 1, syncsFailed: 1},
		{connectionID: connID, offset: -day - time.Nanosecond, syncsFailed: 1},
		{connectionID: connID, offset: -2*day - time.Nanosecond, syncsFailed: 5},
	}}

	result := executeWorkspaceStats(t, uuid.New(), newFakeStorage(), stats)

	require.Len(t, stats.rollupWindows, 2)
	current, previous := stats.rollupWindows[0], stats.rollupWindows[1]
	assert.Equal(t, current.from, previous.to)
	assert.Equal(t, day, current.to.Sub(current.from))
	assert.Equal(t, day, previous.to.Sub(previous.from))
	assert.Equal(t, current, stats.failedJobsWindow)
	assert.Equal(t, current, stats.topWindow)

	assert.Equal(t, int64(1), result.TotalSyncs, "bucket at window start included, bucket at window end excluded")
	assert.Zero(t, result.FailedSyncs)
	assert.InDelta(t, -100.0, result.FailedSyncsDelta, 1e-9, "previous window holds one failure, older bucket excluded")
}

func TestGetWorkspaceStats_TopConnections(t *testing.T) {
	workspaceID := uuid.New()
	first := newActiveConnection(workspaceID, nil)
	first.Name = "first"
	second := newActiveConnection(workspaceID, nil)
	second.Name = "second"
	unknownID := uuid.New()
	lastCompleted := time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC)

	stats := &fakeStatsStorage{
		topConnections: []pipelineservice.TopConnectionRow{
			{ConnectionID: second.ID, RecordsSynced: 900, BytesSynced: 9000},
			{ConnectionID: first.ID, RecordsSynced: 100, BytesSynced: 1000},
			{ConnectionID: unknownID, RecordsSynced: 1},
		},
		lastCompleted: map[uuid.UUID]*time.Time{second.ID: &lastCompleted},
		sparklines: map[uuid.UUID][]int64{
			second.ID: {30, 20, 10},
			first.ID:  {5, 4},
		},
	}

	result := executeWorkspaceStats(t, workspaceID, newFakeStorage(first, second), stats)

	assert.Equal(t, 5, stats.topLimit)
	assert.Equal(t, 7, stats.sparklineLimit)
	assert.Equal(t, []pipelineservice.TopConnectionItem{
		{ConnectionID: second.ID, ConnectionName: "second", RecordsSynced: 900, BytesSynced: 9000, LastSyncAt: &lastCompleted, SparklineValues: []int64{10, 20, 30}},
		{ConnectionID: first.ID, ConnectionName: "first", RecordsSynced: 100, BytesSynced: 1000, SparklineValues: []int64{4, 5}},
		{ConnectionID: unknownID, RecordsSynced: 1},
	}, result.TopConnections)
}

func TestGetWorkspaceStats_BestEffortQueries(t *testing.T) {
	workspaceID := uuid.New()
	conn := newActiveConnection(workspaceID, ptr(everyMinute))

	t.Run("top connections failure is tolerated", func(t *testing.T) {
		stats := &fakeStatsStorage{errs: map[string]error{"QueryTopConnections": errStorage}}

		result := executeWorkspaceStats(t, workspaceID, newFakeStorage(conn), stats)

		assert.Nil(t, result.TopConnections)
	})

	t.Run("batch lookup failures degrade to empty data", func(t *testing.T) {
		stats := &fakeStatsStorage{
			topConnections: []pipelineservice.TopConnectionRow{{ConnectionID: conn.ID, RecordsSynced: 10}},
			errs: map[string]error{
				"QueryLastJobInfoBatch":               errStorage,
				"QueryConnectionLastCompletedAtBatch": errStorage,
				"QueryConnectionSparklineBatch":       errStorage,
			},
		}

		result := executeWorkspaceStats(t, workspaceID, newFakeStorage(conn), stats)

		assert.Equal(t, []pipelineservice.ConnectionHealthItem{
			{ConnectionID: conn.ID, ConnectionName: conn.Name, Health: pipelineservice.HealthHealthy},
		}, result.ConnectionHealths)
		assert.Equal(t, []pipelineservice.TopConnectionItem{
			{ConnectionID: conn.ID, ConnectionName: conn.Name, RecordsSynced: 10},
		}, result.TopConnections)
	})
}

func TestGetWorkspaceStats_FailureBreakdown(t *testing.T) {
	stats := &fakeStatsStorage{failedJobs: []pipelineservice.FailedJobRow{
		{Error: ptr("kubernetes pod evicted")},
		{FailureReason: ptr("source error: bad credentials")},
		{Error: ptr("docker daemon unavailable")},
	}}

	result := executeWorkspaceStats(t, uuid.New(), newFakeStorage(), stats)

	assert.ElementsMatch(t, []pipelineservice.FailureCategoryItem{
		{Category: pipelineservice.FailureCategoryInfrastructure, Count: 2},
		{Category: pipelineservice.FailureCategoryConnector, Count: 1},
	}, result.FailureBreakdown)
}

func TestGetWorkspaceStats_StorageErrors(t *testing.T) {
	for _, method := range []string{"QueryWorkspaceRollup", "QueryWorkspaceRollup:previous", "QueryWorkspaceFailedJobs"} {
		t.Run(method, func(t *testing.T) {
			stats := &fakeStatsStorage{errs: map[string]error{method: errStorage}}

			result, err := pipelinestats.NewGetWorkspaceStats(stats, newFakeStorage(), newLogger()).Execute(context.Background(), pipelineservice.GetWorkspaceStatsParams{
				WorkspaceID: uuid.New(),
			})

			require.ErrorIs(t, err, errStorage)
			assert.Nil(t, result)
		})
	}

	t.Run("listing connections", func(t *testing.T) {
		storage := newFakeStorage()
		storage.connections.err = errStorage

		_, err := pipelinestats.NewGetWorkspaceStats(&fakeStatsStorage{}, storage, newLogger()).Execute(context.Background(), pipelineservice.GetWorkspaceStatsParams{
			WorkspaceID: uuid.New(),
		})

		require.ErrorIs(t, err, errStorage)
	})
}

func TestGetWorkspaceStats_BucketSize(t *testing.T) {
	cases := map[pipelineservice.TimeRange]pipelineservice.BucketSize{
		pipelineservice.TimeRange24h: pipelineservice.BucketSizeHourly,
		pipelineservice.TimeRange7d:  pipelineservice.BucketSizeDaily,
		pipelineservice.TimeRange30d: pipelineservice.BucketSizeDaily,
	}

	for timeRange, want := range cases {
		t.Run(string(timeRange), func(t *testing.T) {
			workspaceID := uuid.New()
			conn := newActiveConnection(workspaceID, nil)
			stats := &fakeStatsStorage{
				topConnections: []pipelineservice.TopConnectionRow{{ConnectionID: conn.ID, RecordsSynced: 1}},
			}

			_, err := pipelinestats.NewGetWorkspaceStats(stats, newFakeStorage(conn), newLogger()).Execute(context.Background(), pipelineservice.GetWorkspaceStatsParams{
				WorkspaceID: workspaceID,
				TimeRange:   timeRange,
			})
			require.NoError(t, err)

			assert.Equal(t, map[string][]pipelineservice.BucketSize{
				"QueryWorkspaceRollup":          {want, want},
				"QueryTopConnections":           {want},
				"QueryConnectionSparklineBatch": {want},
			}, stats.bucketSizes)
		})
	}
}
