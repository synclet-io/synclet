import { useDestinations } from '@entities/destination'
import { useSources } from '@entities/source'
import { useExecutors } from '@entities/system'
import { computed } from 'vue'
import { executorGroupOptions, isOnlyDefault } from './executorGroups'

export function useExecutorGroups() {
  const executors = useExecutors()
  const sources = useSources()
  const destinations = useDestinations()
  const groups = computed(() => executorGroupOptions(
    executors.data.value ?? [],
    sources.data.value?.items ?? [],
    destinations.data.value?.items ?? [],
  ))
  return {
    groups,
    isLoading: computed(() => executors.isLoading.value || sources.isLoading.value || destinations.isLoading.value),
    onlyDefault: computed(() => isOnlyDefault(groups.value)),
  }
}
