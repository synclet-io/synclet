import type { InstanceKind } from '../instanceKind'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import InstanceEditPage from '../InstanceEditPage.vue'
import InstanceSetupPage from '../InstanceSetupPage.vue'
import { calls } from './fakeEntities'

vi.mock('vue-router', async importOriginal => ({
  ...await importOriginal<typeof import('vue-router')>(),
  useRouter: () => ({ push: vi.fn() }),
  useRoute: () => ({ params: { id: 'x' } }),
  onBeforeRouteLeave: vi.fn(),
}))
vi.mock('@entities/source', async () => (await import('./fakeEntities')).fakeEntity('source'))
vi.mock('@entities/destination', async () => (await import('./fakeEntities')).fakeEntity('destination'))
vi.mock('@entities/connector', () => ({
  useManagedConnectors: () => ({
    data: ref([
      { id: 'mc-1', name: 'Postgres', connectorType: 'source', dockerImage: 'airbyte/source-postgres', dockerTag: '1.0' },
      { id: 'mc-2', name: 'BigQuery', connectorType: 'destination', dockerImage: 'airbyte/destination-bigquery', dockerTag: '2.0' },
    ]),
    isLoading: ref(false),
    error: ref(null),
  }),
  useConnectorSpec: () => ({
    data: ref({
      spec: {
        connectionSpecification: {
          type: 'object',
          required: ['host', 'port'],
          properties: {
            host: { type: 'string', title: 'Host' },
            port: { type: 'integer', title: 'Port', minimum: 1 },
          },
        },
      },
      externalDocumentationUrls: [],
    }),
    isLoading: ref(false),
    error: ref(null),
  }),
}))
vi.mock('@entities/connector-task', () => ({
  useConnectorTaskResult: () => ({ data: ref(null), polling: ref(false), error: ref(null), timedOut: ref(false) }),
}))

const global = { stubs: { ConnectorStarToggle: true, RouterLink: { template: '<a><slot /></a>' } } }

beforeEach(() => {
  calls.length = 0
})

describe.each<{ kind: InstanceKind, connector: string }>([
  { kind: 'source', connector: 'Postgres' },
  { kind: 'destination', connector: 'BigQuery' },
])('$kind config validation on submit', ({ kind, connector }) => {
  it('blocks creating with missing required fields and marks them', async () => {
    const wrapper = mount(InstanceSetupPage, { props: { kind }, global })
    await wrapper.findAll('button').find(b => b.text().includes(connector))!.trigger('click')
    await wrapper.find('input').setValue('Warehouse')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(calls).toEqual([])
    expect(wrapper.text()).toContain('Fix the highlighted fields')
    expect(wrapper.find('#field-host').classes()).toContain('border-danger')
    expect(wrapper.find('#field-port').classes()).toContain('border-danger')

    await wrapper.find('#field-host').setValue('db')
    await wrapper.find('#field-port').setValue('5432')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(calls).toEqual([{ kind, op: 'create', arg: { name: 'Warehouse', managedConnectorId: kind === 'source' ? 'mc-1' : 'mc-2', config: { host: 'db', port: 5432 } } }])
  })

  it('blocks saving an edit whose stored config misses a required field', async () => {
    const wrapper = mount(InstanceEditPage, { props: { kind }, global })
    await flushPromises()
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(calls).toEqual([])
    expect(wrapper.find('#field-port').classes()).toContain('border-danger')
    expect(wrapper.find('#field-host').classes()).not.toContain('border-danger')

    await wrapper.find('#field-port').setValue('5432')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(calls).toEqual([{ kind, op: 'update', arg: { id: `${kind}-1`, name: `My ${kind}`, config: { host: 'db', port: 5432 } } }])
  })
})
