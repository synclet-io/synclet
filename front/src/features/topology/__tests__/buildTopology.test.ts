import type { Connection } from '@entities/connection'
import type { Destination } from '@entities/destination'
import type { Source } from '@entities/source'
import type { ConnectionHealth } from '@entities/stats'
import { describe, expect, it } from 'vitest'
import { buildTopology } from '../buildTopology'

function mkSource(id: string, name: string): Source {
  return {
    id,
    workspaceId: 'ws',
    name,
    managedConnectorId: 'mc',
    config: {},
    runtimeConfig: null,
    createdAt: undefined,
    updatedAt: undefined,
  }
}

function mkDest(id: string, name: string): Destination {
  return {
    id,
    workspaceId: 'ws',
    name,
    managedConnectorId: 'mc',
    config: {},
    runtimeConfig: null,
    createdAt: undefined,
    updatedAt: undefined,
  }
}

function mkConn(id: string, name: string, srcId: string, destId: string, status: Connection['status'] = 'active'): Connection {
  return {
    id,
    workspaceId: 'ws',
    name,
    status,
    sourceId: srcId,
    destinationId: destId,
    schedule: '',
    schemaChangePolicy: 'propagate',
    createdAt: undefined,
    updatedAt: undefined,
    maxAttempts: 3,
    namespaceDefinition: 'source',
    customNamespaceFormat: '',
    streamPrefix: '',
  }
}

function mkHealth(connectionId: string, health: ConnectionHealth['health']): ConnectionHealth {
  return { connectionId, connectionName: connectionId, health, lastSyncAt: undefined }
}

describe('buildTopology', () => {
  it('produces a source, connection, destination triple with two edges per connection', () => {
    const topo = buildTopology({
      sources: [mkSource('s1', 'Postgres')],
      destinations: [mkDest('d1', 'BigQuery')],
      connections: [mkConn('c1', 'pg→bq', 's1', 'd1')],
      healths: [mkHealth('c1', 'healthy')],
    })
    expect(topo.sources).toHaveLength(1)
    expect(topo.destinations).toHaveLength(1)
    expect(topo.connections).toHaveLength(1)
    expect(topo.edges).toEqual([
      { from: 's1', to: 'c1', connectionId: 'c1' },
      { from: 'c1', to: 'd1', connectionId: 'c1' },
    ])
  })

  it('aggregates source/destination health from connected connections (worst wins)', () => {
    const topo = buildTopology({
      sources: [mkSource('s1', 's')],
      destinations: [mkDest('d1', 'd')],
      connections: [
        mkConn('c1', 'a', 's1', 'd1'),
        mkConn('c2', 'b', 's1', 'd1'),
      ],
      healths: [
        mkHealth('c1', 'healthy'),
        mkHealth('c2', 'failing'),
      ],
    })
    expect(topo.sources[0].health).toBe('failing')
    expect(topo.destinations[0].health).toBe('failing')
  })

  it('marks connection health "disabled" when status is paused/inactive', () => {
    const topo = buildTopology({
      sources: [mkSource('s1', 's')],
      destinations: [mkDest('d1', 'd')],
      connections: [mkConn('c1', 'a', 's1', 'd1', 'inactive')],
      healths: [mkHealth('c1', 'healthy')],
    })
    expect(topo.connections[0].health).toBe('disabled')
  })

  it('marks "never" when no health record exists', () => {
    const topo = buildTopology({
      sources: [mkSource('s1', 's')],
      destinations: [mkDest('d1', 'd')],
      connections: [mkConn('c1', 'a', 's1', 'd1')],
      healths: [],
    })
    expect(topo.connections[0].health).toBe('never')
  })

  it('filters to failing-only and prunes unreferenced sources/destinations', () => {
    const topo = buildTopology({
      sources: [mkSource('s1', 'kept'), mkSource('s2', 'dropped')],
      destinations: [mkDest('d1', 'kept'), mkDest('d2', 'dropped')],
      connections: [
        mkConn('c1', 'failing', 's1', 'd1'),
        mkConn('c2', 'healthy', 's2', 'd2'),
      ],
      healths: [
        mkHealth('c1', 'failing'),
        mkHealth('c2', 'healthy'),
      ],
      filters: { failingOnly: true, enabledOnly: false },
    })
    expect(topo.connections.map(c => c.id)).toEqual(['c1'])
    expect(topo.sources.map(s => s.id)).toEqual(['s1'])
    expect(topo.destinations.map(d => d.id)).toEqual(['d1'])
  })

  it('filters by sourceId to narrow the graph to one source', () => {
    const topo = buildTopology({
      sources: [mkSource('s1', 'a'), mkSource('s2', 'b')],
      destinations: [mkDest('d1', 'x'), mkDest('d2', 'y')],
      connections: [
        mkConn('c1', 'k', 's1', 'd1'),
        mkConn('c2', 'k', 's2', 'd2'),
      ],
      healths: [],
      filters: { failingOnly: false, enabledOnly: false, sourceId: 's1' },
    })
    expect(topo.connections.map(c => c.id)).toEqual(['c1'])
    expect(topo.destinations.map(d => d.id)).toEqual(['d1'])
  })

  it('filters by enabledOnly drops inactive connections', () => {
    const topo = buildTopology({
      sources: [mkSource('s1', 's')],
      destinations: [mkDest('d1', 'd')],
      connections: [
        mkConn('c1', 'active', 's1', 'd1', 'active'),
        mkConn('c2', 'paused', 's1', 'd1', 'paused'),
      ],
      healths: [],
      filters: { failingOnly: false, enabledOnly: true },
    })
    expect(topo.connections.map(c => c.id)).toEqual(['c1'])
  })

  it('orphaned sources/destinations show up when no filters are active', () => {
    const topo = buildTopology({
      sources: [mkSource('s1', 'orphan'), mkSource('s2', 'used')],
      destinations: [mkDest('d1', 'used')],
      connections: [mkConn('c1', 'k', 's2', 'd1')],
      healths: [],
    })
    expect(topo.sources.map(s => s.id).sort()).toEqual(['s1', 's2'])
  })
})
