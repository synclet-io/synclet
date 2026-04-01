package pipelineconnect

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	statsv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/stats/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/stats/v1/statsv1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestats"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// timeRangeToDomain converts a proto TimeRange enum to the domain TimeRange type.
func timeRangeToDomain(tr statsv1.TimeRange) pipelineservice.TimeRange {
	switch tr {
	case statsv1.TimeRange_TIME_RANGE_24H:
		return pipelineservice.TimeRange24h
	case statsv1.TimeRange_TIME_RANGE_7D:
		return pipelineservice.TimeRange7d
	case statsv1.TimeRange_TIME_RANGE_30D:
		return pipelineservice.TimeRange30d
	default:
		return pipelineservice.TimeRange24h
	}
}

// healthStatusToProto converts a domain Health enum to a proto HealthStatus enum.
func healthStatusToProto(h pipelineservice.Health) statsv1.HealthStatus {
	switch h {
	case pipelineservice.HealthHealthy:
		return statsv1.HealthStatus_HEALTH_STATUS_HEALTHY
	case pipelineservice.HealthWarning:
		return statsv1.HealthStatus_HEALTH_STATUS_WARNING
	case pipelineservice.HealthFailing:
		return statsv1.HealthStatus_HEALTH_STATUS_FAILING
	case pipelineservice.HealthDisabled:
		return statsv1.HealthStatus_HEALTH_STATUS_DISABLED
	default:
		return statsv1.HealthStatus_HEALTH_STATUS_UNSPECIFIED
	}
}

// syncStatusToProto converts a domain SyncStatus enum to a proto SyncStatus enum.
func syncStatusToProto(s pipelineservice.SyncStatus) statsv1.SyncStatus {
	switch s {
	case pipelineservice.SyncStatusCompleted:
		return statsv1.SyncStatus_SYNC_STATUS_COMPLETED
	case pipelineservice.SyncStatusFailed:
		return statsv1.SyncStatus_SYNC_STATUS_FAILED
	default:
		return statsv1.SyncStatus_SYNC_STATUS_UNSPECIFIED
	}
}

// failureCategoryToProto converts a domain FailureCategory enum to a proto enum.
func failureCategoryToProto(c pipelineservice.FailureCategory) statsv1.FailureCategoryType {
	switch c {
	case pipelineservice.FailureCategoryTimeout:
		return statsv1.FailureCategoryType_FAILURE_CATEGORY_TYPE_TIMEOUT
	case pipelineservice.FailureCategoryOOM:
		return statsv1.FailureCategoryType_FAILURE_CATEGORY_TYPE_OOM
	case pipelineservice.FailureCategoryConnector:
		return statsv1.FailureCategoryType_FAILURE_CATEGORY_TYPE_CONNECTOR
	case pipelineservice.FailureCategoryInfrastructure:
		return statsv1.FailureCategoryType_FAILURE_CATEGORY_TYPE_INFRASTRUCTURE
	case pipelineservice.FailureCategoryUnknown:
		return statsv1.FailureCategoryType_FAILURE_CATEGORY_TYPE_UNKNOWN
	default:
		return statsv1.FailureCategoryType_FAILURE_CATEGORY_TYPE_UNSPECIFIED
	}
}

// StatsHandler implements the StatsService ConnectRPC handler.
type StatsHandler struct {
	statsv1connect.UnimplementedStatsServiceHandler

	getWorkspaceStats  *pipelinestats.GetWorkspaceStats
	getConnectionStats *pipelinestats.GetConnectionStats
	getSyncTimeline    *pipelinestats.GetSyncTimeline
}

// NewStatsHandler creates a new stats handler.
func NewStatsHandler(
	getWorkspaceStats *pipelinestats.GetWorkspaceStats,
	getConnectionStats *pipelinestats.GetConnectionStats,
	getSyncTimeline *pipelinestats.GetSyncTimeline,
) *StatsHandler {
	return &StatsHandler{
		getWorkspaceStats:  getWorkspaceStats,
		getConnectionStats: getConnectionStats,
		getSyncTimeline:    getSyncTimeline,
	}
}

