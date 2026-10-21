import type { UpdateWorkflowInput, Workflow, WorkflowInput, WorkflowRun, WorkflowRunStatus, WorkflowRunTrigger, WorkflowStepRun, WorkflowStepRunStatus } from './types'
import type { Workflow as ProtoWorkflow, WorkflowRun as ProtoWorkflowRun, WorkflowStepRun as ProtoWorkflowStepRun } from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'
import { workflowClient } from '@shared/api/services'
import { requireField } from '@shared/lib/errorUtils'
import { tsToDate } from '@shared/lib/formatting'
import {
  WorkflowRunStatus as ProtoWorkflowRunStatus,
  WorkflowRunTrigger as ProtoWorkflowRunTrigger,
  WorkflowStepRunStatus as ProtoWorkflowStepRunStatus,
} from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'

function mapRunStatus(proto: ProtoWorkflowRunStatus): WorkflowRunStatus {
  switch (proto) {
    case ProtoWorkflowRunStatus.RUNNING: return 'running'
    case ProtoWorkflowRunStatus.SUCCEEDED: return 'succeeded'
    case ProtoWorkflowRunStatus.FAILED: return 'failed'
    default: return 'unknown'
  }
}

function mapRunTrigger(proto: ProtoWorkflowRunTrigger): WorkflowRunTrigger {
  switch (proto) {
    case ProtoWorkflowRunTrigger.SCHEDULE: return 'schedule'
    case ProtoWorkflowRunTrigger.MANUAL: return 'manual'
    default: return 'unknown'
  }
}

function mapStepRunStatus(proto: ProtoWorkflowStepRunStatus): WorkflowStepRunStatus {
  switch (proto) {
    case ProtoWorkflowStepRunStatus.PENDING: return 'pending'
    case ProtoWorkflowStepRunStatus.QUEUED: return 'queued'
    case ProtoWorkflowStepRunStatus.SUCCEEDED: return 'succeeded'
    case ProtoWorkflowStepRunStatus.FAILED: return 'failed'
    case ProtoWorkflowStepRunStatus.SKIPPED: return 'skipped'
    default: return 'unknown'
  }
}

export function mapWorkflow(proto: ProtoWorkflow): Workflow {
  return {
    id: proto.id,
    workspaceId: proto.workspaceId,
    name: proto.name,
    schedule: proto.schedule ?? '',
    nextScheduledAt: tsToDate(proto.nextScheduledAt),
    steps: (proto.steps ?? []).map(s => ({
      id: s.id,
      connectionId: s.connectionId,
      afterConnectionIds: [...(s.afterConnectionIds ?? [])],
    })),
    createdAt: tsToDate(proto.createdAt),
    updatedAt: tsToDate(proto.updatedAt),
  }
}

function mapStepRun(proto: ProtoWorkflowStepRun): WorkflowStepRun {
  return {
    id: proto.id,
    stepId: proto.stepId,
    connectionId: proto.connectionId,
    status: mapStepRunStatus(proto.status),
    jobId: proto.jobId ?? '',
    error: proto.error ?? '',
    startedAt: tsToDate(proto.startedAt),
    completedAt: tsToDate(proto.completedAt),
  }
}

export function mapWorkflowRun(proto: ProtoWorkflowRun): WorkflowRun {
  return {
    id: proto.id,
    workflowId: proto.workflowId,
    status: mapRunStatus(proto.status),
    trigger: mapRunTrigger(proto.trigger),
    startedAt: tsToDate(proto.startedAt),
    completedAt: tsToDate(proto.completedAt),
    stepRuns: (proto.stepRuns ?? []).map(mapStepRun),
  }
}

function toProtoDefinition(input: WorkflowInput) {
  const schedule = input.schedule.trim()
  return {
    name: input.name,
    ...(schedule ? { schedule } : {}),
    steps: input.steps.map(s => ({ connectionId: s.connectionId, afterConnectionIds: s.afterConnectionIds })),
  }
}

export async function listWorkflows(): Promise<Workflow[]> {
  const res = await workflowClient.listWorkflows({})
  return (res.workflows ?? []).map(mapWorkflow)
}

export async function getWorkflow(id: string): Promise<Workflow> {
  const res = await workflowClient.getWorkflow({ id })
  return mapWorkflow(requireField(res.workflow, 'workflow'))
}

export async function createWorkflow(input: WorkflowInput): Promise<Workflow> {
  const res = await workflowClient.createWorkflow(toProtoDefinition(input))
  return mapWorkflow(requireField(res.workflow, 'workflow'))
}

export async function updateWorkflow(input: UpdateWorkflowInput): Promise<Workflow> {
  const res = await workflowClient.updateWorkflow({ id: input.id, ...toProtoDefinition(input) })
  return mapWorkflow(requireField(res.workflow, 'workflow'))
}

export async function deleteWorkflow(id: string): Promise<void> {
  await workflowClient.deleteWorkflow({ id })
}

export async function runWorkflow(id: string): Promise<WorkflowRun> {
  const res = await workflowClient.runWorkflow({ id })
  return mapWorkflowRun(requireField(res.run, 'run'))
}

export async function rerunWorkflowRun(runId: string): Promise<WorkflowRun> {
  const res = await workflowClient.rerunWorkflowRun({ runId })
  return mapWorkflowRun(requireField(res.run, 'run'))
}

export async function listWorkflowRuns(workflowId: string): Promise<WorkflowRun[]> {
  const res = await workflowClient.listWorkflowRuns({ workflowId })
  return (res.runs ?? []).map(mapWorkflowRun)
}
