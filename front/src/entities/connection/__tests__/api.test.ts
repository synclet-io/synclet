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

    expect(connectionClient.listConnections).toHaveBeenCalledWith({ pageSize: 20, offset: 0, includePendingSchemaChanges: true })
    expect(items[0]).toMatchObject({ status: 'unknown', pendingSchemaChanges: 3 })
  })
})
