import type { HealthStatus } from '@entities/stats'

export type NodeKind = 'source' | 'connection' | 'destination'

export interface TopologyNode {
  id: string
  kind: NodeKind
  name: string
  health: HealthStatus | 'never'
  enabled: boolean
  lastSyncAt?: Date
}

export interface TopologyEdge {
  from: string
  to: string
  connectionId: string
}

export interface TopologyData {
  sources: TopologyNode[]
  connections: TopologyNode[]
  destinations: TopologyNode[]
  edges: TopologyEdge[]
}

export interface TopologyFilters {
  failingOnly: boolean
  enabledOnly: boolean
  sourceId?: string
  destinationId?: string
}
