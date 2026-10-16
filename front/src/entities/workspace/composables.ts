import type { MaybeRef, Ref } from 'vue'
import type { MemberRole } from './types'
import { useAuth } from '@entities/auth'
import { connectionKeys, destinationKeys, inviteKeys, sourceKeys, statsKeys, workspaceKeys } from '@shared/lib/queryKeys'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, toValue } from 'vue'
import * as workspaceApi from './api'

export function useWorkspaceMembers(workspaceId: MaybeRef<string>) {
  const { isAuthenticated } = useAuth()
  return useQuery({
    queryKey: computed(() => workspaceKeys.members(toValue(workspaceId))),
    queryFn: () => workspaceApi.listMembers(toValue(workspaceId)),
    enabled: computed(() => isAuthenticated.value && !!toValue(workspaceId)),
  })
}

export function useCreateWorkspace() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => workspaceApi.createWorkspace(name),
    onSuccess: () => { qc.invalidateQueries({ queryKey: workspaceKeys.all }) },
  })
}

export function useRemoveMember() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (params: { workspaceId: string, userId: string }) =>
      workspaceApi.removeMember(params.workspaceId, params.userId),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: workspaceKeys.members(variables.workspaceId) })
    },
  })
}

export function useUpdateMemberRole() {
  const qc = useQueryClient()
  const auth = useAuth()
  return useMutation({
    mutationFn: (params: { workspaceId: string, userId: string, role: MemberRole }) =>
      workspaceApi.updateMemberRole(params.workspaceId, params.userId, params.role),
    onSuccess: async (_data, variables) => {
      qc.invalidateQueries({ queryKey: workspaceKeys.members(variables.workspaceId) })
      if (variables.userId === auth.user.value?.id)
        await auth.reloadWorkspaces()
    },
  })
}

export function useInvites() {
  const { isAuthenticated } = useAuth()
  return useQuery({
    queryKey: computed(() => workspaceKeys.invites()),
    queryFn: () => workspaceApi.listInvites(),
    enabled: isAuthenticated,
  })
}

export function useCreateInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (params: { email: string, role: MemberRole }) => workspaceApi.createInvite(params),
    onSuccess: () => { qc.invalidateQueries({ queryKey: workspaceKeys.invites() }) },
  })
}

export function useAcceptInvite() {
  const qc = useQueryClient()
  const auth = useAuth()
  return useMutation({
    mutationFn: (token: string) => workspaceApi.acceptInvite(token),
    onSuccess: async ({ workspaceId }) => {
      qc.invalidateQueries({ queryKey: workspaceKeys.all })
      await auth.reloadWorkspaces()
      auth.switchWorkspace(workspaceId)
    },
  })
}

export function useDeclineInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (token: string) => workspaceApi.declineInvite(token),
    onSuccess: (_data, token) => { qc.invalidateQueries({ queryKey: inviteKeys.byToken(token) }) },
  })
}

export function useRevokeInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (inviteId: string) => workspaceApi.revokeInvite(inviteId),
    onSuccess: () => { qc.invalidateQueries({ queryKey: workspaceKeys.invites() }) },
  })
}

export function useResendInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (inviteId: string) => workspaceApi.resendInvite(inviteId),
    onSuccess: () => { qc.invalidateQueries({ queryKey: workspaceKeys.invites() }) },
  })
}

export function useInviteByToken(token: Ref<string>) {
  return useQuery({
    queryKey: computed(() => inviteKeys.byToken(token.value)),
    queryFn: () => workspaceApi.getInviteByToken(token.value),
    enabled: computed(() => !!token.value),
  })
}

export function useExportConfig() {
  return useMutation({
    mutationFn: () => workspaceApi.exportConfig(),
  })
}

export function useImportConfig() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: (file: File) => workspaceApi.importConfig(file),
    onSuccess: () => {
      const workspaceId = currentWorkspaceId.value ?? ''
      for (const queryKey of [
        sourceKeys.all(workspaceId),
        destinationKeys.all(workspaceId),
        connectionKeys.all(workspaceId),
        statsKeys.all(workspaceId),
      ]) {
        qc.invalidateQueries({ queryKey })
      }
    },
  })
}
