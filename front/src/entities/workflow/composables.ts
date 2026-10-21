import type { MaybeRef } from 'vue'
import { useAuth } from '@entities/auth'
import { connectionKeys, workflowKeys } from '@shared/lib/queryKeys'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, toValue } from 'vue'
import * as workflowApi from './api'

const ACTIVE_RUN_POLL_MS = 3000

export function useWorkflows() {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => workflowKeys.list(currentWorkspaceId.value ?? '')),
    queryFn: () => workflowApi.listWorkflows(),
    enabled: computed(() => !!currentWorkspaceId.value),
  })
}

export function useWorkflow(id: MaybeRef<string>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => workflowKeys.detail(currentWorkspaceId.value ?? '', toValue(id))),
    queryFn: () => workflowApi.getWorkflow(toValue(id)),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(id)),
  })
}

/** Runs of a workflow, newest first; polled while one of them is running. */
export function useWorkflowRuns(workflowId: MaybeRef<string>) {
  const { currentWorkspaceId } = useAuth()
  return useQuery({
    queryKey: computed(() => workflowKeys.runs(currentWorkspaceId.value ?? '', toValue(workflowId))),
    queryFn: () => workflowApi.listWorkflowRuns(toValue(workflowId)),
    enabled: computed(() => !!currentWorkspaceId.value && !!toValue(workflowId)),
    refetchInterval: query => (query.state.data?.some(run => run.status === 'running') ? ACTIVE_RUN_POLL_MS : false),
  })
}

export function useCreateWorkflow() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: workflowApi.createWorkflow,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: workflowKeys.all(currentWorkspaceId.value ?? '') })
      qc.invalidateQueries({ queryKey: connectionKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}

export function useUpdateWorkflow() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: workflowApi.updateWorkflow,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: workflowKeys.all(currentWorkspaceId.value ?? '') })
      qc.invalidateQueries({ queryKey: connectionKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}

export function useDeleteWorkflow() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: workflowApi.deleteWorkflow,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: workflowKeys.all(currentWorkspaceId.value ?? '') })
      qc.invalidateQueries({ queryKey: connectionKeys.all(currentWorkspaceId.value ?? '') })
    },
  })
}

export function useRunWorkflow() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: workflowApi.runWorkflow,
    onSuccess: (run) => {
      qc.invalidateQueries({ queryKey: workflowKeys.runs(currentWorkspaceId.value ?? '', run.workflowId) })
    },
  })
}

export function useRerunWorkflowRun() {
  const qc = useQueryClient()
  const { currentWorkspaceId } = useAuth()
  return useMutation({
    mutationFn: workflowApi.rerunWorkflowRun,
    onSuccess: (run) => {
      qc.invalidateQueries({ queryKey: workflowKeys.runs(currentWorkspaceId.value ?? '', run.workflowId) })
    },
  })
}
