<script setup lang="ts">
import type { SearchResult, SearchResultKind } from './useGlobalSearch'
import { useAuth } from '@entities/auth'
import { ArrowRightLeft, Cog, Database, GitBranch, Search, Server } from 'lucide-vue-next'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useGlobalSearch } from './useGlobalSearch'
import { useRecentHistory } from './useRecentHistory'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()

const router = useRouter()
const auth = useAuth()
const inputEl = ref<HTMLInputElement | null>(null)
const dialogEl = ref<HTMLElement | null>(null)
let returnFocusTo: HTMLElement | null = null

const query = ref('')
const { groups, flatResults, hasResults, isLoading, refreshAll } = useGlobalSearch(query)

const history = useRecentHistory({
  userId: () => auth.user.value?.id ?? '',
  workspaceId: () => auth.currentWorkspaceId.value ?? '',
})

const showHistory = computed(() => !query.value.trim() && (history.searches.value.length > 0 || history.visits.value.length > 0))

// Unified flat list for keyboard navigation: history items when in history view, results otherwise.
const historyVisitResults = computed<SearchResult[]>(() =>
  history.visits.value.map(v => ({
    id: v.id,
    kind: v.kind as SearchResultKind,
    label: v.label,
    to: v.to,
  })),
)

const navList = computed<SearchResult[]>(() => {
  if (showHistory.value)
    return historyVisitResults.value
  return flatResults.value
})

const activeIndex = ref(0)

watch(() => props.open, async (open) => {
  if (open) {
    returnFocusTo = document.activeElement instanceof HTMLElement ? document.activeElement : null
    query.value = ''
    activeIndex.value = 0
    refreshAll()
    await nextTick()
    inputEl.value?.focus()
  }
  else {
    returnFocusTo?.focus()
    returnFocusTo = null
  }
})

watch(navList, (list) => {
  if (activeIndex.value >= list.length)
    activeIndex.value = Math.max(0, list.length - 1)
})

watch(() => query.value, () => {
  activeIndex.value = 0
})

function activate(result: SearchResult) {
  if (query.value.trim())
    history.rememberSearch(query.value)
  history.rememberVisit({ id: result.id, kind: result.kind, label: result.label, to: result.to })
  emit('close')
  router.push(result.to)
}

function activateAtIndex(idx: number) {
  const r = navList.value[idx]
  if (r)
    activate(r)
}

function trapTab(e: KeyboardEvent) {
  const focusables = Array.from(dialogEl.value?.querySelectorAll<HTMLElement>('input, button:not([disabled])') ?? [])
  const first = focusables[0]
  const last = focusables.at(-1)
  if (!first || !last)
    return
  const inside = dialogEl.value?.contains(document.activeElement) ?? false
  if (e.shiftKey && (document.activeElement === first || !inside)) {
    e.preventDefault()
    last.focus()
  }
  else if (!e.shiftKey && (document.activeElement === last || !inside)) {
    e.preventDefault()
    first.focus()
  }
}

function onKeydown(e: KeyboardEvent) {
  if (!props.open)
    return
  if (e.key === 'Escape') {
    e.preventDefault()
    emit('close')
    return
  }
  if (e.key === 'Tab') {
    trapTab(e)
    return
  }
  if (e.target !== inputEl.value)
    return
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    activeIndex.value = Math.min(activeIndex.value + 1, navList.value.length - 1)
  }
  else if (e.key === 'ArrowUp') {
    e.preventDefault()
    activeIndex.value = Math.max(activeIndex.value - 1, 0)
  }
  else if (e.key === 'Enter') {
    e.preventDefault()
    activateAtIndex(activeIndex.value)
  }
}

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
})

const ICONS: Record<SearchResultKind, typeof Database> = {
  source: Database,
  destination: Server,
  connection: ArrowRightLeft,
  repository: GitBranch,
  settings: Cog,
}

function indexOfResult(group: ReturnType<typeof useGlobalSearch>['groups']['value'][number], result: SearchResult): number {
  let acc = 0
  for (const g of groups.value) {
    if (g === group)
      return acc + g.results.indexOf(result)
    acc += g.results.length
  }
  return -1
}

