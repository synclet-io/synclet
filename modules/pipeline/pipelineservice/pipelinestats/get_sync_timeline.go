package pipelinestats

import (
	"context"
	"fmt"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetSyncTimeline retrieves time-bucketed sync data from rollups.
type GetSyncTimeline struct {
	storage      pipelineservice.Storage
	statsStorage pipelineservice.StatsStorage
}

// NewGetSyncTimeline creates a new GetSyncTimeline use case.
func NewGetSyncTimeline(storage pipelineservice.Storage, statsStorage pipelineservice.StatsStorage) *GetSyncTimeline {
	return &GetSyncTimeline{storage: storage, statsStorage: statsStorage}
}

// Execute returns time-bucketed sync timeline data with auto-selected granularity.
// 24h -> hourly buckets, 7d/30d -> daily buckets.
func (uc *GetSyncTimeline) Execute(ctx context.Context, params pipelineservice.GetSyncTimelineParams) (*pipelineservice.SyncTimelineResult, error) {
	// Verify connection belongs to workspace when a specific connection is requested.
	if params.ConnectionID != nil {
		conn, err := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
			ID:          filter.Equals(*params.ConnectionID),
			WorkspaceID: filter.Equals(params.WorkspaceID),
		})
		if err != nil {
			return nil, fmt.Errorf("verifying connection ownership: %w", err)
		}
		if conn == nil {
			return nil, pipelineservice.ErrConnectionNotFound
		}
	}

	duration := params.TimeRange.Duration()
	now := time.Now()
	from := now.Add(-duration)

	// Auto-select bucket size per D-20.
	bucketSize := pipelineservice.BucketSizeDaily
	if params.TimeRange == pipelineservice.TimeRange24h {
		bucketSize = pipelineservice.BucketSizeHourly
	}

	rows, err := uc.statsStorage.QuerySyncTimeline(ctx, params.WorkspaceID, bucketSize, from, now, params.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("querying timeline: %w", err)
	}

	result := &pipelineservice.SyncTimelineResult{
		Points:     make([]pipelineservice.TimelinePointItem, len(rows)),
		Throughput: make([]pipelineservice.ThroughputPointItem, len(rows)),
		Durations:  make([]pipelineservice.DurationPointItem, len(rows)),
	}

	for i, r := range rows {
		label := formatBucketLabel(r.BucketStart, params.TimeRange)

		result.Points[i] = pipelineservice.TimelinePointItem{
			Label:     label,
			Succeeded: r.SyncsSucceeded,
			Failed:    r.SyncsFailed,
		}

		result.Throughput[i] = pipelineservice.ThroughputPointItem{
			Label:       label,
			RecordsRead: r.RecordsRead,
			BytesSynced: r.BytesSynced,
		}

		result.Durations[i] = pipelineservice.DurationPointItem{
			Label:         label,
			AvgDurationMs: r.AvgDurationMs,
		}
	}

	return result, nil
}
