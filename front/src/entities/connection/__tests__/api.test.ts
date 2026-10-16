import { beforeEach, describe, expect, it, vi } from 'vitest'
import { listConnections } from '../api'

const connectionClient = vi.hoisted(() => ({ listConnections: vi.fn() }))

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

    expect(connectionClient.listConnections).toHaveBeenCalledWith({ pageSize: 20, offset: 0, includePendingSchemaChanges: true, sourceId: '', destinationId: '' })
    expect(items[0]).toMatchObject({ status: 'unknown', pendingSchemaChanges: 3 })
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

    expect(connectionClient.listConnections).toHaveBeenCalledWith({ pageSize: 0, offset: 0, includePendingSchemaChanges: false, sourceId: 's1', destinationId: 'd1' })
    expect(items[0]).toMatchObject({
      sourceName: 'Orders DB',
      destinationName: 'Warehouse',
      sourceConnectorName: 'Postgres',
      destinationConnectorName: 'BigQuery',
    })
  })
})
