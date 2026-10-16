import { Code, ConnectError } from '@connectrpc/connect'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getSourceCatalog } from '../api'

const sourceClient = vi.hoisted(() => ({ getSourceCatalog: vi.fn() }))

vi.mock('@shared/api/services', () => ({ sourceClient }))

describe('getSourceCatalog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns null when the source was never discovered', async () => {
    sourceClient.getSourceCatalog.mockRejectedValue(new ConnectError('no catalog', Code.NotFound))
    await expect(getSourceCatalog('s1')).resolves.toBeNull()
  })

  it('rethrows every other error', async () => {
    const err = new ConnectError('denied', Code.PermissionDenied)
    sourceClient.getSourceCatalog.mockRejectedValue(err)
    await expect(getSourceCatalog('s1')).rejects.toBe(err)
  })
})
