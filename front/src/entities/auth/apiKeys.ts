import { apiKeyKeys } from '@shared/lib/queryKeys'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed } from 'vue'
import * as authApi from './api'
import { useAuth } from './composables'

export function useAPIKeys() {
  const { isAuthenticated, currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => apiKeyKeys.list(currentWorkspaceId.value ?? '')),
    queryFn: () => authApi.listAPIKeys(currentWorkspaceId.value ?? ''),
    enabled: computed(() => isAuthenticated.value && !!currentWorkspaceId.value),
  })
}

export function useCreateAPIKey() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: (name: string) => authApi.createAPIKey(currentWorkspaceId.value ?? '', name),
    onSuccess: () => { qc.invalidateQueries({ queryKey: apiKeyKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}

export function useRevokeAPIKey() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: (id: string) => authApi.revokeAPIKey(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: apiKeyKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}
