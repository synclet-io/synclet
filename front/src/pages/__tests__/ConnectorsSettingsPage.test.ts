import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import ConnectorsSettingsPage from '../ConnectorsSettingsPage.vue'

vi.mock('@entities/connector', () => {
  const mutation = () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: ref(false) })
  return {
    useManagedConnectors: () => ({
      data: ref([
        { id: 'c1', name: 'Faker', connectorType: 'source', dockerImage: 'airbyte/source-faker', dockerTag: '6.2', starred: true, used: false },
        { id: 'c2', name: 'Stripe', connectorType: 'source', dockerImage: 'airbyte/source-stripe', dockerTag: '5.0', starred: false, used: true },
      ]),
      isLoading: ref(false),
    }),
    useAddConnector: mutation,
    useUpdateManagedConnector: mutation,
    useBatchUpdateConnectors: mutation,
  }
})
vi.mock('@entities/repository', () => {
  const mutation = () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: ref(false) })
  return {
    useRepositories: () => ({ data: ref([]), isLoading: ref(false) }),
    useAddRepository: mutation,
    useDeleteRepository: mutation,
    useSyncRepository: mutation,
  }
})
vi.mock('@features/connector-update', () => ({ BreakingChangeDialog: defineComponent(() => () => null) }))
vi.mock('@features/connector-star', () => ({
  ConnectorStarToggle: defineComponent({
    props: { connector: { type: Object, required: true } },
    setup: props => () => h('span', { 'data-testid': 'star-toggle' }, `${(props.connector as { id: string }).id}:${(props.connector as { starred: boolean }).starred}`),
  }),
}))

describe('connectorsSettingsPage', () => {
  it('has a star toggle on each installed connector', async () => {
    const wrapper = mount(ConnectorsSettingsPage, { global: { stubs: { RouterLink: true } } })
    await flushPromises()
    expect(wrapper.findAll('[data-testid="star-toggle"]').map(t => t.text())).toEqual(['c1:true', 'c2:false'])
  })
})
