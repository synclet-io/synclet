import type { MaybeRef, Ref } from 'vue'
import type { JobStatus } from './types'
import { useAuth } from '@entities/auth'
import { jobKeys } from '@shared/lib/queryKeys'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, toValue, watch } from 'vue'
import * as jobApi from './api'

const ACTIVE_STATUSES: ReadonlySet<JobStatus> = new Set(['scheduled', 'starting', 'running'])
const ACTIVE_JOB_PAGE_SIZE = 5

export function isActiveJobStatus(status: JobStatus | undefined): boolean {
  return !!status && ACTIVE_STATUSES.has(status)
}

export function useJobs(connectionId: MaybeRef<string>, options?: { page?: Ref<number>, pageSize?: number }) {
  const { currentWorkspaceId } = useAuth()
  const page = options?.page
  const pageSize = options?.pageSize ?? 0
  return useQuery({
    queryKey: computed(() => [...jobKeys.list(currentWorkspaceId.value ?? '', toValue(connectionId)), page ? toValue(page) : 'all', pageSize]),
    queryFn: () => jobApi.listJobs(
      toValue(connectionId),
      page ? { pageSize, offset: (toValue(page) - 1) * pageSize } : {},
    ),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(connectionId)),
    refetchInterval: (query) => {
      const hasActive = query.state.data?.items.some(j => isActiveJobStatus(j.status))
      return hasActive ? 2_000 : 10_000
    },
  })
}

export function useActiveJob(connectionId: MaybeRef<string>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => [...jobKeys.list(currentWorkspaceId.value ?? '', toValue(connectionId)), 'active']),
    queryFn: async () => {
      const { items } = await jobApi.listJobs(toValue(connectionId), { pageSize: ACTIVE_JOB_PAGE_SIZE })
      return items.find(j => isActiveJobStatus(j.status)) ?? null
    },
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(connectionId)),
    refetchInterval: query => query.state.data ? 2_000 : 10_000,
  })
}

export function useJob(id: MaybeRef<string>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => jobKeys.detail(currentWorkspaceId.value ?? '', toValue(id))),
    queryFn: () => jobApi.getJob(toValue(id)),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(id)),
    refetchInterval: query => isActiveJobStatus(query.state.data?.status) ? 2_000 : false,
  })
}

interface JobLogBuffer {
  lines: string[]
  lastId: number
}

export function useJobLogs(jobId: MaybeRef<string>, isActive: MaybeRef<boolean>) {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()

  const query = useQuery({
    queryKey: computed(() => [...jobKeys.detail(currentWorkspaceId.value ?? '', toValue(jobId)), 'logs']),
    queryFn: async ({ queryKey }): Promise<JobLogBuffer> => {
      const prev = qc.getQueryData<JobLogBuffer>(queryKey) ?? { lines: [], lastId: 0 }
      const result = await jobApi.getJobLogs(toValue(jobId), prev.lastId)
      if (result.lines.length === 0)
        return prev
      return { lines: [...prev.lines, ...result.lines], lastId: result.lastId }
    },
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(jobId)),
    refetchInterval: computed(() => toValue(isActive) ? 5_000 : false),
  })

  watch(() => toValue(isActive), (active, wasActive) => {
    if (wasActive && !active)
      query.refetch()
  })

  return {
    lines: computed(() => query.data.value?.lines ?? []),
    isLoading: query.isLoading,
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
