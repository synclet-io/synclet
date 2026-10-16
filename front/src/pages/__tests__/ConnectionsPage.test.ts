import type { Router } from 'vue-router'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, defineComponent, ref, toValue } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import ConnectionsPage from '../ConnectionsPage.vue'

const SRC_PG = '11111111-1111-4111-8111-111111111111'
const SRC_MYSQL = '22222222-2222-4222-8222-222222222222'
const DST_BQ = '33333333-3333-4333-8333-333333333333'
const DST_S3 = '44444444-4444-4444-8444-444444444444'

const mocks = vi.hoisted(() => ({
  connections: [] as Array<Record<string, unknown>>,
}))

function conn(id: string, name: string, sourceId: string, destinationId: string) {
  return { id, name, sourceId, destinationId, sourceName: '', destinationName: '', status: 'active', schedule: '', pendingSchemaChanges: 0 }
}

vi.mock('@entities/connection', () => ({
  useConnections: (opts: { sourceId: unknown, destinationId: unknown }) => {
    const data = computed(() => {
      const sourceId = toValue(opts.sourceId) as string
      const destinationId = toValue(opts.destinationId) as string
      const items = mocks.connections.filter(c =>
        (!sourceId || c.sourceId === sourceId) && (!destinationId || c.destinationId === destinationId))
      return { items, total: items.length }
    })
    return { data, isLoading: ref(false), error: ref(null) }
  },
}))
vi.mock('@entities/connection/api', () => ({
  deleteConnection: vi.fn(),
  enableConnection: vi.fn(),
  disableConnection: vi.fn(),
  listConnections: vi.fn(),
}))
vi.mock('@entities/source', () => ({
  useSources: () => ({ data: ref({ items: [{ id: SRC_PG, name: 'Postgres' }, { id: SRC_MYSQL, name: 'MySQL' }] }) }),
}))
vi.mock('@entities/destination', () => ({
  useDestinations: () => ({ data: ref({ items: [{ id: DST_BQ, name: 'BigQuery' }, { id: DST_S3, name: 'S3' }] }) }),
}))
vi.mock('@entities/auth', () => ({
  useAuth: () => ({ currentWorkspaceId: ref('ws1') }),
}))
vi.mock('@tanstack/vue-query', () => ({
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}))

const Empty = defineComponent({ template: '<div />' })

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/connections', name: 'connections', component: ConnectionsPage },
      { path: '/connections/:id', name: 'connection-detail', component: Empty },
      { path: '/connections/:id/schema-changes', name: 'connection-schema-changes', component: Empty },
      { path: '/sources/:id', name: 'source-detail', component: Empty },
      { path: '/destinations/:id', name: 'destination-detail', component: Empty },
      { path: '/:rest(.*)*', component: Empty },
    ],
  })
}

async function mountAt(query: Record<string, string> = {}) {
  const router = makeRouter()
  await router.push({ path: '/connections', query })
  const wrapper = mount(ConnectionsPage, { global: { plugins: [router] }, attachTo: document.body })
  await flushPromises()
  return { router, wrapper }
}

function combobox(wrapper: ReturnType<typeof mount>, label: string) {
  const root = wrapper.findAll('label').find(l => l.text() === label)!.element.closest('div')!.parentElement!
  return wrapper.find(`#${CSS.escape(root.querySelector('input')!.id)}`)
}

async function pick(wrapper: ReturnType<typeof mount>, label: string, option: string) {
  await combobox(wrapper, label).trigger('click')
  const li = wrapper.findAll('li[role="option"]').find(o => o.text() === option)!
  await li.trigger('click')
  await flushPromises()
}

function rowNames(wrapper: ReturnType<typeof mount>) {
  return wrapper.findAll('a[href^="/connections/c"]').map(a => a.text())
}

describe('connectionsPage filters', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    mocks.connections = [
      conn('c1', 'PG to BQ', SRC_PG, DST_BQ),
      conn('c2', 'MySQL to BQ', SRC_MYSQL, DST_BQ),
      conn('c3', 'PG to S3', SRC_PG, DST_S3),
    ]
  })

  it('writes the picked source and destination to the URL query', async () => {
    const { router, wrapper } = await mountAt()

    await pick(wrapper, 'Source', 'Postgres')
    expect(router.currentRoute.value.query).toEqual({ sourceId: SRC_PG })
    expect(rowNames(wrapper)).toEqual(['PG to BQ', 'PG to S3'])

    await pick(wrapper, 'Destination', 'S3')
    expect(router.currentRoute.value.query).toEqual({ sourceId: SRC_PG, destinationId: DST_S3 })
    expect(rowNames(wrapper)).toEqual(['PG to S3'])
  })

  it('preselects the filters from the URL query on load', async () => {
    const { wrapper } = await mountAt({ sourceId: SRC_MYSQL, destinationId: DST_BQ })

    expect((combobox(wrapper, 'Source').element as HTMLInputElement).value).toBe('MySQL')
    expect((combobox(wrapper, 'Destination').element as HTMLInputElement).value).toBe('BigQuery')
    expect(rowNames(wrapper)).toEqual(['MySQL to BQ'])
  })

  it('clearing one filter removes it from the URL', async () => {
    const { router, wrapper } = await mountAt({ sourceId: SRC_PG, destinationId: DST_S3 })

    await wrapper.find('button[aria-label="Clear Source"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query).toEqual({ destinationId: DST_S3 })
    expect((combobox(wrapper, 'Source').element as HTMLInputElement).value).toBe('')
    expect(rowNames(wrapper)).toEqual(['PG to S3'])
  })

  it('clear filters removes both from the URL and shows all connections', async () => {
    const { router, wrapper } = await mountAt({ sourceId: SRC_PG, destinationId: DST_S3, page: '2' })

    const clearAll = wrapper.findAll('button').find(b => b.text() === 'Clear filters')!
    await clearAll.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query).toEqual({})
    expect(rowNames(wrapper)).toEqual(['PG to BQ', 'MySQL to BQ', 'PG to S3'])
  })

  it('shows the filtered empty state when the filter matches nothing', async () => {
    const { wrapper } = await mountAt({ sourceId: SRC_MYSQL, destinationId: DST_S3 })

    expect(rowNames(wrapper)).toEqual([])
    expect(wrapper.text()).toContain('No connections match these filters')
    expect(wrapper.text()).not.toContain('No connections yet')
    expect(wrapper.find('input[role="combobox"]').exists()).toBe(true)
  })
})
