import type { MaybeRef } from 'vue'
import type { ListAuditEventsParams } from './types'
import { useAuth } from '@entities/auth'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, toValue } from 'vue'
import * as auditApi from './api'

function retentionKey(workspaceId: string) {
  return ['audit-retention', workspaceId] as const
}

export function useAuditEvents(params: MaybeRef<ListAuditEventsParams>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => ['audit', currentWorkspaceId.value ?? '', toValue(params)]),
    queryFn: () => auditApi.listAuditEvents(toValue(params)),
    enabled: computed(() => !!currentWorkspaceId.value),
  })
}

export function useAuditRetention() {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => retentionKey(currentWorkspaceId.value ?? '')),
    queryFn: () => auditApi.getAuditRetention(),
    enabled: computed(() => !!currentWorkspaceId.value),
  })
}

export function useUpdateAuditRetention() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: auditApi.updateAuditRetention,
    onSuccess: (retention) => {
      qc.setQueryData(retentionKey(currentWorkspaceId.value ?? ''), retention)
    },
  })
}
