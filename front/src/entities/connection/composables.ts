import type { MaybeRef, Ref } from 'vue'
import type { ConfiguredStream } from './types'
import { useAuth } from '@entities/auth'
import { useConnectorTaskResult } from '@entities/connector-task'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { connectionKeys } from '@shared/lib/queryKeys'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, ref, toValue, watch } from 'vue'
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

export function useResetStreamState() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: (params: { connectionId: string, streamNamespace: string, streamName: string }) =>
      connectionApi.resetStreamState(params.connectionId, params.streamNamespace, params.streamName),
    onSuccess: (_data, params) => {
      qc.invalidateQueries({ queryKey: connectionKeys.streamStates(currentWorkspaceId.value ?? '', params.connectionId) })
    },
  })
}

export function useResetConnectionState() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: (connectionId: string) => connectionApi.resetConnectionState(connectionId),
    onSuccess: (_data, connectionId) => {
      qc.invalidateQueries({ queryKey: connectionKeys.streamStates(currentWorkspaceId.value ?? '', connectionId) })
    },
  })
}

export function useConnections(options?: {
  page?: Ref<number>
  pageSize?: number
  includePendingSchemaChanges?: boolean
  sourceId?: MaybeRef<string>
  destinationId?: MaybeRef<string>
  tags?: MaybeRef<string[]>
  enabled?: MaybeRef<boolean>
}) {
  const { currentWorkspaceId } = useAuth()
  const page = options?.page
  const pageSize = options?.pageSize ?? 0
  const includePendingSchemaChanges = options?.includePendingSchemaChanges ?? false
  const sourceId = computed(() => toValue(options?.sourceId) ?? '')
  const destinationId = computed(() => toValue(options?.destinationId) ?? '')
  const tags = computed(() => toValue(options?.tags) ?? [])
  return useQuery({
    queryKey: computed(() => [
      ...connectionKeys.list(currentWorkspaceId.value ?? ''),
      page ? toValue(page) : 'all',
      ...(includePendingSchemaChanges ? ['pendingSchemaChanges'] : []),
      ...(sourceId.value || destinationId.value ? [{ sourceId: sourceId.value, destinationId: destinationId.value }] : []),
      ...(tags.value.length ? [{ tags: tags.value }] : []),
    ]),
    queryFn: () => connectionApi.listConnections({
      ...(page ? { pageSize, offset: (toValue(page) - 1) * pageSize } : {}),
      includePendingSchemaChanges,
      sourceId: sourceId.value,
      destinationId: destinationId.value,
      tags: tags.value,
    }),
    enabled: computed(() => !!currentWorkspaceId.value && toValue(options?.enabled ?? true)),
  })
}

export function useConnectionTags() {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => connectionKeys.tags(currentWorkspaceId.value ?? '')),
    queryFn: () => connectionApi.listConnectionTags(),
    enabled: computed(() => !!currentWorkspaceId.value),
  })
}

export function useConfiguredCatalog(connectionId: MaybeRef<string>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => connectionKeys.configuredCatalog(currentWorkspaceId.value ?? '', toValue(connectionId))),
    queryFn: () => connectionApi.getConfiguredCatalog(toValue(connectionId)),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(connectionId)),
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

export function useSchemaChanges(connectionId: MaybeRef<string>, options?: { enabled?: MaybeRef<boolean> }) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => connectionKeys.schemaChanges(currentWorkspaceId.value ?? '', toValue(connectionId))),
    queryFn: () => connectionApi.getSchemaChanges(toValue(connectionId)),
    enabled: computed(() => {
      const baseEnabled = !!currentWorkspaceId.value && !!toValue(connectionId)
      if (options?.enabled === undefined)
        return baseEnabled
      return baseEnabled && toValue(options.enabled)
    }),
    staleTime: 30_000,
  })
}

export function useDiscoverSchemaTask(
  connectionId: MaybeRef<string>,
  callbacks?: { onCompleted?: () => void, onFailed?: (message: string) => void },
) {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  const taskId = ref<string | null>(null)
  const discoveringFor = ref('')
  const task = useConnectorTaskResult(taskId)
  const start = useMutation({
    mutationFn: (id: string) => connectionApi.discoverSchema(id),
  })

  function finish(failure?: string) {
    const id = discoveringFor.value
    taskId.value = null
    if (failure !== undefined) {
      callbacks?.onFailed?.(failure)
      return
    }
    const workspaceId = currentWorkspaceId.value ?? ''
    qc.invalidateQueries({ queryKey: connectionKeys.schemaChanges(workspaceId, id) })
    qc.invalidateQueries({ queryKey: connectionKeys.streamStates(workspaceId, id) })
    callbacks?.onCompleted?.()
  }

  watch([task.data, task.isError, task.timedOut], ([data, isError, timedOut]) => {
    if (!taskId.value)
      return
    if (data?.status === 'completed')
      finish()
    else if (data?.status === 'failed')
      finish(data.errorMessage || 'Schema discovery failed')
    else if (isError)
      finish(getErrorMessage(task.error.value))
    else if (timedOut)
      finish('Schema discovery is taking longer than expected. Check the job logs and try again.')
  })

  async function discover() {
    const id = toValue(connectionId)
    try {
      const res = await start.mutateAsync(id)
      discoveringFor.value = id
      taskId.value = res.taskId
    }
    catch (e: unknown) {
      callbacks?.onFailed?.(getErrorMessage(e))
    }
  }

  const isDiscovering = computed(() => start.isPending.value || !!taskId.value)

  return { discover, isDiscovering }
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
