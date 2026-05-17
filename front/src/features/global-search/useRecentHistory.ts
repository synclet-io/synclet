import { computed, ref, watch } from 'vue'

const MAX_RECENT_SEARCHES = 5
const MAX_RECENT_VISITS = 8

export interface RecentVisit {
  id: string
  kind: string
  label: string
  to: string
  at: number
}

interface PersistedHistory {
  searches: string[]
  visits: RecentVisit[]
}

interface Storage {
  getItem: (key: string) => string | null
  setItem: (key: string, value: string) => void
}

function storageKey(userId: string, workspaceId: string): string {
  return `synclet.search.history.${userId}.${workspaceId}`
}

function load(storage: Storage, userId: string, workspaceId: string): PersistedHistory {
  if (!userId || !workspaceId)
    return { searches: [], visits: [] }
  try {
    const raw = storage.getItem(storageKey(userId, workspaceId))
    if (!raw)
      return { searches: [], visits: [] }
    const parsed = JSON.parse(raw)
    return {
      searches: Array.isArray(parsed.searches) ? parsed.searches.filter((s: unknown) => typeof s === 'string') : [],
      visits: Array.isArray(parsed.visits) ? parsed.visits.filter((v: unknown) => isVisit(v)) : [],
    }
  }
  catch {
    return { searches: [], visits: [] }
  }
}

function isVisit(v: unknown): v is RecentVisit {
  if (!v || typeof v !== 'object')
    return false
  const o = v as Record<string, unknown>
  return typeof o.id === 'string' && typeof o.kind === 'string' && typeof o.label === 'string' && typeof o.to === 'string' && typeof o.at === 'number'
}

export interface UseRecentHistoryOptions {
  userId: () => string
  workspaceId: () => string
  storage?: Storage
}

export function useRecentHistory(options: UseRecentHistoryOptions) {
  const storage: Storage = options.storage ?? (typeof window !== 'undefined'
    ? window.localStorage
    : {
        getItem: () => null,
        setItem: () => undefined,
      })

  const userId = computed(options.userId)
  const workspaceId = computed(options.workspaceId)

  const state = ref<PersistedHistory>(load(storage, userId.value, workspaceId.value))

  watch([userId, workspaceId], ([uid, wid]) => {
    state.value = load(storage, uid, wid)
  })

  function persist() {
    if (!userId.value || !workspaceId.value)
      return
    storage.setItem(storageKey(userId.value, workspaceId.value), JSON.stringify(state.value))
  }

  function rememberSearch(query: string) {
    const q = query.trim()
    if (!q)
      return
    const filtered = state.value.searches.filter(s => s !== q)
    state.value = {
      ...state.value,
      searches: [q, ...filtered].slice(0, MAX_RECENT_SEARCHES),
    }
    persist()
  }

  function rememberVisit(visit: Omit<RecentVisit, 'at'>) {
    const filtered = state.value.visits.filter(v => v.id !== visit.id || v.kind !== visit.kind)
    state.value = {
      ...state.value,
      visits: [{ ...visit, at: Date.now() }, ...filtered].slice(0, MAX_RECENT_VISITS),
    }
    persist()
  }

  function clear() {
    state.value = { searches: [], visits: [] }
    persist()
  }

  return {
    searches: computed(() => state.value.searches),
    visits: computed(() => state.value.visits),
    rememberSearch,
    rememberVisit,
    clear,
  }
}
