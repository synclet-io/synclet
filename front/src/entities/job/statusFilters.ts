import type { JobStatus } from './types'

export type JobStatusFilterValue = '' | 'active' | 'completed' | 'failed' | 'cancelled'

export interface JobStatusFilter {
  value: JobStatusFilterValue
  label: string
  statuses: readonly JobStatus[]
}

export const JOB_STATUS_FILTERS: readonly JobStatusFilter[] = [
  { value: '', label: 'All', statuses: [] },
  { value: 'active', label: 'Active', statuses: ['scheduled', 'starting', 'running'] },
  { value: 'completed', label: 'Completed', statuses: ['completed'] },
  { value: 'failed', label: 'Failed', statuses: ['failed'] },
  { value: 'cancelled', label: 'Cancelled', statuses: ['cancelled'] },
]
