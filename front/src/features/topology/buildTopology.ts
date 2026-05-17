import type { Connection } from '@entities/connection'
import type { Destination } from '@entities/destination'
import type { Source } from '@entities/source'
import type { ConnectionHealth, HealthStatus } from '@entities/stats'
import type { TopologyData, TopologyFilters, TopologyNode } from './types'

export interface BuildTopologyInput {
  sources: Source[]
  destinations: Destination[]
  connections: Connection[]
  healths: ConnectionHealth[]
  filters?: TopologyFilters
}

function healthFromConnection(conn: Connection, healthByConn: Map<string, ConnectionHealth>): TopologyNode['health'] {
  if (conn.status !== 'active')
    return 'disabled'
  const h = healthByConn.get(conn.id)
  if (!h)
    return 'never'
  return h.health
}

function aggregateHealth(healths: TopologyNode['health'][]): TopologyNode['health'] {
  if (healths.length === 0)
    return 'never'
  // Worst wins: failing > warning > disabled > healthy > never.
  const order: Record<TopologyNode['health'], number> = {
    failing: 4,
    warning: 3,
    disabled: 2,
    healthy: 1,
    never: 0,
  }
  let best: TopologyNode['health'] = 'never'
  for (const h of healths) {
    if (order[h] > order[best])
      best = h
  }
  return best
}

export function buildTopology(input: BuildTopologyInput): TopologyData {
  const { sources, destinations, connections } = input
  const filters = input.filters ?? { failingOnly: false, enabledOnly: false }

  const healthByConn = new Map<string, ConnectionHealth>()
  for (const h of input.healths)
    healthByConn.set(h.connectionId, h)

  let filteredConnections = connections
  if (filters.sourceId)
    filteredConnections = filteredConnections.filter(c => c.sourceId === filters.sourceId)
  if (filters.destinationId)
    filteredConnections = filteredConnections.filter(c => c.destinationId === filters.destinationId)
  if (filters.enabledOnly)
    filteredConnections = filteredConnections.filter(c => c.status === 'active')
  if (filters.failingOnly) {
    filteredConnections = filteredConnections.filter((c) => {
      const h = healthFromConnection(c, healthByConn)
      return h === 'failing' || h === 'warning'
    })
  }

  const connectionNodes: TopologyNode[] = filteredConnections.map(c => ({
    id: c.id,
    kind: 'connection' as const,
    name: c.name,
    health: healthFromConnection(c, healthByConn),
    enabled: c.status === 'active',
    lastSyncAt: healthByConn.get(c.id)?.lastSyncAt,
  }))

  const connectionsBySource = new Map<string, TopologyNode[]>()
  const connectionsByDest = new Map<string, TopologyNode[]>()
  for (let i = 0; i < filteredConnections.length; i++) {
    const conn = filteredConnections[i]
    const node = connectionNodes[i]
    if (!connectionsBySource.has(conn.sourceId))
      connectionsBySource.set(conn.sourceId, [])
    connectionsBySource.get(conn.sourceId)!.push(node)
    if (!connectionsByDest.has(conn.destinationId))
      connectionsByDest.set(conn.destinationId, [])
    connectionsByDest.get(conn.destinationId)!.push(node)
  }

  const referencedSourceIds = new Set(filteredConnections.map(c => c.sourceId))
  const referencedDestIds = new Set(filteredConnections.map(c => c.destinationId))

  const sourceNodes: TopologyNode[] = sources
    .filter(s => filters.failingOnly || filters.enabledOnly || filters.sourceId || filters.destinationId
      ? referencedSourceIds.has(s.id)
      : true)
    .map(s => ({
      id: s.id,
      kind: 'source' as const,
      name: s.name,
      health: aggregateHealth((connectionsBySource.get(s.id) ?? []).map(n => n.health)),
      enabled: true,
    }))

  const destNodes: TopologyNode[] = destinations
    .filter(d => filters.failingOnly || filters.enabledOnly || filters.sourceId || filters.destinationId
      ? referencedDestIds.has(d.id)
      : true)
    .map(d => ({
      id: d.id,
      kind: 'destination' as const,
      name: d.name,
      health: aggregateHealth((connectionsByDest.get(d.id) ?? []).map(n => n.health)),
      enabled: true,
    }))

  const edges = filteredConnections.flatMap(c => [
    { from: c.sourceId, to: c.id, connectionId: c.id },
    { from: c.id, to: c.destinationId, connectionId: c.id },
  ])

  return {
    sources: sourceNodes,
    connections: connectionNodes,
    destinations: destNodes,
    edges,
  }
}

export function nodeHealthColor(h: TopologyNode['health']): { fill: string, stroke: string, text: string } {
  switch (h) {
    case 'healthy':
      return { fill: 'fill-green-500/15', stroke: 'stroke-green-500', text: 'text-green-700 dark:text-green-300' }
    case 'warning':
      return { fill: 'fill-amber-500/15', stroke: 'stroke-amber-500', text: 'text-amber-700 dark:text-amber-300' }
    case 'failing':
      return { fill: 'fill-red-500/15', stroke: 'stroke-red-500', text: 'text-red-700 dark:text-red-300' }
    case 'disabled':
      return { fill: 'fill-slate-400/15', stroke: 'stroke-slate-400', text: 'text-text-secondary' }
    case 'never':
    default:
      return { fill: 'fill-slate-300/15', stroke: 'stroke-slate-300', text: 'text-text-muted' }
  }
}

export function isFailing(h: HealthStatus | 'never'): boolean {
  return h === 'failing' || h === 'warning'
}
