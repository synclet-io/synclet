import { useAuth } from '@entities/auth'
import { systemKeys } from '@shared/lib/queryKeys'
import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import * as systemApi from './api'

export function useSystemInfo() {
  return useQuery({
    queryKey: systemKeys.info(),
    queryFn: () => systemApi.getSystemInfo(),
    staleTime: Infinity, // execution mode doesn't change at runtime
  })
}

export function useRuntimeDefaults() {
  const auth = useAuth()
  return useQuery({
    queryKey: systemKeys.runtimeDefaults(),
    queryFn: () => systemApi.getRuntimeDefaults(),
    enabled: computed(() => auth.isAuthenticated.value && !!auth.currentWorkspaceId.value),
    staleTime: Infinity,
  })
}
