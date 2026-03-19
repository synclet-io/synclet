import type { MaybeRef } from 'vue'
import type { TimeRange } from './types'
import { useAuth } from '@entities/auth'
import { statsKeys } from '@shared/lib/queryKeys'
import { useQuery } from '@tanstack/vue-query'
import { computed, toValue } from 'vue'
import * as statsApi from './api'

export function useWorkspaceStats(timeRange: MaybeRef<TimeRange>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => statsKeys.workspace(currentWorkspaceId.value ?? '', toValue(timeRange))),
    queryFn: () => statsApi.getWorkspaceStats(toValue(timeRange)),
    enabled: computed(() => !!currentWorkspaceId.value),
    refetchInterval: 30_000,
  })
}

export function useSyncTimeline(timeRange: MaybeRef<TimeRange>, connectionId?: MaybeRef<string>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => statsKeys.timeline(currentWorkspaceId.value ?? '', toValue(timeRange), connectionId ? toValue(connectionId) : '')),
    queryFn: () => statsApi.getSyncTimeline(toValue(timeRange), connectionId ? toValue(connectionId) : undefined),
    enabled: computed(() => !!currentWorkspaceId.value),
    refetchInterval: 30_000,
  })
}

export function useConnectionStats(connectionId: MaybeRef<string>, timeRange: MaybeRef<TimeRange>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => statsKeys.connection(currentWorkspaceId.value ?? '', toValue(connectionId), toValue(timeRange))),
    queryFn: () => statsApi.getConnectionStats(toValue(connectionId), toValue(timeRange)),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(connectionId)),
    refetchInterval: 30_000,
  })
}
