import type {
  ConnectionHealth,
  ConnectionStats,
  FailureCategory,
  FailureCategoryName,
  HealthStatus,
  RecordsTimelinePoint,
  SyncDurationPoint,
  SyncStatus,
  SyncTimeline,
  TimeRange,
  TopConnection,
  WorkspaceStats,
} from './types'
import { createClient } from '@connectrpc/connect'
import { transport } from '@shared/api/transport'
import { tsToDate } from '@shared/lib/formatting'
import type { ConnectionHealth as ProtoConnectionHealth, TopConnection as ProtoTopConnection } from '@/gen/synclet/publicapi/stats/v1/stats_pb'
import { FailureCategoryType as ProtoFailureCategoryType, HealthStatus as ProtoHealthStatus, SyncStatus as ProtoSyncStatus, TimeRange as ProtoTimeRange, StatsService } from '@/gen/synclet/publicapi/stats/v1/stats_pb'

const statsClient = createClient(StatsService, transport)

function timeRangeToProto(tr: TimeRange): ProtoTimeRange {
  switch (tr) {
    case '24h': return ProtoTimeRange.TIME_RANGE_24H
    case '7d': return ProtoTimeRange.TIME_RANGE_7D
    case '30d': return ProtoTimeRange.TIME_RANGE_30D
    default: return ProtoTimeRange.TIME_RANGE_24H
  }
}

function mapHealthStatus(s: ProtoHealthStatus): HealthStatus {
  switch (s) {
    case ProtoHealthStatus.HEALTHY: return 'healthy'
    case ProtoHealthStatus.WARNING: return 'warning'
    case ProtoHealthStatus.FAILING: return 'failing'
    case ProtoHealthStatus.DISABLED: return 'disabled'
    default: return 'disabled'
  }
}

function mapSyncStatus(s: ProtoSyncStatus): SyncStatus {
  switch (s) {
    case ProtoSyncStatus.COMPLETED: return 'completed'
    case ProtoSyncStatus.FAILED: return 'failed'
    default: return 'unknown'
  }
}

function mapFailureCategory(s: ProtoFailureCategoryType): FailureCategoryName {
  switch (s) {
    case ProtoFailureCategoryType.TIMEOUT: return 'timeout'
    case ProtoFailureCategoryType.OOM: return 'oom'
    case ProtoFailureCategoryType.CONNECTOR: return 'connector'
    case ProtoFailureCategoryType.INFRASTRUCTURE: return 'infrastructure'
    case ProtoFailureCategoryType.UNKNOWN: return 'unknown'
    default: return 'unknown'
  }
}

function mapConnectionHealth(proto: ProtoConnectionHealth): ConnectionHealth {
  return {
    connectionId: proto.connectionId,
    connectionName: proto.connectionName,
    health: mapHealthStatus(proto.health),
    lastSyncAt: tsToDate(proto.lastSyncAt),
  }
}

function mapTopConnection(proto: ProtoTopConnection): TopConnection {
  return {
    connectionId: proto.connectionId,
    connectionName: proto.connectionName,
    recordsSynced: Number(proto.recordsSynced),
    bytesSynced: Number(proto.bytesSynced),
    lastSyncAt: tsToDate(proto.lastSyncAt),
    sparklineValues: (proto.sparklineValues || []).map(Number),
  }
}

export async function getWorkspaceStats(timeRange: TimeRange): Promise<WorkspaceStats> {
  const res = await statsClient.getWorkspaceStats({ timeRange: timeRangeToProto(timeRange) })
  return {
    totalSyncs: Number(res.totalSyncs),
    successRate: res.successRate,
    recordsSynced: Number(res.recordsSynced),
    activeConnections: res.activeConnections,
    failedSyncs: Number(res.failedSyncs),
    totalSyncsDelta: res.totalSyncsDelta,
    successRateDelta: res.successRateDelta,
    recordsSyncedDelta: res.recordsSyncedDelta,
    failedSyncsDelta: res.failedSyncsDelta,
    connectionHealths: (res.connectionHealths || []).map(mapConnectionHealth),
    topConnections: (res.topConnections || []).map(mapTopConnection),
    failureBreakdown: (res.failureBreakdown || []).map((f): FailureCategory => ({
      category: mapFailureCategory(f.category),
      count: f.count,
    })),
  }
}

export async function getConnectionStats(connectionId: string, timeRange: TimeRange): Promise<ConnectionStats> {
  const res = await statsClient.getConnectionStats({ connectionId, timeRange: timeRangeToProto(timeRange) })
  return {
    avgDurationMs: Number(res.avgDurationMs),
    successRate: res.successRate,
    totalRecords: Number(res.totalRecords),
    lastSyncAt: tsToDate(res.lastSyncAt),
    avgDurationDelta: res.avgDurationDelta,
    successRateDelta: res.successRateDelta,
    totalRecordsDelta: res.totalRecordsDelta,
    health: mapHealthStatus(res.health),
    durationChart: (res.durationChart || []).map((p): SyncDurationPoint => ({
      label: p.label,
      durationMs: Number(p.durationMs),
      status: mapSyncStatus(p.status),
    })),
    recordsChart: (res.recordsChart || []).map((p): RecordsTimelinePoint => ({
      label: p.label,
      recordsRead: Number(p.recordsRead),
    })),
    failureBreakdown: (res.failureBreakdown || []).map((f): FailureCategory => ({
      category: mapFailureCategory(f.category),
      count: f.count,
    })),
  }
}

export async function getSyncTimeline(timeRange: TimeRange, connectionId?: string): Promise<SyncTimeline> {
  const res = await statsClient.getSyncTimeline({ timeRange: timeRangeToProto(timeRange), connectionId: connectionId ?? '' })
  return {
    points: (res.points || []).map(p => ({
      label: p.label,
      succeeded: p.succeeded,
      failed: p.failed,
    })),
    throughput: (res.throughput || []).map(p => ({
      label: p.label,
      recordsRead: Number(p.recordsRead),
      bytesSynced: Number(p.bytesSynced),
    })),
    durations: (res.durations || []).map(p => ({
      label: p.label,
      avgDurationMs: Number(p.avgDurationMs),
    })),
  }
}