function pickRecentSearch(q: string) {
  query.value = q
  inputEl.value?.focus()
}
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-150"
      leave-active-class="transition-opacity duration-100"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <div v-if="open" class="fixed inset-0 z-50 flex items-start justify-center pt-[10vh] px-4">
        <div class="fixed inset-0 bg-slate-900/60 backdrop-blur-sm" @click="emit('close')" />
        <div
          ref="dialogEl"
          role="dialog"
          aria-modal="true"
          aria-label="Search"
          class="relative bg-surface border border-border rounded-2xl shadow-overlay w-full max-w-xl flex flex-col max-h-[70vh]"
        >
          <div class="flex items-center gap-2 px-4 py-3 border-b border-border">
            <Search class="w-4 h-4 text-text-muted" />
            <input
              ref="inputEl"
              v-model="query"
              type="text"
              placeholder="Search connections, sources, destinations…"
              aria-label="Search"
              class="flex-1 bg-transparent outline-none text-sm placeholder-text-muted text-heading"
              autocomplete="off"
              spellcheck="false"
            >
            <button
              type="button"
              class="rounded focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
              aria-label="Close search"
              @click="emit('close')"
            >
              <kbd class="inline-flex text-xs text-text-muted border border-border rounded px-1.5 py-0.5">Esc</kbd>
            </button>
          </div>

          <div class="flex-1 overflow-y-auto py-2">
            <!-- History view -->
            <div v-if="showHistory" class="px-2">
              <div v-if="history.visits.value.length > 0" class="mb-3">
                <p class="px-3 py-1 text-[11px] font-semibold uppercase tracking-wider text-text-muted">
                  Recently visited
                </p>
                <button
                  v-for="(visit, vi) in history.visits.value"
                  :key="`${visit.kind}-${visit.id}`"
                  type="button"
                  class="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors text-left focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
                  :class="vi === activeIndex ? 'bg-primary/15 text-primary' : 'text-heading hover:bg-surface-hover'"
                  @mouseenter="activeIndex = vi"
                  @click="activate({ id: visit.id, kind: visit.kind as SearchResultKind, label: visit.label, to: visit.to })"
                >
                  <component :is="ICONS[visit.kind as SearchResultKind] ?? Cog" class="w-4 h-4 text-text-muted shrink-0" />
                  <span class="truncate">{{ visit.label }}</span>
                </button>
              </div>
              <div v-if="history.searches.value.length > 0">
                <p class="px-3 py-1 text-[11px] font-semibold uppercase tracking-wider text-text-muted">
                  Recent searches
                </p>
                <button
                  v-for="q in history.searches.value"
                  :key="q"
                  type="button"
                  class="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-heading hover:bg-surface-hover transition-colors text-left focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
                  @click="pickRecentSearch(q)"
                >
                  <Search class="w-4 h-4 text-text-muted shrink-0" />
                  <span class="truncate">{{ q }}</span>
                </button>
              </div>
            </div>

            <div v-else-if="!hasResults && isLoading" role="status" class="px-6 py-10 text-center text-sm text-text-muted">
              Loading…
            </div>

            <!-- Empty state when there's a query but no results -->
            <div v-else-if="!hasResults" class="px-6 py-10 text-center text-sm text-text-muted">
              No results for &quot;{{ query }}&quot;
            </div>

            <!-- Result groups -->
            <div v-else class="px-2 space-y-3">
              <div v-for="group in groups" :key="group.kind">
                <p class="px-3 py-1 text-[11px] font-semibold uppercase tracking-wider text-text-muted">
                  {{ group.title }}
                </p>
                <button
                  v-for="result in group.results"
                  :key="`${group.kind}-${result.id}`"
                  type="button"
                  class="w-full flex items-center justify-between gap-3 px-3 py-2 rounded-lg text-sm text-heading transition-colors text-left focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
                  :class="indexOfResult(group, result) === activeIndex ? 'bg-primary/15 text-primary' : 'hover:bg-surface-hover'"
                  @mouseenter="activeIndex = indexOfResult(group, result)"
                  @click="activate(result)"
                >
                  <span class="flex items-center gap-3 min-w-0">
                    <component :is="ICONS[result.kind]" class="w-4 h-4 shrink-0 opacity-70" />
                    <span class="truncate">{{ result.label }}</span>
                  </span>
                  <span v-if="result.hint" class="text-xs text-text-muted truncate max-w-[40%]">
                    {{ result.hint }}
                  </span>
                </button>
              </div>
              <p v-if="isLoading" role="status" class="px-3 py-1 text-xs text-text-muted">
                Loading more results…
              </p>
            </div>
          </div>
          <div class="px-4 py-2 border-t border-border text-[11px] text-text-muted flex items-center justify-between">
            <span><kbd class="border border-border rounded px-1">↑</kbd> <kbd class="border border-border rounded px-1">↓</kbd> navigate</span>
            <span><kbd class="border border-border rounded px-1">↵</kbd> open</span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