func (h *StatsHandler) GetWorkspaceStats(ctx context.Context, req *connect.Request[statsv1.GetWorkspaceStatsRequest]) (*connect.Response[statsv1.GetWorkspaceStatsResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	result, err := h.getWorkspaceStats.Execute(ctx, pipelineservice.GetWorkspaceStatsParams{
		WorkspaceID: workspaceID,
		TimeRange:   timeRangeToDomain(req.Msg.GetTimeRange()),
	})
	if err != nil {
		return nil, mapError(err)
	}

	// Map connection healths.
	healths := make([]*statsv1.ConnectionHealth, len(result.ConnectionHealths))
	for i, ch := range result.ConnectionHealths {
		healths[i] = &statsv1.ConnectionHealth{
			ConnectionId:   ch.ConnectionID.String(),
			ConnectionName: ch.ConnectionName,
			Health:         healthStatusToProto(ch.Health),
		}
		if ch.LastSyncAt != nil {
			healths[i].LastSyncAt = timestamppb.New(*ch.LastSyncAt)
		}
	}

	// Map top connections.
	topConns := make([]*statsv1.TopConnection, len(result.TopConnections))
	for i, topConn := range result.TopConnections {
		topConns[i] = &statsv1.TopConnection{
			ConnectionId:    topConn.ConnectionID.String(),
			ConnectionName:  topConn.ConnectionName,
			RecordsSynced:   topConn.RecordsSynced,
			BytesSynced:     topConn.BytesSynced,
			SparklineValues: topConn.SparklineValues,
		}
		if topConn.LastSyncAt != nil {
			topConns[i].LastSyncAt = timestamppb.New(*topConn.LastSyncAt)
		}
	}

	// Map failure breakdown.
	failures := make([]*statsv1.FailureCategory, len(result.FailureBreakdown))
	for i, fb := range result.FailureBreakdown {
		failures[i] = &statsv1.FailureCategory{
			Category: failureCategoryToProto(fb.Category),
			Count:    fb.Count,
		}
	}

	return connect.NewResponse(&statsv1.GetWorkspaceStatsResponse{
		TotalSyncs:         result.TotalSyncs,
		SuccessRate:        result.SuccessRate,
		RecordsSynced:      result.RecordsSynced,
		ActiveConnections:  result.ActiveConnections,
		FailedSyncs:        result.FailedSyncs,
		TotalSyncsDelta:    result.TotalSyncsDelta,
		SuccessRateDelta:   result.SuccessRateDelta,
		RecordsSyncedDelta: result.RecordsSyncedDelta,
		FailedSyncsDelta:   result.FailedSyncsDelta,
		ConnectionHealths:  healths,
		TopConnections:     topConns,
		FailureBreakdown:   failures,
	}), nil
}

func (h *StatsHandler) GetConnectionStats(ctx context.Context, req *connect.Request[statsv1.GetConnectionStatsRequest]) (*connect.Response[statsv1.GetConnectionStatsResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	connectionID, err := uuid.Parse(req.Msg.GetConnectionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connection_id: %w", err))
	}

	result, err := h.getConnectionStats.Execute(ctx, pipelineservice.GetConnectionStatsParams{
		ConnectionID: connectionID,
		WorkspaceID:  workspaceID,
		TimeRange:    timeRangeToDomain(req.Msg.GetTimeRange()),
	})
	if err != nil {
		return nil, mapError(err)
	}

	// Map duration chart.
	durChart := make([]*statsv1.SyncDurationPoint, len(result.DurationChart))
	for i, dp := range result.DurationChart {
		durChart[i] = &statsv1.SyncDurationPoint{
			Label:      dp.Label,
			DurationMs: dp.DurationMs,
			Status:     syncStatusToProto(dp.Status),
		}
	}

	// Map records chart.
	recChart := make([]*statsv1.RecordsTimelinePoint, len(result.RecordsChart))
	for i, rp := range result.RecordsChart {
		recChart[i] = &statsv1.RecordsTimelinePoint{
			Label:       rp.Label,
			RecordsRead: rp.RecordsRead,
		}
	}

	// Map failure breakdown.
	failures := make([]*statsv1.FailureCategory, len(result.FailureBreakdown))
	for i, fb := range result.FailureBreakdown {
		failures[i] = &statsv1.FailureCategory{
			Category: failureCategoryToProto(fb.Category),
			Count:    fb.Count,
		}
	}

	resp := &statsv1.GetConnectionStatsResponse{
		AvgDurationMs:     result.AvgDurationMs,
		SuccessRate:       result.SuccessRate,
		TotalRecords:      result.TotalRecords,
		AvgDurationDelta:  result.AvgDurationDelta,
		SuccessRateDelta:  result.SuccessRateDelta,
		TotalRecordsDelta: result.TotalRecordsDelta,
		Health:            healthStatusToProto(result.Health),
		DurationChart:     durChart,
		RecordsChart:      recChart,
		FailureBreakdown:  failures,
	}
	if result.LastSyncAt != nil {
		resp.LastSyncAt = timestamppb.New(*result.LastSyncAt)
	}

	return connect.NewResponse(resp), nil
}

func (h *StatsHandler) GetSyncTimeline(ctx context.Context, req *connect.Request[statsv1.GetSyncTimelineRequest]) (*connect.Response[statsv1.GetSyncTimelineResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	params := pipelineservice.GetSyncTimelineParams{
		WorkspaceID: workspaceID,
		TimeRange:   timeRangeToDomain(req.Msg.GetTimeRange()),
	}
	if req.Msg.GetConnectionId() != "" {
		connID, err := uuid.Parse(req.Msg.GetConnectionId())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connection_id: %w", err))
		}

		params.ConnectionID = &connID
	}

	result, err := h.getSyncTimeline.Execute(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	points := make([]*statsv1.TimelinePoint, len(result.Points))
	for i, p := range result.Points {
		points[i] = &statsv1.TimelinePoint{
			Label:     p.Label,
			Succeeded: p.Succeeded,
			Failed:    p.Failed,
		}
	}

	throughput := make([]*statsv1.ThroughputPoint, len(result.Throughput))
	for i, tp := range result.Throughput {
		throughput[i] = &statsv1.ThroughputPoint{
			Label:       tp.Label,
			RecordsRead: tp.RecordsRead,
			BytesSynced: tp.BytesSynced,
		}
	}

	durations := make([]*statsv1.DurationPoint, len(result.Durations))
	for i, dp := range result.Durations {
		durations[i] = &statsv1.DurationPoint{
			Label:         dp.Label,
			AvgDurationMs: dp.AvgDurationMs,
		}
	}

	return connect.NewResponse(&statsv1.GetSyncTimelineResponse{
		Points:     points,
		Throughput: throughput,
		Durations:  durations,
	}), nil
}
