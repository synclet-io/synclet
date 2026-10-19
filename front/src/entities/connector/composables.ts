import type { MaybeRef, Ref } from 'vue'
import type { ManagedConnector } from './types'
import { useAuth } from '@entities/auth'
import { connectorKeys } from '@shared/lib/queryKeys'
import { useIsMutating, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
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
    queryKey: computed(() => connectorKeys.managedDetail(currentWorkspaceId.value ?? '', toValue(id))),
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

export function useConnectorSpec(id: MaybeRef<string | null | undefined>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => connectorKeys.spec(currentWorkspaceId.value ?? '', toValue(id) ?? '')),
    queryFn: () => connectorApi.getConnectorSpec(toValue(id)!),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(id)),
  })
}

export function useConnectorVersions(connectorImage: Ref<string>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => connectorKeys.versions(currentWorkspaceId.value ?? '', connectorImage.value)),
    queryFn: () => connectorApi.getConnectorVersions(connectorImage.value),
    enabled: computed(() => !!currentWorkspaceId.value && !!connectorImage.value),
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

const STAR_MUTATION_KEY = ['connector-star'] as const

export function useSetConnectorStar() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  const managedKey = () => [...connectorKeys.all(currentWorkspaceId.value ?? ''), 'managed']
  const setStarred = (id: string, starred: boolean) =>
    qc.setQueriesData<ManagedConnector[]>({ queryKey: managedKey() }, list =>
      list?.map(c => (c.id === id ? { ...c, starred } : c)))
  return useMutation({
    mutationKey: STAR_MUTATION_KEY,
    mutationFn: ({ id, starred }: { id: string, starred: boolean }) =>
      starred ? connectorApi.starManagedConnector(id) : connectorApi.unstarManagedConnector(id),
    onMutate: async ({ id, starred }) => {
      await qc.cancelQueries({ queryKey: managedKey() })
      setStarred(id, starred)
    },
    onError: (_err, { id, starred }) => { setStarred(id, !starred) },
    onSettled: () => qc.invalidateQueries({ queryKey: connectorKeys.all(currentWorkspaceId.value ?? '') }),
  })
}

export function useConnectorStarPending(id: MaybeRef<string>) {
  const count = useIsMutating({
    mutationKey: STAR_MUTATION_KEY,
    predicate: m => (m.state.variables as { id?: string } | undefined)?.id === toValue(id),
  })
  return computed(() => count.value > 0)
}
