import type { MaybeRef, Ref } from 'vue'
import type { ConfiguredStream } from './types'
import { useAuth } from '@entities/auth'
import { connectionKeys } from '@shared/lib/queryKeys'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, toValue } from 'vue'
import * as connectionApi from './api'

export function useStreamStates(connectionId: MaybeRef<string>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => connectionKeys.streamStates(currentWorkspaceId.value ?? '', toValue(connectionId))),
    queryFn: () => connectionApi.listStreamStates(toValue(connectionId)),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(connectionId)),
  })
}

export function useUpdateStreamState() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: (params: { connectionId: string, streamName: string, streamNamespace: string, stateData: string }) =>
      connectionApi.updateStreamState(params.connectionId, params.streamName, params.streamNamespace, params.stateData),
    onSuccess: (_data, params) => {
      qc.invalidateQueries({ queryKey: connectionKeys.streamStates(currentWorkspaceId.value ?? '', params.connectionId) })
    },
  })
}

export function useConnections(options?: { page?: Ref<number>, pageSize?: number }) {
  const { currentWorkspaceId } = useAuth()
  const page = options?.page
  const pageSize = options?.pageSize ?? 0
  return useQuery({
    queryKey: computed(() => [...connectionKeys.list(currentWorkspaceId.value ?? ''), page ? toValue(page) : 'all']),
    queryFn: () => connectionApi.listConnections(
      page ? { pageSize, offset: (toValue(page) - 1) * pageSize } : {},
    ),
    enabled: computed(() => !!currentWorkspaceId.value),
  })
}

export function useConnection(id: MaybeRef<string>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => connectionKeys.detail(currentWorkspaceId.value ?? '', toValue(id))),
    queryFn: () => connectionApi.getConnection(toValue(id)),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(id)),
  })
}

export function useCreateConnection() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: connectionApi.createConnection,
    onSuccess: () => { qc.invalidateQueries({ queryKey: connectionKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}

export function useDeleteConnection() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: connectionApi.deleteConnection,
    onSuccess: () => { qc.invalidateQueries({ queryKey: connectionKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}

export function useEnableConnection() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: connectionApi.enableConnection,
    onSuccess: () => { qc.invalidateQueries({ queryKey: connectionKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}

export function useDisableConnection() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: connectionApi.disableConnection,
    onSuccess: () => { qc.invalidateQueries({ queryKey: connectionKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}

export function useUpdateConnection() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: connectionApi.updateConnection,
    onSuccess: () => { qc.invalidateQueries({ queryKey: connectionKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}

export function useConfigureStreams() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: (params: { connectionId: string, streams: ConfiguredStream[] }) =>
      connectionApi.configureStreams(params.connectionId, params.streams),
    onSuccess: () => { qc.invalidateQueries({ queryKey: connectionKeys.all(currentWorkspaceId.value ?? '') }) },
  })
}
