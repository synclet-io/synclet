import StreamConfigEditor from '@features/stream-config/StreamConfigEditor.vue'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import StreamConfigPage from '../StreamConfigPage.vue'

const mocks = vi.hoisted(() => ({ configure: vi.fn(async () => undefined), push: vi.fn() }))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: 'c1' } }),
  useRouter: () => ({ push: mocks.push }),
  onBeforeRouteLeave: vi.fn(),
}))
vi.mock('@entities/connection', () => ({
  discoverSchema: vi.fn(),
  getConfiguredCatalog: vi.fn(async () => ({
    streams: [{ name: 'orders', syncMode: 'full_refresh', destinationSyncMode: 'overwrite', selectedFields: [] }],
  })),
  getDiscoveredCatalog: vi.fn(async () => undefined),
  useConfigureStreams: () => ({ mutateAsync: mocks.configure }),
  useConnection: () => ({ data: ref({ id: 'c1', name: 'Orders', sourceId: 's1' }) }),
  useResetConnectionState: () => ({ mutateAsync: vi.fn(), isPending: ref(false) }),
  useStreamStates: () => ({ data: ref(undefined) }),
}))
vi.mock('@entities/connector-task', () => ({ useConnectorTaskResult: () => ({ data: ref(null) }) }))
vi.mock('@entities/source', () => ({
  getSourceCatalog: vi.fn(async () => ({
    catalog: {
      streams: [{
        stream: {
          name: 'orders',
          json_schema: { properties: { id: { type: 'integer' }, total: { type: 'number' } } },
          supported_sync_modes: ['full_refresh'],
        },
      }],
    },
  })),
}))
vi.mock('@shared/ui', async (orig) => {
  const actual = await orig<typeof import('@shared/ui')>()
  return { ...actual, useToast: () => ({ success: vi.fn(), error: vi.fn() }) }
})

async function mountPage() {
  const wrapper = mount(StreamConfigPage, { global: { stubs: { StreamExpandedRow: true, PageHeader: { template: '<div><slot name="actions" /></div>' } } } })
  await flushPromises()
  return wrapper
}

describe('stream configuration page', () => {
  it('edits streams with the shared stream editor', async () => {
    const wrapper = await mountPage()
    expect(wrapper.findComponent(StreamConfigEditor).exists()).toBe(true)
  })

  it('shows a saved stream with an empty field selection as all fields and saves it', async () => {
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('2/2 fields')

    await wrapper.findAll('button').find(b => b.text().includes('Save Configuration'))!.trigger('click')
    await flushPromises()

    expect(mocks.configure).toHaveBeenCalledWith({
      connectionId: 'c1',
      streams: [expect.objectContaining({ streamName: 'orders', selectedFields: [] })],
    })
  })
})
