import type { MaybeRef } from 'vue'
import type { ListAuditEventsParams } from './types'
import { useAuth } from '@entities/auth'
import { useQuery } from '@tanstack/vue-query'
import { computed, toValue } from 'vue'
import * as auditApi from './api'

export function useAuditEvents(params: MaybeRef<ListAuditEventsParams>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => ['audit', currentWorkspaceId.value ?? '', toValue(params)]),
    queryFn: () => auditApi.listAuditEvents(toValue(params)),
    enabled: computed(() => !!currentWorkspaceId.value),
  })
}
