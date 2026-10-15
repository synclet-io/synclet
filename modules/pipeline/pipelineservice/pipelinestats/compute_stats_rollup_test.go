package pipelinestats_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestats"
)

func TestComputeStatsRollup(t *testing.T) {
	cfg := pipelineservice.Config{
		StatsRollupHourlyLookback: 2 * time.Hour,
		StatsRollupDailyLookback:  48 * time.Hour,
	}

	t.Run("upserts hourly then daily buckets with their lookbacks", func(t *testing.T) {
		stats := &fakeStatsStorage{}
		before := time.Now()

		require.NoError(t, pipelinestats.NewComputeStatsRollup(stats, cfg).Execute(context.Background()))

		after := time.Now()

		require.Len(t, stats.upserts, 2)

		hourly, daily := stats.upserts[0], stats.upserts[1]
		assert.Equal(t, pipelineservice.BucketSizeHourly, hourly.bucketSize)
		assert.Equal(t, "hour", hourly.truncUnit)
		assert.WithinRange(t, hourly.since, before.Add(-2*time.Hour), after.Add(-2*time.Hour))

		assert.Equal(t, pipelineservice.BucketSizeDaily, daily.bucketSize)
		assert.Equal(t, "day", daily.truncUnit)
		assert.WithinRange(t, daily.since, before.Add(-48*time.Hour), after.Add(-48*time.Hour))
	})

	t.Run("hourly failure stops before daily", func(t *testing.T) {
		stats := &fakeStatsStorage{errs: map[string]error{"UpsertRollups:hour": errStorage}}

		err := pipelinestats.NewComputeStatsRollup(stats, cfg).Execute(context.Background())

		require.ErrorIs(t, err, errStorage)
		assert.Contains(t, err.Error(), "computing hourly rollups")
		assert.Len(t, stats.upserts, 1)
	})

	t.Run("daily failure is reported", func(t *testing.T) {
		stats := &fakeStatsStorage{errs: map[string]error{"UpsertRollups:day": errStorage}}

		err := pipelinestats.NewComputeStatsRollup(stats, cfg).Execute(context.Background())

		require.ErrorIs(t, err, errStorage)
		assert.Contains(t, err.Error(), "computing daily rollups")
		assert.Len(t, stats.upserts, 2)
	})
}
