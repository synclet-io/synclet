import { connectorKeys } from '@shared/lib/queryKeys'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { useCreateSource, useDeleteSource } from '../composables'

const api = vi.hoisted(() => ({
  createSource: vi.fn().mockResolvedValue({}),
  deleteSource: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('@entities/auth', () => ({ useAuth: () => ({ currentWorkspaceId: ref('ws-1') }) }))
vi.mock('../api', () => api)

function setup() {
  const queryClient = new QueryClient()
  const invalidate = vi.spyOn(queryClient, 'invalidateQueries')
  let mutations!: { create: ReturnType<typeof useCreateSource>, remove: ReturnType<typeof useDeleteSource> }
  mount(defineComponent({
    setup() {
      mutations = { create: useCreateSource(), remove: useDeleteSource() }
      return () => null
    },
  }), { global: { plugins: [[VueQueryPlugin, { queryClient }]] } })
  const invalidatedManagedConnectors = () => invalidate.mock.calls.some(([filters]) =>
    JSON.stringify(filters?.queryKey) === JSON.stringify([...connectorKeys.all('ws-1'), 'managed']))
  return { mutations, invalidatedManagedConnectors }
}

describe('source mutations', () => {
  it('reload the connector lists after a source is created', async () => {
    const { mutations, invalidatedManagedConnectors } = setup()
    mutations.create.mutate({} as never)
    await flushPromises()
    expect(invalidatedManagedConnectors()).toBe(true)
  })

  it('reload the connector lists after a source is deleted', async () => {
    const { mutations, invalidatedManagedConnectors } = setup()
    mutations.remove.mutate('id-1' as never)
    await flushPromises()
    expect(invalidatedManagedConnectors()).toBe(true)
  })
})
