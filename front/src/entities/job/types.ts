export type JobStatus = 'scheduled' | 'starting' | 'running' | 'completed' | 'failed' | 'cancelled'
export type JobType = 'sync' | 'discover' | 'check'

export interface Job {
  id: string
  connectionId: string
  status: JobStatus
  jobType: JobType
  scheduledAt: Date | undefined
  startedAt: Date | undefined
  completedAt: Date | undefined
  error: string
  attempt: number
  maxAttempts: number
  createdAt: Date | undefined
  attempts: JobAttempt[]
}

export interface JobAttempt {
  id: string
  attemptNumber: number
  startedAt: Date | undefined
  completedAt: Date | undefined
  error: string
  syncStats: SyncStats | undefined
}

export interface SyncStats {
  recordsRead: number
  bytesSynced: number
  durationMs: number
}
