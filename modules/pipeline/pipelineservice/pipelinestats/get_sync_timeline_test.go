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

func TestGetSyncTimeline_BucketSizeAndWindow(t *testing.T) {
	cases := []struct {
		timeRange  pipelineservice.TimeRange
		bucketSize pipelineservice.BucketSize
		duration   time.Duration
		label      string
	}{
		{pipelineservice.TimeRange24h, pipelineservice.BucketSizeHourly, 24 * time.Hour, "14:00"},
		{pipelineservice.TimeRange7d, pipelineservice.BucketSizeDaily, 7 * 24 * time.Hour, "May 17"},
		{pipelineservice.TimeRange30d, pipelineservice.BucketSizeDaily, 30 * 24 * time.Hour, "May 17"},
	}

	for _, tc := range cases {
		t.Run(string(tc.timeRange), func(t *testing.T) {
			workspaceID := uuid.New()
			stats := &fakeStatsStorage{timelineRows: []pipelineservice.TimelineRow{
				{BucketStart: time.Date(2026, 5, 17, 14, 0, 0, 0, time.UTC)},
			}}
			before := time.Now()

			result, err := pipelinestats.NewGetSyncTimeline(newFakeStorage(), stats).Execute(context.Background(), pipelineservice.GetSyncTimelineParams{
				WorkspaceID: workspaceID,
				TimeRange:   tc.timeRange,
			})
			require.NoError(t, err)

			after := time.Now()

			require.Len(t, stats.timelineCalls, 1)
			call := stats.timelineCalls[0]
			assert.Equal(t, workspaceID, call.workspaceID)
			assert.Equal(t, tc.bucketSize, call.bucketSize)
			assert.Nil(t, call.connectionID)
			assert.WithinRange(t, call.window.to, before, after)
			assert.Equal(t, tc.duration, call.window.to.Sub(call.window.from))
			assert.Equal(t, tc.label, result.Points[0].Label)
		})
	}
}

func TestGetSyncTimeline_MapsBucketsInOrder(t *testing.T) {
	base := time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC)
	stats := &fakeStatsStorage{timelineRows: []pipelineservice.TimelineRow{
		{BucketStart: base, SyncsSucceeded: 3, SyncsFailed: 1, RecordsRead: 100, BytesSynced: 2048, AvgDurationMs: 1500},
		{BucketStart: base.Add(time.Hour), SyncsSucceeded: 0, SyncsFailed: 2, RecordsRead: 0, BytesSynced: 0, AvgDurationMs: 300},
		{BucketStart: base.Add(2 * time.Hour), SyncsSucceeded: 5, RecordsRead: 900, BytesSynced: 4096, AvgDurationMs: 2000},
	}}

	result, err := pipelinestats.NewGetSyncTimeline(newFakeStorage(), stats).Execute(context.Background(), pipelineservice.GetSyncTimelineParams{
		WorkspaceID: uuid.New(),
		TimeRange:   pipelineservice.TimeRange24h,
	})
	require.NoError(t, err)

	assert.Equal(t, []pipelineservice.TimelinePointItem{
		{Label: "10:00", Succeeded: 3, Failed: 1},
		{Label: "11:00", Succeeded: 0, Failed: 2},
		{Label: "12:00", Succeeded: 5, Failed: 0},
	}, result.Points)
	assert.Equal(t, []pipelineservice.ThroughputPointItem{
		{Label: "10:00", RecordsRead: 100, BytesSynced: 2048},
		{Label: "11:00"},
		{Label: "12:00", RecordsRead: 900, BytesSynced: 4096},
	}, result.Throughput)
	assert.Equal(t, []pipelineservice.DurationPointItem{
		{Label: "10:00", AvgDurationMs: 1500},
		{Label: "11:00", AvgDurationMs: 300},
		{Label: "12:00", AvgDurationMs: 2000},
	}, result.Durations)
}

func TestGetSyncTimeline_EmptyData(t *testing.T) {
	result, err := pipelinestats.NewGetSyncTimeline(newFakeStorage(), &fakeStatsStorage{}).Execute(context.Background(), pipelineservice.GetSyncTimelineParams{
		WorkspaceID: uuid.New(),
		TimeRange:   pipelineservice.TimeRange7d,
	})
	require.NoError(t, err)

	assert.Empty(t, result.Points)
	assert.Empty(t, result.Throughput)
	assert.Empty(t, result.Durations)
	assert.NotNil(t, result.Points)
}

func TestGetSyncTimeline_ConnectionScope(t *testing.T) {
	workspaceID := uuid.New()
	conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: workspaceID}

	t.Run("owned connection is passed through to the query", func(t *testing.T) {
		stats := &fakeStatsStorage{}

		_, err := pipelinestats.NewGetSyncTimeline(newFakeStorage(conn), stats).Execute(context.Background(), pipelineservice.GetSyncTimelineParams{
			WorkspaceID:  workspaceID,
			ConnectionID: &conn.ID,
			TimeRange:    pipelineservice.TimeRange24h,
		})
		require.NoError(t, err)

		require.Len(t, stats.timelineCalls, 1)
		require.NotNil(t, stats.timelineCalls[0].connectionID)
		assert.Equal(t, conn.ID, *stats.timelineCalls[0].connectionID)
	})

	t.Run("connection from another workspace is rejected", func(t *testing.T) {
		stats := &fakeStatsStorage{}

		_, err := pipelinestats.NewGetSyncTimeline(newFakeStorage(conn), stats).Execute(context.Background(), pipelineservice.GetSyncTimelineParams{
			WorkspaceID:  uuid.New(),
			ConnectionID: &conn.ID,
			TimeRange:    pipelineservice.TimeRange24h,
		})

		require.ErrorIs(t, err, pipelineservice.ErrConnectionNotFound)
		assert.Empty(t, stats.timelineCalls)
	})

	t.Run("nil connection from storage is not found", func(t *testing.T) {
		storage := newFakeStorage(conn)
		storage.connections.firstNil = true

		_, err := pipelinestats.NewGetSyncTimeline(storage, &fakeStatsStorage{}).Execute(context.Background(), pipelineservice.GetSyncTimelineParams{
			WorkspaceID:  workspaceID,
			ConnectionID: &conn.ID,
		})

		require.ErrorIs(t, err, pipelineservice.ErrConnectionNotFound)
	})
}

func TestGetSyncTimeline_QueryError(t *testing.T) {
	stats := &fakeStatsStorage{errs: map[string]error{"QuerySyncTimeline": errStorage}}

	_, err := pipelinestats.NewGetSyncTimeline(newFakeStorage(), stats).Execute(context.Background(), pipelineservice.GetSyncTimelineParams{
		WorkspaceID: uuid.New(),
	})

	require.ErrorIs(t, err, errStorage)
}
