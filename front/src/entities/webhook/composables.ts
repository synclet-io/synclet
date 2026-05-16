import { useAuth } from '@entities/auth'
import { webhookKeys } from '@shared/lib/queryKeys'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed } from 'vue'
import * as webhookApi from './api'

export function useWebhooks() {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => webhookKeys.list(currentWorkspaceId.value ?? '')),
    queryFn: () => webhookApi.listWebhooks(),
    enabled: computed(() => !!currentWorkspaceId.value),
  })
}

export function useCreateWebhook() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: webhookApi.createWebhook,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: webhookKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}

export function useUpdateWebhook() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: webhookApi.updateWebhook,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: webhookKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}

export function useDeleteWebhook() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: webhookApi.deleteWebhook,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: webhookKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}

export function useTestWebhook() {
  return useMutation({
    mutationFn: webhookApi.testWebhook,
  })
}
