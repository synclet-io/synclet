import type { InstanceKind } from '../instanceKind'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import InstanceSetupPage from '../InstanceSetupPage.vue'

const setStar = vi.hoisted(() => vi.fn())

vi.mock('vue-router', async importOriginal => ({
  ...await importOriginal<typeof import('vue-router')>(),
  useRouter: () => ({ push: vi.fn() }),
  onBeforeRouteLeave: vi.fn(),
}))
vi.mock('@entities/source', async () => (await import('./fakeEntities')).fakeEntity('source'))
vi.mock('@entities/destination', async () => (await import('./fakeEntities')).fakeEntity('destination'))
vi.mock('@entities/auth', () => ({
  useAuth: () => ({ currentWorkspaceId: ref('ws-1'), workspaces: ref([{ workspaceId: 'ws-1', role: 'editor' }]) }),
}))
vi.mock('@entities/connector', () => {
  const c = (id: string, name: string, connectorType: string, used: boolean, starred: boolean) =>
    ({ id, name, connectorType, used, starred, dockerImage: `airbyte/${connectorType}-${id}`, dockerTag: '1.0' })
  return {
    useManagedConnectors: () => ({
      data: ref([
        c('zendesk', 'Zendesk', 'source', true, false),
        c('faker', 'Faker', 'source', false, false),
        c('stripe', 'Stripe', 'source', false, true),
        c('amplitude', 'Amplitude', 'source', true, true),
        c('bigquery', 'BigQuery', 'destination', true, true),
        c('snowflake', 'Snowflake', 'destination', false, false),
        c('duckdb', 'DuckDB', 'destination', false, true),
      ]),
      isLoading: ref(false),
      error: ref(null),
    }),
    useConnectorSpec: () => ({ data: ref(null), isLoading: ref(false), error: ref(null) }),
    useSetConnectorStar: () => ({ mutate: setStar, isPending: ref(false) }),
  }
})
vi.mock('@entities/connector-task', () => ({
  useConnectorTaskResult: () => ({ data: ref(null), polling: ref(false), error: ref(null), timedOut: ref(false) }),
}))

const global = { stubs: { RouterLink: true, JsonSchemaForm: true } }

function sections(wrapper: ReturnType<typeof mount>) {
  return wrapper.findAll('[data-testid="connector-section"]').map(s => ({
    title: s.find('h2').text(),
    names: s.findAll('h3').map(h => h.text()),
  }))
}

const CASES: Array<{ kind: InstanceKind, toStar: string, used: string[], other: string[], starred: Array<{ title: string, names: string[] }> }> = [
  {
    kind: 'source',
    toStar: 'Faker',
    used: ['Amplitude', 'Zendesk'],
    other: ['Faker', 'Stripe'],
    starred: [{ title: 'Used in this workspace', names: ['Amplitude'] }, { title: 'Other connectors', names: ['Stripe'] }],
  },
  {
    kind: 'destination',
    toStar: 'Snowflake',
    used: ['BigQuery'],
    other: ['DuckDB', 'Snowflake'],
    starred: [{ title: 'Used in this workspace', names: ['BigQuery'] }, { title: 'Other connectors', names: ['DuckDB'] }],
  },
]

beforeEach(() => setStar.mockClear())

describe.each(CASES)('$kind connector picker', ({ kind, toStar, used, other, starred }) => {
  it('shows the used connectors first, then the rest, each A to Z', () => {
    const wrapper = mount(InstanceSetupPage, { props: { kind }, global })
    expect(sections(wrapper)).toEqual([
      { title: 'Used in this workspace', names: used },
      { title: 'Other connectors', names: other },
    ])
  })

  it('filters both sections to starred connectors', async () => {
    const wrapper = mount(InstanceSetupPage, { props: { kind }, global })
    const filter = wrapper.findAll('button').find(b => b.text() === 'Starred')!
    await filter.trigger('click')
    expect(filter.attributes('aria-pressed')).toBe('true')
    expect(sections(wrapper)).toEqual(starred)
  })

  it('stars a connector in place', async () => {
    const wrapper = mount(InstanceSetupPage, { props: { kind }, global })
    const card = wrapper.findAll('[data-testid="connector-card"]').find(c => c.find('h3').text() === toStar)!
    await card.get(`button[aria-label="Star ${toStar}"]`).trigger('click')
    expect(setStar.mock.calls[0]![0]).toEqual({ id: toStar.toLowerCase(), starred: true })
    expect(sections(wrapper)).toHaveLength(2)
    expect(wrapper.find('form').exists()).toBe(false)
  })
})
