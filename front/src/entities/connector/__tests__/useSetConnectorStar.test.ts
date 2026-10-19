import type { ManagedConnector } from '../types'
import { connectorKeys } from '@shared/lib/queryKeys'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { useSetConnectorStar } from '../composables'

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
  ])
  let mutation!: ReturnType<typeof useSetConnectorStar>
  mount(defineComponent({
    setup() {
      mutation = useSetConnectorStar()
      return () => null
    },
  }), { global: { plugins: [[VueQueryPlugin, { queryClient }]] } })
  const starredInCache = () => queryClient.getQueryData<ManagedConnector[]>(KEY)![0]!.starred
  return { mutation, starredInCache }
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
})
