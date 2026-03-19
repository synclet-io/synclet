package pipelinestats

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetWorkspaceStats retrieves workspace-level stats from pre-computed rollups.
type GetWorkspaceStats struct {
	statsStorage pipelineservice.StatsStorage
	storage      pipelineservice.Storage
	logger       *logging.Logger
}

// NewGetWorkspaceStats creates a new GetWorkspaceStats use case.
func NewGetWorkspaceStats(statsStorage pipelineservice.StatsStorage, storage pipelineservice.Storage, logger *logging.Logger) *GetWorkspaceStats {
	return &GetWorkspaceStats{statsStorage: statsStorage, storage: storage, logger: logger}
}

// Execute computes workspace stats for the given time range.
func (uc *GetWorkspaceStats) Execute(ctx context.Context, params pipelineservice.GetWorkspaceStatsParams) (*pipelineservice.WorkspaceStatsResult, error) {
	duration := params.TimeRange.Duration()
	now := time.Now()
	currentStart := now.Add(-duration)
	previousStart := currentStart.Add(-duration)

	result := &pipelineservice.WorkspaceStatsResult{}

	// Query current period rollup totals.
	currentStats, err := uc.statsStorage.QueryWorkspaceRollup(ctx, params.WorkspaceID, currentStart, now)
	if err != nil {
		return nil, fmt.Errorf("querying current stats: %w", err)
	}

	result.TotalSyncs = currentStats.SyncsTotal
	result.FailedSyncs = currentStats.SyncsFailed
	result.RecordsSynced = currentStats.RecordsRead
	if currentStats.SyncsTotal > 0 {
		result.SuccessRate = float64(currentStats.SyncsSucceeded) / float64(currentStats.SyncsTotal) * 100
	}

	// Query previous period for trend deltas.
	previousStats, err := uc.statsStorage.QueryWorkspaceRollup(ctx, params.WorkspaceID, previousStart, currentStart)
	if err != nil {
		return nil, fmt.Errorf("querying previous stats: %w", err)
	}

	result.TotalSyncsDelta = computeDelta(float64(previousStats.SyncsTotal), float64(currentStats.SyncsTotal))
	result.FailedSyncsDelta = computeDelta(float64(previousStats.SyncsFailed), float64(currentStats.SyncsFailed))
	result.RecordsSyncedDelta = computeDelta(float64(previousStats.RecordsRead), float64(currentStats.RecordsRead))

	var prevSuccessRate float64
	if previousStats.SyncsTotal > 0 {
		prevSuccessRate = float64(previousStats.SyncsSucceeded) / float64(previousStats.SyncsTotal) * 100
	}
	result.SuccessRateDelta = result.SuccessRate - prevSuccessRate

	// Active connections count.
	connections, err := uc.storage.Connections().Find(ctx, &pipelineservice.ConnectionFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing connections: %w", err)
	}

	var activeCount int32
	for _, conn := range connections {
		if conn.Status == pipelineservice.ConnectionStatusActive {
			activeCount++
		}
	}
	result.ActiveConnections = activeCount

	// Connection health grid.
	result.ConnectionHealths = uc.computeConnectionHealths(ctx, connections)

	// Top connections by records synced.
	topConns, topErr := uc.queryTopConnections(ctx, params.WorkspaceID, currentStart, now, connections)
	if topErr != nil {
		uc.logger.WithError(topErr).Warn(ctx, "failed to query top connections for dashboard")
	}
	result.TopConnections = topConns

	// Failure breakdown.
	failedJobs, err := uc.statsStorage.QueryWorkspaceFailedJobs(ctx, params.WorkspaceID, currentStart, now)
	if err != nil {
		return nil, fmt.Errorf("querying failure breakdown: %w", err)
	}
	result.FailureBreakdown = categorizeFailedJobs(failedJobs)

	return result, nil
}

func (uc *GetWorkspaceStats) computeConnectionHealths(ctx context.Context, connections []*pipelineservice.Connection) []pipelineservice.ConnectionHealthItem {
	// Batch-fetch last job info for all connections in a single query.
	connIDs := make([]uuid.UUID, len(connections))
	for i, c := range connections {
		connIDs[i] = c.ID
	}

	jobInfoMap, _ := uc.statsStorage.QueryLastJobInfoBatch(ctx, connIDs)
	if jobInfoMap == nil {
		jobInfoMap = make(map[uuid.UUID]*pipelineservice.LastJobInfo)
	}

	healthItems := make([]pipelineservice.ConnectionHealthItem, 0, len(connections))
	for _, conn := range connections {
		schedule := ""
		if conn.Schedule != nil {
			schedule = *conn.Schedule
		}

		var lastStatus string
		var lastSyncAt *time.Time

		if info := jobInfoMap[conn.ID]; info != nil {
			lastStatus = info.Status
			lastSyncAt = info.CompletedAt
		}

		health := pipelineservice.HealthDisabled
		if conn.Status == pipelineservice.ConnectionStatusActive {
			if lastSyncAt != nil {
				health = ComputeHealthStatus(lastStatus, schedule, *lastSyncAt, time.Now())
			} else if schedule != "" {
				health = pipelineservice.HealthHealthy // New connection, not yet synced.
			}
		}

		healthItems = append(healthItems, pipelineservice.ConnectionHealthItem{
			ConnectionID:   conn.ID,
			ConnectionName: conn.Name,
			Health:         health,
			LastSyncAt:     lastSyncAt,
		})
	}

	return healthItems
}

func (uc *GetWorkspaceStats) queryTopConnections(ctx context.Context, workspaceID uuid.UUID, from, to time.Time, connections []*pipelineservice.Connection) ([]pipelineservice.TopConnectionItem, error) {
	// Build connection name map.
	nameMap := make(map[uuid.UUID]string)
	for _, c := range connections {
		nameMap[c.ID] = c.Name
	}

	rows, err := uc.statsStorage.QueryTopConnections(ctx, workspaceID, from, to, 5)
	if err != nil {
		return nil, err
	}

	// Batch-fetch last completed times and sparklines for all top connections.
	topConnIDs := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		topConnIDs[i] = r.ConnectionID
	}

	lastCompletedMap, _ := uc.statsStorage.QueryConnectionLastCompletedAtBatch(ctx, topConnIDs)
	if lastCompletedMap == nil {
		lastCompletedMap = make(map[uuid.UUID]*time.Time)
	}

	sparklineMap, _ := uc.statsStorage.QueryConnectionSparklineBatch(ctx, topConnIDs, from, to, 7)
	if sparklineMap == nil {
		sparklineMap = make(map[uuid.UUID][]int64)
	}

	items := make([]pipelineservice.TopConnectionItem, len(rows))
	for i, r := range rows {
		sparkline := sparklineMap[r.ConnectionID]

		// Reverse to chronological order (batch query returns DESC order).
		for l, ri := 0, len(sparkline)-1; l < ri; l, ri = l+1, ri-1 {
			sparkline[l], sparkline[ri] = sparkline[ri], sparkline[l]
		}

		items[i] = pipelineservice.TopConnectionItem{
			ConnectionID:    r.ConnectionID,
			ConnectionName:  nameMap[r.ConnectionID],
			RecordsSynced:   r.RecordsSynced,
			BytesSynced:     r.BytesSynced,
			LastSyncAt:      lastCompletedMap[r.ConnectionID],
			SparklineValues: sparkline,
		}
	}

	return items, nil
}

// computeDelta computes percentage change from previous to current value.
func computeDelta(previous, current float64) float64 {
	if previous == 0 {
		if current == 0 {
			return 0
		}
		return 100
	}
	return ((current - previous) / previous) * 100
}
