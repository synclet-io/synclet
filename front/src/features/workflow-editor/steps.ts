import type { WorkflowStepInput } from '@entities/workflow'

export function addStep(steps: WorkflowStepInput[], connectionId: string): WorkflowStepInput[] {
  if (!connectionId || steps.some(s => s.connectionId === connectionId))
    return steps
  return [...steps, { connectionId, afterConnectionIds: [] }]
}

/** Removes the step and drops it from the "after" list of every other step. */
export function removeStep(steps: WorkflowStepInput[], connectionId: string): WorkflowStepInput[] {
  return steps
    .filter(s => s.connectionId !== connectionId)
    .map(s => ({ ...s, afterConnectionIds: s.afterConnectionIds.filter(id => id !== connectionId) }))
}

export function setAfter(steps: WorkflowStepInput[], connectionId: string, afterConnectionIds: string[]): WorkflowStepInput[] {
  return steps.map(s => (s.connectionId === connectionId ? { ...s, afterConnectionIds } : s))
}
