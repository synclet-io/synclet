import type { Job, JobStatus, JobType, SyncStats } from './types'
import type { Job as ProtoJob, JobAttempt as ProtoJobAttempt, SyncStats as ProtoSyncStats } from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'
import { jobClient } from '@shared/api/services'
import { tsToDate } from '@shared/lib/formatting'
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
    default: return 'unknown'
  }
}

const PROTO_JOB_STATUS: Record<JobStatus, ProtoJobStatus> = {
  scheduled: ProtoJobStatus.SCHEDULED,
  starting: ProtoJobStatus.STARTING,
  running: ProtoJobStatus.RUNNING,
  completed: ProtoJobStatus.COMPLETED,
  failed: ProtoJobStatus.FAILED,
  cancelled: ProtoJobStatus.CANCELLED,
  unknown: ProtoJobStatus.UNSPECIFIED,
}

function mapJobType(proto: ProtoJobType): JobType {
  switch (proto) {
    case ProtoJobType.SYNC: return 'sync'
    case ProtoJobType.DISCOVER: return 'discover'
    case ProtoJobType.CHECK: return 'check'
    default: return 'unknown'
  }
}

function mapSyncStats(proto: ProtoSyncStats | undefined): SyncStats | undefined {
  if (!proto)
    return undefined
  return {
    recordsRead: Number(proto.recordsRead),
    recordsFiltered: Number(proto.recordsFiltered),
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

export interface ListJobsParams {
  pageSize?: number
  offset?: number
  statuses?: readonly JobStatus[]
}

/** An empty `connectionId` lists jobs across every connection in the workspace. */
export async function listJobs(connectionId: string, params: ListJobsParams = {}): Promise<{ items: Job[], total: number }> {
  const res = await jobClient.listJobs({
    connectionId,
    pageSize: params.pageSize ?? 0,
    offset: params.offset ?? 0,
    statuses: (params.statuses ?? []).map(s => PROTO_JOB_STATUS[s]),
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

const LOG_PAGE_SIZE = 5000

/**
 * Without a `limit`, follows `hasMore` until every line after `afterId` is loaded,
 * one server-capped page per request.
 */
export async function getJobLogs(
  jobId: string,
  afterId: number = 0,
  limit: number = 0,
): Promise<JobLogPage> {
  if (limit > 0)
    return getJobLogsPage(jobId, afterId, limit)

  const lines: string[] = []
  let lastId = afterId
  for (;;) {
    const page = await getJobLogsPage(jobId, lastId, LOG_PAGE_SIZE)
    lines.push(...page.lines)
    if (page.lines.length > 0)
      lastId = page.lastId
    if (!page.hasMore || page.lines.length === 0)
      return { lines, lastId, hasMore: false, droppedLines: page.droppedLines }
  }
}

export interface JobLogPage {
  lines: string[]
  lastId: number
  hasMore: boolean
  /** Lines dropped by the per-attempt line cap, keyed by attempt number. */
  droppedLines: Record<number, number>
}

async function getJobLogsPage(jobId: string, afterId: number, limit: number): Promise<JobLogPage> {
  const res = await jobClient.getJobLogs({ jobId, afterId: BigInt(afterId), limit })
  const droppedLines: Record<number, number> = {}
  for (const a of res.attempts ?? [])
    droppedLines[a.attempt] = Number(a.droppedLines)
  return {
    lines: res.lines,
    lastId: Number(res.lastId),
    hasMore: res.hasMore,
    droppedLines,
  }
}

/** Largest attempt log, in bytes, that the browser collects in memory. */
export const MAX_DOWNLOAD_LOG_BYTES = 200 * 1024 * 1024

/**
 * Streams the full log of one attempt into a plain-text Blob. Throws and
 * cancels the stream once the log passes maxBytes.
 */
export async function downloadJobLogs(jobId: string, attempt: number, maxBytes = MAX_DOWNLOAD_LOG_BYTES): Promise<Blob> {
  const chunks: Uint8Array<ArrayBuffer>[] = []
  let size = 0
  for await (const res of jobClient.downloadJobLogs({ jobId, attempt })) {
    size += res.chunk.byteLength
    if (size > maxBytes) {
      const mb = Math.round(maxBytes / 1024 / 1024)
      throw new Error(`This log is larger than ${mb} MB and too large to download in the browser. Enable the log archive (filesystem or S3) and read the attempt's object from there.`)
    }
    chunks.push(new Uint8Array(res.chunk))
  }
  return new Blob(chunks, { type: 'text/plain' })
}
