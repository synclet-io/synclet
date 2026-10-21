import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createConnection, listConnections, listConnectionTags, updateConnection } from '../api'

const connectionClient = vi.hoisted(() => ({
  listConnections: vi.fn(),
  updateConnection: vi.fn(),
  createConnection: vi.fn(),
  listConnectionTags: vi.fn(),
}))

vi.mock('@shared/api/services', async importOriginal => ({
  ...(await importOriginal<typeof import('@shared/api/services')>()),
  connectionClient,
}))

describe('listConnections', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('maps an unrecognised status to unknown and carries pending schema changes', async () => {
    connectionClient.listConnections.mockResolvedValue({
      connections: [{ id: 'c1', status: 99, pendingSchemaChanges: 3 }],
      total: 1,
    })

    const { items } = await listConnections({ pageSize: 20, includePendingSchemaChanges: true })

    expect(connectionClient.listConnections).toHaveBeenCalledWith({ pageSize: 20, offset: 0, includePendingSchemaChanges: true, sourceId: '', destinationId: '', tags: [] })
    expect(items[0]).toMatchObject({ status: 'unknown', pendingSchemaChanges: 3 })
  })

  it('carries whether streams are configured', async () => {
    connectionClient.listConnections.mockResolvedValue({
      connections: [{ id: 'c1', streamsConfigured: true }, { id: 'c2' }],
      total: 2,
    })

    const { items } = await listConnections({ pageSize: 20 })

    expect(items.map(c => c.streamsConfigured)).toEqual([true, false])
  })

  it('forwards source and destination filters and maps endpoint names', async () => {
    connectionClient.listConnections.mockResolvedValue({
      connections: [{
        id: 'c1',
        sourceId: 's1',
        destinationId: 'd1',
        sourceName: 'Orders DB',
        destinationName: 'Warehouse',
        sourceConnectorName: 'Postgres',
        destinationConnectorName: 'BigQuery',
      }],
      total: 1,
    })

    const { items } = await listConnections({ sourceId: 's1', destinationId: 'd1' })

    expect(connectionClient.listConnections).toHaveBeenCalledWith({ pageSize: 0, offset: 0, includePendingSchemaChanges: false, sourceId: 's1', destinationId: 'd1', tags: [] })
    expect(items[0]).toMatchObject({
      sourceName: 'Orders DB',
      destinationName: 'Warehouse',
      sourceConnectorName: 'Postgres',
      destinationConnectorName: 'BigQuery',
    })
  })
})

describe('connection tags', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('maps tags and forwards the tag filter', async () => {
    connectionClient.listConnections.mockResolvedValue({ connections: [{ id: 'c1', tags: ['eu', 'prod'] }], total: 1 })

    const { items } = await listConnections({ tags: ['prod'] })

    expect(connectionClient.listConnections).toHaveBeenCalledWith(expect.objectContaining({ tags: ['prod'] }))
    expect(items[0]!.tags).toEqual(['eu', 'prod'])
  })

  it('update sends tags as a TagList and leaves unset tags undefined', async () => {
    connectionClient.updateConnection.mockResolvedValue({})

    await updateConnection({ id: 'c1', tags: [] })
    expect(connectionClient.updateConnection.mock.calls[0]![0].tags).toEqual({ values: [] })

    await updateConnection({ id: 'c1', tags: ['prod'] })
    expect(connectionClient.updateConnection.mock.calls[1]![0].tags).toEqual({ values: ['prod'] })

    await updateConnection({ id: 'c1', name: 'x' })
    expect(connectionClient.updateConnection.mock.calls[2]![0].tags).toBeUndefined()
  })

  it('create sends tags', async () => {
    connectionClient.createConnection.mockResolvedValue({})

    await createConnection({ name: 'n', sourceId: 's', destinationId: 'd', schemaChangePolicy: 'propagate', maxAttempts: 3, tags: ['prod'] })

    expect(connectionClient.createConnection.mock.calls[0]![0].tags).toEqual(['prod'])
  })

  it('lists workspace tags with counts', async () => {
    connectionClient.listConnectionTags.mockResolvedValue({ tags: [{ tag: 'prod', connectionCount: 2 }] })

    expect(await listConnectionTags()).toEqual([{ tag: 'prod', connectionCount: 2 }])
  })
})
