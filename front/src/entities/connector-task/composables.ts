import type { Ref } from 'vue'
import type { ConnectorTaskStatus } from './types'
import { connectorTaskKeys } from '@shared/lib/queryKeys'
import { useQuery } from '@tanstack/vue-query'
import { computed, ref, watch } from 'vue'
import { getConnectorTaskResult } from './api'

const POLL_INTERVAL_MS = 1000
const MAX_POLL_MS = 10 * 60 * 1000

export function isTerminalTaskStatus(status: ConnectorTaskStatus | undefined): boolean {
  return status === 'completed' || status === 'failed'
}

export function useConnectorTaskResult(taskId: Ref<string | null>) {
  const timedOut = ref(false)
  const query = useQuery({
    queryKey: computed(() => connectorTaskKeys.detail(taskId.value ?? '')),
    queryFn: () => getConnectorTaskResult(taskId.value!),
    enabled: computed(() => !!taskId.value && !timedOut.value),
    refetchInterval: (query) => {
      if (query.state.status === 'error' || isTerminalTaskStatus(query.state.data?.status))
        return false
      return POLL_INTERVAL_MS
    },
  })

  watch(taskId, (id, _old, onCleanup) => {
    timedOut.value = false
    if (!id)
      return
    const timer = setTimeout(() => {
      timedOut.value = !query.isError.value && !isTerminalTaskStatus(query.data.value?.status)
    }, MAX_POLL_MS)
    onCleanup(() => clearTimeout(timer))
  }, { immediate: true })

  const polling = computed(() =>
    !!taskId.value
    && !timedOut.value
    && !query.isError.value
    && !isTerminalTaskStatus(query.data.value?.status),
  )

  return { ...query, timedOut, polling }
}
