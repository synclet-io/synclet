import type { InstanceKind } from '../instanceKind'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import DeleteInstanceDialog from '../DeleteInstanceDialog.vue'
import InstanceConnectionsCard from '../InstanceConnectionsCard.vue'
import InstanceEditPage from '../InstanceEditPage.vue'
import InstanceListPage from '../InstanceListPage.vue'
import InstanceSetupPage from '../InstanceSetupPage.vue'
import { calls } from './fakeEntities'

const push = vi.fn()
vi.mock('vue-router', async importOriginal => ({
  ...await importOriginal<typeof import('vue-router')>(),
  useRouter: () => ({ push }),
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
    data: ref({ spec: { connectionSpecification: { type: 'object' } }, externalDocumentationUrls: [] }),
    isLoading: ref(false),
    error: ref(null),
  }),
}))
vi.mock('@entities/connector-task', () => ({
  useConnectorTaskResult: () => ({ data: ref(null), polling: ref(false), error: ref(null), timedOut: ref(false) }),
}))
vi.mock('@entities/connection', () => ({
  useConnections: () => ({ data: ref({ items: [] }), isLoading: ref(false) }),
}))

const global = {
  stubs: {
    RouterLink: { props: ['to'], template: '<a :data-to="JSON.stringify(to)"><slot /></a>' },
    JsonSchemaForm: true,
    ConnectorStarToggle: true,
  },
}

const KINDS: Array<{ kind: InstanceKind, plural: string, detail: string, connector: string, other: string }> = [
  { kind: 'source', plural: 'Sources', detail: 'source-detail', connector: 'Postgres', other: 'BigQuery' },
  { kind: 'destination', plural: 'Destinations', detail: 'destination-detail', connector: 'BigQuery', other: 'Postgres' },
]

beforeEach(() => {
  calls.length = 0
  push.mockClear()
})

describe.each(KINDS)('$kind pages', ({ kind, plural, detail, connector, other }) => {
  it('lists instances linking to their detail page and deletes through the kind', async () => {
    const wrapper = mount(InstanceListPage, { props: { kind }, global })
    expect(wrapper.find('h1').text()).toBe(plural)
    const link = wrapper.findAll('a').find(a => a.text() === `My ${kind}`)!
    expect(JSON.parse(link.attributes('data-to')!)).toEqual({ name: detail, params: { id: `${kind}-1` } })

    await wrapper.find('button[title="Delete"]').trigger('click')
    const dialog = wrapper.findComponent(DeleteInstanceDialog)
    expect(dialog.props('kind')).toBe(kind)
    dialog.vm.$emit('confirm')
    await flushPromises()
    expect(calls).toEqual([{ kind, op: 'delete', arg: `${kind}-1` }])
  })

  it('offers only connectors of the kind and opens the created instance', async () => {
    const wrapper = mount(InstanceSetupPage, { props: { kind }, global })
    expect(wrapper.text()).toContain(connector)
    expect(wrapper.text()).not.toContain(other)

    await wrapper.findAll('button').find(b => b.text().includes(connector))!.trigger('click')
    await wrapper.find('input').setValue('Warehouse')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(calls).toEqual([{ kind, op: 'create', arg: { name: 'Warehouse', managedConnectorId: kind === 'source' ? 'mc-1' : 'mc-2', config: {} } }])
    expect(push).toHaveBeenCalledWith({ name: detail, params: { id: `${kind}-1` } })
  })

  it('saves edits through the kind and returns to the detail page', async () => {
    const wrapper = mount(InstanceEditPage, { props: { kind }, global })
    await flushPromises()
    const input = wrapper.find('input')
    expect((input.element as HTMLInputElement).value).toBe(`My ${kind}`)

    await input.setValue('Renamed')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(calls).toEqual([{ kind, op: 'update', arg: { id: `${kind}-1`, name: 'Renamed', config: { host: 'db' } } }])
    expect(push).toHaveBeenCalledWith({ name: detail, params: { id: `${kind}-1` } })
  })

  it('preselects the instance when creating a connection from it', () => {
    const wrapper = mount(InstanceConnectionsCard, { props: { kind, instanceId: 'abc' }, global })
    const cta = wrapper.findAll('a').find(a => a.text().includes('Create a connection'))!
    expect(JSON.parse(cta.attributes('data-to')!)).toEqual({
      name: 'connection-new',
      query: { [kind === 'source' ? 'sourceId' : 'destinationId']: 'abc' },
    })
  })
})
