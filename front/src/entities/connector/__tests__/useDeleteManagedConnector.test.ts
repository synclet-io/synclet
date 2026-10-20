import { connectorKeys } from '@shared/lib/queryKeys'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { useDeleteManagedConnector } from '../composables'

const api = vi.hoisted(() => ({ deleteManagedConnector: vi.fn() }))

vi.mock('@entities/auth', () => ({ useAuth: () => ({ currentWorkspaceId: ref('ws-1') }) }))
vi.mock('../api', () => api)

function setup() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } })
  queryClient.setQueryData(connectorKeys.managed('ws-1'), [])
  queryClient.setQueryData(connectorKeys.managed('ws-2'), [])
  let mutation!: ReturnType<typeof useDeleteManagedConnector>
  mount(defineComponent({
    setup() {
      mutation = useDeleteManagedConnector()
      return () => null
    },
  }), { global: { plugins: [[VueQueryPlugin, { queryClient }]] } })
  const invalidated = (workspaceId: string) => queryClient.getQueryState(connectorKeys.managed(workspaceId))!.isInvalidated
  return { mutation, invalidated }
}

describe('useDeleteManagedConnector', () => {
  it('deletes the connector and invalidates the workspace connector lists', async () => {
    api.deleteManagedConnector.mockResolvedValue(undefined)
    const { mutation, invalidated } = setup()

    await mutation.mutateAsync('c1')
    await flushPromises()
    expect(api.deleteManagedConnector).toHaveBeenCalledWith('c1')
    expect(invalidated('ws-1')).toBe(true)
    expect(invalidated('ws-2')).toBe(false)
  })

  it('keeps the cached lists when the delete is refused', async () => {
    api.deleteManagedConnector.mockRejectedValue(new Error('cannot delete connector: used by 1 source(s)'))
    const { mutation, invalidated } = setup()

    await expect(mutation.mutateAsync('c1')).rejects.toThrow('used by 1 source(s)')
    expect(invalidated('ws-1')).toBe(false)
  })
})
