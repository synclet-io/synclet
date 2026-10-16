import { beforeEach, describe, expect, it, vi } from 'vitest'
import { JobStatus as ProtoJobStatus } from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'
import { getJobLogs, listJobs } from '../api'

const jobClient = vi.hoisted(() => ({
  listJobs: vi.fn(),
  getJobLogs: vi.fn(),
}))

vi.mock('@shared/api/services', () => ({
  jobClient,
}))

describe('job api', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('sends connection, status filter and page to listJobs', async () => {
    jobClient.listJobs.mockResolvedValue({ jobs: [], total: 3 })

    const result = await listJobs('', { statuses: ['running', 'failed'], pageSize: 25, offset: 50 })

    expect(jobClient.listJobs).toHaveBeenCalledWith({
      connectionId: '',
      pageSize: 25,
      offset: 50,
      statuses: [ProtoJobStatus.RUNNING, ProtoJobStatus.FAILED],
    })
    expect(result).toEqual({ items: [], total: 3 })
  })

  it('maps unrecognised status and type to unknown', async () => {
    jobClient.listJobs.mockResolvedValue({ jobs: [{ id: 'j1', status: 99, jobType: 99, attempts: [] }], total: 1 })

    const { items } = await listJobs('')

    expect(items[0]).toMatchObject({ status: 'unknown', jobType: 'unknown' })
  })

  it('follows hasMore until the whole log is loaded', async () => {
    jobClient.getJobLogs
      .mockResolvedValueOnce({ lines: ['a', 'b'], lastId: 2n, hasMore: true })
      .mockResolvedValueOnce({ lines: ['c'], lastId: 3n, hasMore: false })

    const result = await getJobLogs('job-1', 0)

    expect(jobClient.getJobLogs).toHaveBeenCalledTimes(2)
    expect(jobClient.getJobLogs.mock.calls[1][0]).toEqual(expect.objectContaining({ jobId: 'job-1', afterId: 2n }))
    expect(result).toEqual({ lines: ['a', 'b', 'c'], lastId: 3, hasMore: false })
  })

  it('keeps the cursor when no new lines arrive', async () => {
    jobClient.getJobLogs.mockResolvedValueOnce({ lines: [], lastId: 0n, hasMore: false })

    const result = await getJobLogs('job-1', 7)

    expect(result).toEqual({ lines: [], lastId: 7, hasMore: false })
  })

  it('returns a single page when a limit is given', async () => {
    jobClient.getJobLogs.mockResolvedValueOnce({ lines: ['a'], lastId: 1n, hasMore: true })

    const result = await getJobLogs('job-1', 0, 1)

    expect(jobClient.getJobLogs).toHaveBeenCalledTimes(1)
    expect(result).toEqual({ lines: ['a'], lastId: 1, hasMore: true })
  })
})
