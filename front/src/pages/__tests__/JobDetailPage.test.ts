import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import JobDetailPage from '../JobDetailPage.vue'

const JOB_ID = '0f9e8d7c-6b5a-4f3e-9d2c-1b0a9f8e7d6c'

const mocks = vi.hoisted(() => ({
  droppedLines: {} as Record<number, number>,
  downloadJobLogs: vi.fn(),
}))

vi.mock('@entities/job', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@entities/job')>()
  return {
    ...actual,
    useJob: () => ({
      data: ref({
        id: JOB_ID,
        connectionId: 'conn-1',
        status: 'completed',
        jobType: 'sync',
        attempt: 2,
        maxAttempts: 3,
        error: '',
        startedAt: new Date('2026-10-17T10:00:00Z'),
        completedAt: new Date('2026-10-17T10:05:00Z'),
        attempts: [
          { id: 'a1', attemptNumber: 1, startedAt: new Date('2026-10-17T10:00:00Z'), completedAt: new Date('2026-10-17T10:01:00Z'), error: 'boom' },
          { id: 'a2', attemptNumber: 2, startedAt: new Date('2026-10-17T10:02:00Z'), completedAt: new Date('2026-10-17T10:05:00Z'), error: '' },
        ],
      }),
      isLoading: ref(false),
      error: ref(null),
    }),
    useJobLogs: () => ({
      lines: ref([]),
      droppedLines: ref(mocks.droppedLines),
      isLoading: ref(false),
    }),
  }
})
vi.mock('@entities/job/api', () => ({
  downloadJobLogs: mocks.downloadJobLogs,
}))

async function mountPage() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/jobs/:id', name: 'job-detail', component: JobDetailPage },
      { path: '/jobs', name: 'jobs', component: { template: '<div />' } },
      { path: '/connections/:id', name: 'connection-detail', component: { template: '<div />' } },
    ],
  })
  await router.push(`/jobs/${JOB_ID}`)
  await router.isReady()
  const wrapper = mount(JobDetailPage, { global: { plugins: [router] } })
  await flushPromises()
  return wrapper
}

describe('jobDetailPage logs', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.droppedLines = {}
  })

  it('shows dropped lines for the attempts that dropped any', async () => {
    mocks.droppedLines = { 1: 12, 2: 0 }

    const wrapper = await mountPage()

    expect(wrapper.get('[data-testid="attempt-1-dropped"]').text()).toBe('12 lines dropped')
    expect(wrapper.find('[data-testid="attempt-2-dropped"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="all-logs-dropped"]').text()).toBe('12 lines dropped')
  })

  it('shows nothing when no lines were dropped', async () => {
    mocks.droppedLines = { 1: 0, 2: 0 }

    const wrapper = await mountPage()

    expect(wrapper.text()).not.toContain('dropped')
  })

  it('downloads the log of an attempt', async () => {
    mocks.downloadJobLogs.mockResolvedValue(new Blob(['x'], { type: 'text/plain' }))
    URL.createObjectURL = vi.fn(() => 'blob:x')
    URL.revokeObjectURL = vi.fn()

    const wrapper = await mountPage()
    await wrapper.get('[data-testid="attempt-2-download"]').trigger('click')
    await flushPromises()

    expect(mocks.downloadJobLogs).toHaveBeenCalledWith(JOB_ID, 2)
    expect(URL.createObjectURL).toHaveBeenCalled()
  })

  it('keeps the object URL alive until the browser has started the download', async () => {
    vi.useFakeTimers()
    try {
      mocks.downloadJobLogs.mockResolvedValue(new Blob(['x'], { type: 'text/plain' }))
      URL.createObjectURL = vi.fn(() => 'blob:x')
      URL.revokeObjectURL = vi.fn()

      const wrapper = await mountPage()
      await wrapper.get('[data-testid="attempt-2-download"]').trigger('click')
      await flushPromises()

      expect(URL.createObjectURL).toHaveBeenCalled()
      expect(URL.revokeObjectURL).not.toHaveBeenCalled()

      vi.runAllTimers()
      expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:x')
    }
    finally {
      vi.useRealTimers()
    }
  })
})
