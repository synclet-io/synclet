export type WorkflowRunStatus = 'running' | 'succeeded' | 'failed' | 'unknown'
export type WorkflowRunTrigger = 'schedule' | 'manual' | 'unknown'
export type WorkflowStepRunStatus = 'pending' | 'queued' | 'succeeded' | 'failed' | 'skipped' | 'unknown'

/** A step is one connection; it runs once the steps of afterConnectionIds have succeeded. */
export interface WorkflowStep {
  id: string
  connectionId: string
  afterConnectionIds: string[]
}

export interface Workflow {
  id: string
  workspaceId: string
  name: string
  /** Cron expression; empty when the workflow runs only when started by hand. */
  schedule: string
  nextScheduledAt: Date | undefined
  steps: WorkflowStep[]
  createdAt: Date | undefined
  updatedAt: Date | undefined
}

export interface WorkflowStepRun {
  id: string
  stepId: string
  connectionId: string
  status: WorkflowStepRunStatus
  jobId: string
  error: string
  startedAt: Date | undefined
  completedAt: Date | undefined
}

export interface WorkflowRun {
  id: string
  workflowId: string
  status: WorkflowRunStatus
  trigger: WorkflowRunTrigger
  startedAt: Date | undefined
  completedAt: Date | undefined
  stepRuns: WorkflowStepRun[]
}

export interface WorkflowStepInput {
  connectionId: string
  afterConnectionIds: string[]
}

export interface WorkflowInput {
  name: string
  schedule: string
  steps: WorkflowStepInput[]
}

export interface UpdateWorkflowInput extends WorkflowInput {
  id: string
}
