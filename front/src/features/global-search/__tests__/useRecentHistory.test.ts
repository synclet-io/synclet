import { beforeEach, describe, expect, it } from 'vitest'
import { useRecentHistory } from '../useRecentHistory'

function makeMemoryStorage() {
  const store = new Map<string, string>()
  return {
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => { store.set(k, v) },
    raw: store,
  }
}

describe('useRecentHistory', () => {
  let mem: ReturnType<typeof makeMemoryStorage>

  beforeEach(() => {
    mem = makeMemoryStorage()
  })

  it('starts empty when nothing is persisted', () => {
    const h = useRecentHistory({ userId: () => 'u1', workspaceId: () => 'w1', storage: mem })
    expect(h.searches.value).toEqual([])
    expect(h.visits.value).toEqual([])
  })

  it('records a search and persists it', () => {
    const h = useRecentHistory({ userId: () => 'u1', workspaceId: () => 'w1', storage: mem })
    h.rememberSearch('postgres')
    expect(h.searches.value).toEqual(['postgres'])
    expect(mem.raw.get('synclet.search.history.u1.w1')).toContain('postgres')
  })

  it('deduplicates by promoting an existing search to the front', () => {
    const h = useRecentHistory({ userId: () => 'u1', workspaceId: () => 'w1', storage: mem })
    h.rememberSearch('a')
    h.rememberSearch('b')
    h.rememberSearch('a')
    expect(h.searches.value).toEqual(['a', 'b'])
  })

  it('caps recent searches at 5 entries', () => {
    const h = useRecentHistory({ userId: () => 'u1', workspaceId: () => 'w1', storage: mem })
    for (const q of ['1', '2', '3', '4', '5', '6'])
      h.rememberSearch(q)
    expect(h.searches.value).toEqual(['6', '5', '4', '3', '2'])
  })

  it('ignores blank searches', () => {
    const h = useRecentHistory({ userId: () => 'u1', workspaceId: () => 'w1', storage: mem })
    h.rememberSearch('   ')
    h.rememberSearch('')
    expect(h.searches.value).toEqual([])
  })

  it('records visits with timestamp and deduplicates by id+kind', () => {
    const h = useRecentHistory({ userId: () => 'u1', workspaceId: () => 'w1', storage: mem })
    h.rememberVisit({ id: 'src-1', kind: 'source', label: 'Postgres', to: '/sources/src-1' })
    h.rememberVisit({ id: 'src-2', kind: 'source', label: 'MySQL', to: '/sources/src-2' })
    h.rememberVisit({ id: 'src-1', kind: 'source', label: 'Postgres (renamed)', to: '/sources/src-1' })
    expect(h.visits.value.map(v => v.id)).toEqual(['src-1', 'src-2'])
    expect(h.visits.value[0].label).toBe('Postgres (renamed)')
  })

  it('scopes per workspace/user — switching keys re-reads storage', () => {
    const h1 = useRecentHistory({ userId: () => 'u1', workspaceId: () => 'w1', storage: mem })
    h1.rememberSearch('alpha')
    const h2 = useRecentHistory({ userId: () => 'u1', workspaceId: () => 'w2', storage: mem })
    expect(h2.searches.value).toEqual([])
  })

  it('survives garbage in storage by resetting to empty', () => {
    mem.raw.set('synclet.search.history.u1.w1', '{ not json')
    const h = useRecentHistory({ userId: () => 'u1', workspaceId: () => 'w1', storage: mem })
    expect(h.searches.value).toEqual([])
  })

  it('clear empties both lists', () => {
    const h = useRecentHistory({ userId: () => 'u1', workspaceId: () => 'w1', storage: mem })
    h.rememberSearch('x')
    h.rememberVisit({ id: 's', kind: 'source', label: 'l', to: '/s' })
    h.clear()
    expect(h.searches.value).toEqual([])
    expect(h.visits.value).toEqual([])
  })
})
