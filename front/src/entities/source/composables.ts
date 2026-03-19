import type { MaybeRef, Ref } from 'vue'
import { useAuth } from '@entities/auth'
import { sourceKeys } from '@shared/lib/queryKeys'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, toValue } from 'vue'
import * as sourceApi from './api'

export function useSources(options?: { page?: Ref<number>, pageSize?: number }) {
  const { currentWorkspaceId } = useAuth()
  const page = options?.page
  const pageSize = options?.pageSize ?? 0
  return useQuery({
    queryKey: computed(() => [...sourceKeys.list(currentWorkspaceId.value ?? ''), page ? toValue(page) : 'all']),
    queryFn: () => sourceApi.listSources(
      page ? { pageSize, offset: (toValue(page) - 1) * pageSize } : {},
    ),
    enabled: computed(() => !!currentWorkspaceId.value),
  })
}

export function useSource(id: MaybeRef<string>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => sourceKeys.detail(currentWorkspaceId.value ?? '', toValue(id))),
    queryFn: () => sourceApi.getSource(toValue(id)),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(id)),
  })
}

export function useCreateSource() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: sourceApi.createSource,
    onSuccess: () => { qc.invalidateQueries({ queryKey: sourceKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}

export function useUpdateSource() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: sourceApi.updateSource,
    onSuccess: () => { qc.invalidateQueries({ queryKey: sourceKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}

export function useDeleteSource() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: sourceApi.deleteSource,
    onSuccess: () => { qc.invalidateQueries({ queryKey: sourceKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}

export function useTestSourceConnection() {
  return useMutation({
    mutationFn: sourceApi.testConnection,
  })
}

export function useSourceCatalog(sourceId: MaybeRef<string>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => [...sourceKeys.detail(currentWorkspaceId.value ?? '', toValue(sourceId)), 'catalog']),
    queryFn: () => sourceApi.getSourceCatalog(toValue(sourceId)),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(sourceId)),
  })
}
