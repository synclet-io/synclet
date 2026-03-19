import type { Ref } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { getConnectorTaskResult } from './api'

export function useConnectorTaskResult(taskId: Ref<string | null>) {
  return useQuery({
    queryKey: computed(() => ['connector-task', taskId.value] as const),
    queryFn: () => getConnectorTaskResult(taskId.value!),
    enabled: computed(() => !!taskId.value),
    refetchInterval: (query) => {
      const status = query.state.data?.status
      if (status === 'completed' || status === 'failed')
        return false
      return 1000
    },
  })
}
