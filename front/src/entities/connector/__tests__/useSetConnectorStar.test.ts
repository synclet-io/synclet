import type { ManagedConnector } from '../types'
import { connectorKeys } from '@shared/lib/queryKeys'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { useConnectorStarPending, useSetConnectorStar } from '../composables'

const api = vi.hoisted(() => ({
  starManagedConnector: vi.fn(),
  unstarManagedConnector: vi.fn(),
  listManagedConnectors: vi.fn(),
}))

vi.mock('@entities/auth', () => ({ useAuth: () => ({ currentWorkspaceId: ref('ws-1') }) }))
vi.mock('../api', () => api)

const KEY = connectorKeys.managed('ws-1')

function setup(starred: boolean) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } })
  queryClient.setQueryData<ManagedConnector[]>(KEY, [
    { id: 'c1', name: 'Faker', dockerImage: 'x', dockerTag: '1', connectorType: 'source', starred, used: false },
    { id: 'c2', name: 'Postgres', dockerImage: 'y', dockerTag: '1', connectorType: 'source', starred, used: false },
  ])
  let mutation!: ReturnType<typeof useSetConnectorStar>
  let pending!: Record<'c1' | 'c2', ReturnType<typeof useConnectorStarPending>>
  mount(defineComponent({
    setup() {
      mutation = useSetConnectorStar()
      pending = { c1: useConnectorStarPending('c1'), c2: useConnectorStarPending('c2') }
      return () => null
    },
  }), { global: { plugins: [[VueQueryPlugin, { queryClient }]] } })
  const starredInCache = (id = 'c1') => queryClient.getQueryData<ManagedConnector[]>(KEY)!.find(c => c.id === id)!.starred
  return { mutation, starredInCache, pending }
}

describe('useSetConnectorStar', () => {
  it('stars the connector in the cached lists before the call returns', async () => {
    api.starManagedConnector.mockReturnValue(new Promise<void>(() => {}))
    const { mutation, starredInCache } = setup(false)

    mutation.mutate({ id: 'c1', starred: true })
    await flushPromises()
    expect(api.starManagedConnector).toHaveBeenCalledWith('c1')
    expect(starredInCache()).toBe(true)
  })

  it('rolls the star back when the call fails', async () => {
    api.unstarManagedConnector.mockRejectedValue(new Error('permission denied'))
    const { mutation, starredInCache } = setup(true)

    mutation.mutate({ id: 'c1', starred: false })
    await flushPromises()
    expect(api.unstarManagedConnector).toHaveBeenCalledWith('c1')
    expect(starredInCache()).toBe(true)
  })

  it('leaves both connectors unstarred when two overlapping stars fail', async () => {
    const rejects: Array<(err: Error) => void> = []
    api.starManagedConnector.mockImplementation(() => new Promise<void>((_, reject) => {
      rejects.push(reject)
    }))
    const { mutation, starredInCache } = setup(false)

    mutation.mutate({ id: 'c1', starred: true })
    await flushPromises()
    mutation.mutate({ id: 'c2', starred: true })
    await flushPromises()
    expect(starredInCache('c1')).toBe(true)
    expect(starredInCache('c2')).toBe(true)

    rejects[0]!(new Error('permission denied'))
    await flushPromises()
    rejects[1]!(new Error('permission denied'))
    await flushPromises()
    expect(starredInCache('c1')).toBe(false)
    expect(starredInCache('c2')).toBe(false)
  })

  it('reports a toggle as pending only for its connector', async () => {
    api.starManagedConnector.mockReturnValue(new Promise<void>(() => {}))
    const { mutation, pending } = setup(false)

    mutation.mutate({ id: 'c1', starred: true })
    await flushPromises()
    expect(pending.c1.value).toBe(true)
    expect(pending.c2.value).toBe(false)
  })
})
