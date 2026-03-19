import type { Ref } from 'vue'
import type { NotificationCondition } from './types'
import { useAuth } from '@entities/auth'
import { notificationKeys } from '@shared/lib/queryKeys'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, toValue } from 'vue'
import * as notificationApi from './api'

export function useNotificationChannels() {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => notificationKeys.channels(currentWorkspaceId.value ?? '')),
    queryFn: () => notificationApi.listChannels(),
    enabled: computed(() => !!currentWorkspaceId.value),
  })
}

export function useNotificationRules(connectionId?: Ref<string | undefined>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => notificationKeys.rules(currentWorkspaceId.value ?? '', toValue(connectionId))),
    queryFn: () => notificationApi.listRules({
      connectionId: toValue(connectionId) || undefined,
    }),
    enabled: computed(() => !!currentWorkspaceId.value),
  })
}

export function useCreateChannel() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: notificationApi.createChannel,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: notificationKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}

export function useUpdateChannel() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: (params: { id: string, name: string, config: Record<string, string>, enabled: boolean }) =>
      notificationApi.updateChannel(params.id, { name: params.name, config: params.config, enabled: params.enabled }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: notificationKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}

export function useDeleteChannel() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: notificationApi.deleteChannel,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: notificationKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}

export function useTestChannel() {
  return useMutation({
    mutationFn: notificationApi.testChannel,
  })
}

export function useCreateRule() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: notificationApi.createRule,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: notificationKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}

export function useUpdateRule() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: (params: { id: string, condition: NotificationCondition, conditionValue: number, enabled: boolean }) =>
      notificationApi.updateRule(params.id, { condition: params.condition, conditionValue: params.conditionValue, enabled: params.enabled }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: notificationKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}

export function useDeleteRule() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: notificationApi.deleteRule,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: notificationKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}
