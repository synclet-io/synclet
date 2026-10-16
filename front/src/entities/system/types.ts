export type ExecutorKind = 'docker' | 'k8s' | 'external'

export interface RuntimeDefaults {
  executor: ExecutorKind
  cpuRequest: string
  cpuLimit: string
  memoryRequest: string
  memoryLimit: string
  serviceAccountName: string
  maxConcurrentJobs: number
  idleTimeoutSeconds: number
  logArchiveBackend: string
  dockerMaxConcurrentJobs: number
}

export interface SystemInfo {
  registrationEnabled: boolean
  workspacesMode: 'single' | 'multi'
  emailEnabled: boolean
}
