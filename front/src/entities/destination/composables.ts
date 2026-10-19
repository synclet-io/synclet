import type { MaybeRef, Ref } from 'vue'
import { useAuth } from '@entities/auth'
import { connectionKeys, connectorKeys, destinationKeys, statsKeys } from '@shared/lib/queryKeys'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, toValue } from 'vue'
import * as destinationApi from './api'

export function useDestinations(options?: { page?: Ref<number>, pageSize?: number, enabled?: MaybeRef<boolean> }) {
  const { currentWorkspaceId } = useAuth()
  const page = options?.page
  const pageSize = options?.pageSize ?? 0
  return useQuery({
    queryKey: computed(() => [...destinationKeys.list(currentWorkspaceId.value ?? ''), page ? toValue(page) : 'all']),
    queryFn: () => destinationApi.listDestinations(
      page ? { pageSize, offset: (toValue(page) - 1) * pageSize } : {},
    ),
    enabled: computed(() => !!currentWorkspaceId.value && toValue(options?.enabled ?? true)),
  })
}

export function useDestination(id: MaybeRef<string>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => destinationKeys.detail(currentWorkspaceId.value ?? '', toValue(id))),
    queryFn: () => destinationApi.getDestination(toValue(id)),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(id)),
  })
}

export function useCreateDestination() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: destinationApi.createDestination,
    onSuccess: () => {
      const workspaceId = currentWorkspaceId.value ?? ''
      qc.invalidateQueries({ queryKey: destinationKeys.all(workspaceId) })
      qc.invalidateQueries({ queryKey: [...connectorKeys.all(workspaceId), 'managed'] })
    },
  })
}

export function useUpdateDestination() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: destinationApi.updateDestination,
    onSuccess: () => {
      const workspaceId = currentWorkspaceId.value ?? ''
      qc.invalidateQueries({ queryKey: destinationKeys.all(workspaceId) })
      qc.invalidateQueries({ queryKey: connectionKeys.all(workspaceId) })
    },
  })
}

export function useDeleteDestination() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: destinationApi.deleteDestination,
    onSuccess: () => {
      const workspaceId = currentWorkspaceId.value ?? ''
      qc.invalidateQueries({ queryKey: destinationKeys.all(workspaceId) })
      qc.invalidateQueries({ queryKey: connectionKeys.all(workspaceId) })
      qc.invalidateQueries({ queryKey: statsKeys.all(workspaceId) })
      qc.invalidateQueries({ queryKey: [...connectorKeys.all(workspaceId), 'managed'] })
    },
  })
}

export function useTestDestinationConnection() {
  return useMutation({
    mutationFn: destinationApi.testConnection,
  })
}
