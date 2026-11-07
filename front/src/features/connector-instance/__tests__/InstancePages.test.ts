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

const connections = vi.hoisted(() => ({ items: [] as Array<{ id: string }>, loading: false, error: null as Error | null }))

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
  useConnections: () => ({
    data: ref(connections.loading || connections.error ? undefined : { items: connections.items }),
    isLoading: ref(connections.loading),
    error: ref(connections.error),
  }),
}))
vi.mock('@entities/system', () => ({
  useExecutors: () => ({ data: ref([{ executorGroup: 'default' }, { executorGroup: 'eu' }]), isLoading: ref(false) }),
}))

const global = {
  stubs: {
    RouterLink: { props: ['to'], template: '<a :data-to="JSON.stringify(to)"><slot /></a>' },
    ConnectorStarToggle: true,
    teleport: true,
  },
}

const KINDS: Array<{ kind: InstanceKind, plural: string, detail: string, connector: string, other: string }> = [
  { kind: 'source', plural: 'Sources', detail: 'source-detail', connector: 'Postgres', other: 'BigQuery' },
  { kind: 'destination', plural: 'Destinations', detail: 'destination-detail', connector: 'BigQuery', other: 'Postgres' },
]

beforeEach(() => {
  calls.length = 0
  connections.items = []
  connections.loading = false
  connections.error = null
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

    expect(calls).toEqual([{ kind, op: 'create', arg: { name: 'Warehouse', managedConnectorId: kind === 'source' ? 'mc-1' : 'mc-2', config: {}, executorGroup: 'default' } }])
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

  it('creates the instance in the selected executor group', async () => {
    const wrapper = mount(InstanceSetupPage, { props: { kind }, global })
    await wrapper.findAll('button').find(b => b.text().includes(connector))!.trigger('click')
    await wrapper.find('input').setValue('Warehouse')
    await wrapper.find('select').setValue('eu')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(calls).toEqual([{ kind, op: 'create', arg: { name: 'Warehouse', managedConnectorId: kind === 'source' ? 'mc-1' : 'mc-2', config: {}, executorGroup: 'eu' } }])
  })

  it('saves a changed executor group when no connection uses the instance', async () => {
    const wrapper = mount(InstanceEditPage, { props: { kind }, global })
    await flushPromises()
    const select = wrapper.find('select')
    expect(select.attributes('disabled')).toBeUndefined()
    expect(wrapper.text()).not.toContain('Duplicate to group')

    await select.setValue('eu')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(calls).toEqual([{ kind, op: 'update', arg: { id: `${kind}-1`, name: `My ${kind}`, config: { host: 'db' }, executorGroup: 'eu' } }])
  })

  it('locks the group of a connected instance and duplicates it into another group', async () => {
    connections.items = [{ id: 'conn-1' }]
    const wrapper = mount(InstanceEditPage, { props: { kind }, global })
    await flushPromises()
    expect(wrapper.find('select').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('Used by connections, so the group cannot change. Duplicate it into another group instead.')

    await wrapper.findAll('button').find(b => b.text() === 'Duplicate to group')!.trigger('click')
    const selects = wrapper.findAll('select')
    expect(selects).toHaveLength(2)
    expect(selects[1].findAll('option').map(o => o.text())).toEqual(['eu'])
    const nameInput = wrapper.findAll('input').at(-1)!
    expect((nameInput.element as HTMLInputElement).value).toBe(`My ${kind} (copy)`)

    await wrapper.findAll('button').find(b => b.text() === 'Duplicate')!.trigger('click')
    await flushPromises()

    expect(calls).toEqual([{ kind, op: 'duplicate', arg: { id: `${kind}-1`, executorGroup: 'eu', name: `My ${kind} (copy)` } }])
    expect(push).toHaveBeenCalledWith({ name: detail, params: { id: `${kind}-2` } })
  })

  it('keeps the group locked while the connections of the instance are loading', async () => {
    connections.loading = true
    const wrapper = mount(InstanceEditPage, { props: { kind }, global })
    await flushPromises()
    expect(wrapper.find('select').attributes('disabled')).toBeDefined()
  })

  it('keeps the group locked when the connections of the instance fail to load', async () => {
    connections.error = new Error('unavailable')
    const wrapper = mount(InstanceEditPage, { props: { kind }, global })
    await flushPromises()
    expect(wrapper.find('select').attributes('disabled')).toBeDefined()
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
