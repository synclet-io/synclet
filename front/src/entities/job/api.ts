import type { Job, JobStatus, JobType, SyncStats } from './types'
import { jobClient } from '@shared/api/services'
import { tsToDate } from '@shared/lib/formatting'
import type { Job as ProtoJob, JobAttempt as ProtoJobAttempt, SyncStats as ProtoSyncStats } from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'
import {
  JobStatus as ProtoJobStatus,
  JobType as ProtoJobType,
} from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'

function mapJobStatus(proto: ProtoJobStatus): JobStatus {
  switch (proto) {
    case ProtoJobStatus.SCHEDULED: return 'scheduled'
    case ProtoJobStatus.STARTING: return 'starting'
    case ProtoJobStatus.RUNNING: return 'running'
    case ProtoJobStatus.COMPLETED: return 'completed'
    case ProtoJobStatus.FAILED: return 'failed'
    case ProtoJobStatus.CANCELLED: return 'cancelled'
    default: return 'scheduled'
  }
}

function mapJobType(proto: ProtoJobType): JobType {
  switch (proto) {
    case ProtoJobType.SYNC: return 'sync'
    case ProtoJobType.DISCOVER: return 'discover'
    case ProtoJobType.CHECK: return 'check'
    default: return 'sync'
  }
}

function mapSyncStats(proto: ProtoSyncStats | undefined): SyncStats | undefined {
  if (!proto)
    return undefined
  return {
    recordsRead: Number(proto.recordsRead),
    bytesSynced: Number(proto.bytesSynced),
    durationMs: Number(proto.durationMs),
  }
}

function mapAttempt(proto: ProtoJobAttempt) {
  return {
    id: proto.id as string,
    attemptNumber: proto.attemptNumber as number,
    startedAt: tsToDate(proto.startedAt),
    completedAt: tsToDate(proto.completedAt),
    error: proto.error as string,
    syncStats: mapSyncStats(proto.syncStats),
  }
}

function mapJob(proto: ProtoJob): Job {
  return {
    id: proto.id,
    connectionId: proto.connectionId,
    status: mapJobStatus(proto.status),
    jobType: mapJobType(proto.jobType),
    scheduledAt: tsToDate(proto.scheduledAt),
    startedAt: tsToDate(proto.startedAt),
    completedAt: tsToDate(proto.completedAt),
    error: proto.error,
    attempt: proto.attempt,
    maxAttempts: proto.maxAttempts,
    createdAt: tsToDate(proto.createdAt),
    attempts: (proto.attempts || []).map(mapAttempt),
  }
}

export async function listJobs(connectionId: string, params: { pageSize?: number, offset?: number } = {}): Promise<{ items: Job[], total: number }> {
  const res = await jobClient.listJobs({
    connectionId,
    pageSize: params.pageSize ?? 0,
    offset: params.offset ?? 0,
  })
  return {
    items: (res.jobs || []).map(mapJob),
    total: Number(res.total),
  }
}

export async function getJob(id: string): Promise<Job | undefined> {
  const res = await jobClient.getJob({ jobId: id })
  return res.job ? mapJob(res.job) : undefined
}

export async function cancelJob(jobId: string): Promise<void> {
  await jobClient.cancelJob({ jobId })
}

export async function triggerSync(connectionId: string): Promise<Job | undefined> {
  const res = await jobClient.triggerSync({ connectionId })
  return res.job ? mapJob(res.job) : undefined
}

export async function getJobLogs(
  jobId: string,
  afterId: number = 0,
  limit: number = 0,
): Promise<{ lines: string[], lastId: number, hasMore: boolean }> {
  const res = await jobClient.getJobLogs({ jobId, afterId: BigInt(afterId), limit })
  return {
    lines: res.lines,
    lastId: Number(res.lastId),
    hasMore: res.hasMore,
  }
}
