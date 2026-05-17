import type { Ref } from 'vue'
import type { TopologyFilters } from './types'
import { useConnections } from '@entities/connection'
import { useDestinations } from '@entities/destination'
import { useSources } from '@entities/source'
import { useWorkspaceStats } from '@entities/stats'
import { computed } from 'vue'
import { buildTopology } from './buildTopology'

export function useTopology(filters: Ref<TopologyFilters>) {
  const sources = useSources()
  const destinations = useDestinations()
  const connections = useConnections()
  // Workspace stats uses a fixed 24h window here; lineage is a snapshot view.
  const stats = useWorkspaceStats('24h')

  const isLoading = computed(() =>
    sources.isLoading.value || destinations.isLoading.value || connections.isLoading.value || stats.isLoading.value,
  )

  const data = computed(() => buildTopology({
    sources: sources.data.value?.items ?? [],
    destinations: destinations.data.value?.items ?? [],
    connections: connections.data.value?.items ?? [],
    healths: stats.data.value?.connectionHealths ?? [],
    filters: filters.value,
  }))

  return { data, isLoading }
}
