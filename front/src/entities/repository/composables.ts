import { useAuth } from '@entities/auth'
import { connectorKeys, repositoryKeys } from '@shared/lib/queryKeys'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed } from 'vue'
import * as repositoryApi from './api'

export function useRepositories() {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => repositoryKeys.list(currentWorkspaceId.value ?? '')),
    queryFn: () => repositoryApi.listRepositories(),
    enabled: computed(() => !!currentWorkspaceId.value),
  })
}

export function useAddRepository() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: repositoryApi.addRepository,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: repositoryKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}

export function useDeleteRepository() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: repositoryApi.deleteRepository,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: repositoryKeys.all(currentWorkspaceId.value ?? '') })
      qc.invalidateQueries({ queryKey: connectorKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}

export function useSyncRepository() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: repositoryApi.syncRepository,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: repositoryKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}
