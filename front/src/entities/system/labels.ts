import type { ExecutorKind } from './types'

export const executorKindLabels: Record<ExecutorKind, string> = {
  docker: 'Docker',
  k8s: 'Kubernetes',
  unknown: 'Unknown',
}
