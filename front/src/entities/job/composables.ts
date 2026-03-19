import type { MaybeRef, Ref } from 'vue'
import { useAuth } from '@entities/auth'
import { jobKeys } from '@shared/lib/queryKeys'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, ref, toValue, watch } from 'vue'
import * as jobApi from './api'

export function useJobs(connectionId: MaybeRef<string>, options?: { page?: Ref<number>, pageSize?: number }) {
  const { currentWorkspaceId } = useAuth()
  const page = options?.page
  const pageSize = options?.pageSize ?? 0
  return useQuery({
    queryKey: computed(() => [...jobKeys.list(currentWorkspaceId.value ?? '', toValue(connectionId)), page ? toValue(page) : 'all']),
    queryFn: () => jobApi.listJobs(
      toValue(connectionId),
      page ? { pageSize, offset: (toValue(page) - 1) * pageSize } : {},
    ),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(connectionId)),
    refetchInterval: (query) => {
      const hasActive = query.state.data?.items.some(j => j.status === 'running' || j.status === 'scheduled' || j.status === 'starting')
      return hasActive ? 2_000 : 10_000
    },
  })
}

export function useJob(id: MaybeRef<string>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => jobKeys.detail(currentWorkspaceId.value ?? '', toValue(id))),
    queryFn: () => jobApi.getJob(toValue(id)),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(id)),
  })
}

export function useJobLogs(jobId: Ref<string>, isActive: Ref<boolean>) {
  const accumulatedLines = ref<string[]>([])
  const lastId = ref(0)

  // Reset when job changes
  watch(jobId, () => {
    accumulatedLines.value = []
    lastId.value = 0
  })

  const { isLoading } = useQuery({
    queryKey: computed(() => [...jobKeys.detail('', toValue(jobId)), 'logs']),
    queryFn: async () => {
      const result = await jobApi.getJobLogs(toValue(jobId), lastId.value)
      if (result.lines.length > 0) {
        accumulatedLines.value = [...accumulatedLines.value, ...result.lines]
        lastId.value = result.lastId
      }
      return result
    },
    enabled: computed(() => !!toValue(jobId)),
    refetchInterval: computed(() => isActive.value ? 5000 : false),
  })

  return {
    lines: accumulatedLines,
    isLoading,
  }
}

export function useCancelJob() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: jobApi.cancelJob,
    onSuccess: () => { qc.invalidateQueries({ queryKey: jobKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}

export function useTriggerSync() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: jobApi.triggerSync,
    onSuccess: () => { qc.invalidateQueries({ queryKey: jobKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}
