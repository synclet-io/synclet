export type ExecutorKind = 'docker' | 'k8s' | 'unknown'

/** Resources an executor gives a sync container with no runtime config of its own. */
export interface ExecutorResourceDefaults {
  cpuRequest: string
  cpuLimit: string
  memoryRequest: string
  memoryLimit: string
  /** The CPU values are the executor's own fallback. */
  cpuExecutorDefault: boolean
  /** The memory values are the executor's own fallback. */
  memoryExecutorDefault: boolean
}

export interface ExecutorTypeDefaults {
  type: ExecutorKind
  /** Concurrent syncs of one instance; 0 means unlimited. */
  capacity: number
  resourceDefaults: ExecutorResourceDefaults
}

export interface ExecutorInstance {
  id: string
  type: ExecutorKind
  version: string
  /** Concurrent syncs; 0 means unlimited. */
  capacity: number
  nativeDestinations: string[]
  resourceDefaults: ExecutorResourceDefaults
  firstSeenAt: Date | undefined
  lastSeenAt: Date | undefined
}

export interface RuntimeDefaults {
  cpuRequest: string
  cpuLimit: string
  memoryRequest: string
  memoryLimit: string
  serviceAccountName: string
  maxConcurrentJobs: number
  idleTimeoutSeconds: number
  logArchiveBackend: string
  /** One entry per executor type with a live executor. */
  executors: ExecutorTypeDefaults[]
}

export interface SystemInfo {
  registrationEnabled: boolean
  workspacesMode: 'single' | 'multi'
  emailEnabled: boolean
}
