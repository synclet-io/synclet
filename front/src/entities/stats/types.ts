export type TimeRange = '24h' | '7d' | '30d'
export type HealthStatus = 'healthy' | 'warning' | 'failing' | 'disabled'
export type SyncStatus = 'completed' | 'failed' | 'unknown'
export type FailureCategoryName = 'timeout' | 'oom' | 'connector' | 'infrastructure' | 'unknown'

export interface WorkspaceStats {
  totalSyncs: number
  successRate: number
  recordsSynced: number
  activeConnections: number
  failedSyncs: number
  totalSyncsDelta: number
  successRateDelta: number
  recordsSyncedDelta: number
  failedSyncsDelta: number
  connectionHealths: ConnectionHealth[]
  topConnections: TopConnection[]
  failureBreakdown: FailureCategory[]
}

export interface ConnectionHealth {
  connectionId: string
  connectionName: string
  health: HealthStatus
  lastSyncAt: Date | undefined
}

export interface TopConnection {
  connectionId: string
  connectionName: string
  recordsSynced: number
  bytesSynced: number
  lastSyncAt: Date | undefined
  sparklineValues: number[]
}

export interface ConnectionStats {
  avgDurationMs: number
  successRate: number
  totalRecords: number
  lastSyncAt: Date | undefined
  avgDurationDelta: number
  successRateDelta: number
  totalRecordsDelta: number
  health: HealthStatus
  durationChart: SyncDurationPoint[]
  recordsChart: RecordsTimelinePoint[]
  failureBreakdown: FailureCategory[]
}

export interface SyncDurationPoint {
  label: string
  durationMs: number
  status: SyncStatus
}

export interface RecordsTimelinePoint {
  label: string
  recordsRead: number
}

export interface FailureCategory {
  category: FailureCategoryName
  count: number
}

export interface SyncTimeline {
  points: TimelinePoint[]
  throughput: ThroughputPoint[]
  durations: DurationPoint[]
}

export interface TimelinePoint {
  label: string
  succeeded: number
  failed: number
}

export interface ThroughputPoint {
  label: string
  recordsRead: number
  bytesSynced: number
}

export interface DurationPoint {
  label: string
  avgDurationMs: number
}
