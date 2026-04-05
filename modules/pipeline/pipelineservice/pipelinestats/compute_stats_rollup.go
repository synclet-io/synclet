package pipelinestats

import (
	"context"
	"fmt"
	"time"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ComputeStatsRollup computes pre-aggregated stats rollups from job data.
// Runs as a background job every 5 minutes for fast dashboard queries.
type ComputeStatsRollup struct {
	statsStorage pipelineservice.StatsStorage
	cfg          pipelineservice.Config
}

// NewComputeStatsRollup creates a new ComputeStatsRollup use case.
func NewComputeStatsRollup(statsStorage pipelineservice.StatsStorage, cfg pipelineservice.Config) *ComputeStatsRollup {
	return &ComputeStatsRollup{statsStorage: statsStorage, cfg: cfg}
}

// Execute computes hourly and daily rollup buckets from completed/failed jobs.
// Uses ON CONFLICT for idempotent upserts so re-runs produce the same results.
func (uc *ComputeStatsRollup) Execute(ctx context.Context) error {
	if err := uc.computeRollups(ctx, pipelineservice.BucketSizeHourly, uc.cfg.StatsRollupHourlyLookback); err != nil {
		return fmt.Errorf("computing hourly rollups: %w", err)
	}

	if err := uc.computeRollups(ctx, pipelineservice.BucketSizeDaily, uc.cfg.StatsRollupDailyLookback); err != nil {
		return fmt.Errorf("computing daily rollups: %w", err)
	}

	return nil
}

func (uc *ComputeStatsRollup) computeRollups(ctx context.Context, bucketSize pipelineservice.BucketSize, lookback time.Duration) error {
	truncUnit := "hour"
	if bucketSize == pipelineservice.BucketSizeDaily {
		truncUnit = "day"
	}

	since := time.Now().Add(-lookback)

	return uc.statsStorage.UpsertRollups(ctx, bucketSize, since, truncUnit)
}
