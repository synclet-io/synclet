import type { Ref } from 'vue'
import { useConnections } from '@entities/connection'
import { useDestinations } from '@entities/destination'
import { useRepositories } from '@entities/repository'
import { useSources } from '@entities/source'
import { computed } from 'vue'
import { rankMatches } from './matcher'

export type SearchResultKind = 'source' | 'destination' | 'connection' | 'repository' | 'settings'

export interface SearchResult {
  id: string
  kind: SearchResultKind
  label: string
  hint?: string
  to: string
}

export interface SearchGroup {
  kind: SearchResultKind
  title: string
  results: SearchResult[]
}

const SETTINGS_PAGES: SearchResult[] = [
  { id: 'settings-general', kind: 'settings', label: 'Workspace general', to: '/settings/general' },
  { id: 'settings-members', kind: 'settings', label: 'Workspace members', to: '/settings/members' },
  { id: 'settings-connectors', kind: 'settings', label: 'Connectors', to: '/settings/connectors' },
  { id: 'settings-notifications', kind: 'settings', label: 'Notifications', to: '/settings/notifications' },
  { id: 'settings-webhooks', kind: 'settings', label: 'Webhooks', to: '/settings/webhooks' },
  { id: 'settings-api-keys', kind: 'settings', label: 'API keys', to: '/settings/api-keys' },
  { id: 'settings-account', kind: 'settings', label: 'Account', to: '/settings/account' },
  { id: 'page-dashboard', kind: 'settings', label: 'Dashboard', to: '/' },
  { id: 'page-sources', kind: 'settings', label: 'Sources', to: '/sources' },
  { id: 'page-destinations', kind: 'settings', label: 'Destinations', to: '/destinations' },
  { id: 'page-connections', kind: 'settings', label: 'Connections', to: '/connections' },
  { id: 'page-jobs', kind: 'settings', label: 'Jobs', to: '/jobs' },
]

const PER_GROUP_LIMIT = 8

export function useGlobalSearch(query: Ref<string>) {
  const sources = useSources()
  const destinations = useDestinations()
  const connections = useConnections()
  const repositories = useRepositories()

  const isLoading = computed(() =>
    sources.isLoading.value || destinations.isLoading.value || connections.isLoading.value || repositories.isLoading.value,
  )

  const groups = computed<SearchGroup[]>(() => {
    const q = query.value.trim()

    const sourceResults = rankMatches(
      (sources.data.value?.items ?? []).map(s => ({
        value: { id: s.id, kind: 'source' as const, label: s.name, to: `/sources/${s.id}` },
        text: s.name,
      })),
      q,
      PER_GROUP_LIMIT,
    )
    const destinationResults = rankMatches(
      (destinations.data.value?.items ?? []).map(d => ({
        value: { id: d.id, kind: 'destination' as const, label: d.name, to: `/destinations/${d.id}` },
        text: d.name,
      })),
      q,
      PER_GROUP_LIMIT,
    )
    const connectionResults = rankMatches(
      (connections.data.value?.items ?? []).map(c => ({
        value: { id: c.id, kind: 'connection' as const, label: c.name, hint: c.status, to: `/connections/${c.id}` },
        text: c.name,
      })),
      q,
      PER_GROUP_LIMIT,
    )
    const repoResults = rankMatches(
      (repositories.data.value ?? []).map(r => ({
        value: { id: r.id, kind: 'repository' as const, label: r.name, hint: r.url, to: '/settings/connectors' },
        text: `${r.name} ${r.url}`,
      })),
      q,
      PER_GROUP_LIMIT,
    )
    const settingsResults = rankMatches(
      SETTINGS_PAGES.map(p => ({ value: p, text: p.label })),
      q,
      PER_GROUP_LIMIT,
    )

    const all: SearchGroup[] = [
      { kind: 'connection', title: 'Connections', results: connectionResults },
      { kind: 'source', title: 'Sources', results: sourceResults },
      { kind: 'destination', title: 'Destinations', results: destinationResults },
      { kind: 'repository', title: 'Repositories', results: repoResults },
      { kind: 'settings', title: 'Navigation', results: settingsResults },
    ]
    return all.filter(g => g.results.length > 0)
  })

  const flatResults = computed<SearchResult[]>(() => groups.value.flatMap(g => g.results))
  const hasResults = computed(() => flatResults.value.length > 0)

  function refreshAll() {
    sources.refetch()
    destinations.refetch()
    connections.refetch()
    repositories.refetch()
  }

  return { groups, flatResults, hasResults, isLoading, refreshAll }
}
