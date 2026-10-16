import type { MaybeRef } from 'vue'
import type { ListAuditEventsParams } from './types'
import { useAuth } from '@entities/auth'
import { auditKeys } from '@shared/lib/queryKeys'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, toValue } from 'vue'
import * as auditApi from './api'

export function useAuditEvents(params: MaybeRef<ListAuditEventsParams>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => auditKeys.list(currentWorkspaceId.value ?? '', toValue(params))),
    queryFn: () => auditApi.listAuditEvents(toValue(params)),
    enabled: computed(() => !!currentWorkspaceId.value),
    placeholderData: keepPreviousData,
  })
}

export function useAuditRetention() {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => auditKeys.retention(currentWorkspaceId.value ?? '')),
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
      qc.setQueryData(auditKeys.retention(currentWorkspaceId.value ?? ''), retention)
    },
  })
}
