import type { MaybeRef, Ref } from 'vue'
import { useAuth } from '@entities/auth'
import { connectorKeys } from '@shared/lib/queryKeys'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, toValue } from 'vue'
import * as connectorApi from './api'

export function useAddConnector() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: connectorApi.addConnector,
    onSuccess: () => { qc.invalidateQueries({ queryKey: connectorKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}

export function useManagedConnector(id: MaybeRef<string>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => ['managed-connector', currentWorkspaceId.value ?? '', toValue(id)]),
    queryFn: () => connectorApi.getManagedConnector(toValue(id)),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(id)),
  })
}

export function useManagedConnectors(params?: {
  repositoryId?: Ref<string | null | undefined>
  search?: Ref<string | undefined>
}) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => connectorKeys.managed(
      currentWorkspaceId.value ?? '',
      params?.repositoryId?.value,
      params?.search?.value,
    )),
    queryFn: () => connectorApi.listManagedConnectors({
      repositoryId: params?.repositoryId?.value,
      search: params?.search?.value,
    }),
    enabled: computed(() => !!currentWorkspaceId.value),
  })
}

export function useDeleteManagedConnector() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: connectorApi.deleteManagedConnector,
    onSuccess: () => { qc.invalidateQueries({ queryKey: connectorKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}

export function useConnectorVersions(connectorImage: Ref<string>) {
  return useQuery({
    queryKey: computed(() => ['connector-versions', connectorImage.value]),
    queryFn: () => connectorApi.getConnectorVersions(connectorImage.value),
    enabled: computed(() => !!connectorImage.value),
    staleTime: 1000 * 60 * 60,
  })
}

export function useUpdateManagedConnector() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: (id: string) => connectorApi.updateManagedConnector(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: connectorKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}

export function useBatchUpdateConnectors() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: (connectorIds: string[]) => connectorApi.batchUpdateConnectors(connectorIds),
    onSuccess: () => { qc.invalidateQueries({ queryKey: connectorKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}
