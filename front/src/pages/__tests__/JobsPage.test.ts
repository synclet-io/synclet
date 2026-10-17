import type { Job } from '@entities/job'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import JobsPage from '../JobsPage.vue'

const JOB_ID = '0f9e8d7c-6b5a-4f3e-9d2c-1b0a9f8e7d6c'

const mocks = vi.hoisted(() => ({
  jobs: [] as Array<Record<string, unknown>>,
  cancelJob: vi.fn(),
}))

vi.mock('@entities/job', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@entities/job')>()
  return {
    ...actual,
    useCancelJob: () => ({ mutateAsync: mocks.cancelJob, isPending: ref(false) }),
  }
})
vi.mock('@entities/job/api', () => ({
  listJobs: vi.fn(),
}))
vi.mock('@entities/connection', () => ({
  useConnections: () => ({ data: ref({ items: [{ id: 'conn-1', name: 'PG to BQ' }] }) }),
}))
vi.mock('@entities/auth', () => ({
  useAuth: () => ({ currentWorkspaceId: ref('ws1') }),
}))
vi.mock('@tanstack/vue-query', () => ({
  useQuery: () => ({
    data: ref({ items: mocks.jobs, total: mocks.jobs.length }),
    isLoading: ref(false),
    error: ref(null),
  }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}))

function job(status: Job['status']): Record<string, unknown> {
  return {
    id: JOB_ID,
    connectionId: 'conn-1',
    status,
    jobType: 'sync',
    attempt: 1,
    maxAttempts: 3,
    attempts: [],
    startedAt: new Date('2026-10-17T10:00:00Z'),
  }
}

const Empty = defineComponent({ template: '<div />' })

async function mountPage() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/jobs', name: 'jobs', component: JobsPage },
      { path: '/jobs/:id', name: 'job-detail', component: Empty },
      { path: '/connections/:id', name: 'connection-detail', component: Empty },
      { path: '/:rest(.*)*', component: Empty },
    ],
  })
  await router.push('/jobs')
  const wrapper = mount(JobsPage, { global: { plugins: [router] }, attachTo: document.body })
  await flushPromises()
  return wrapper
}

function buttonByText(text: string): HTMLButtonElement | undefined {
  return Array.from(document.querySelectorAll('button')).find(b => b.textContent?.trim() === text)
}

describe('jobsPage cancel', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    mocks.cancelJob.mockReset()
    mocks.cancelJob.mockResolvedValue(undefined)
  })

  it('cancels a job that is still starting', async () => {
    mocks.jobs = [job('starting')]
    const wrapper = await mountPage()

    const cancel = wrapper.find('button[aria-label="Cancel job"]')
    expect(cancel.exists()).toBe(true)

    await cancel.trigger('click')
    await flushPromises()

    const confirm = buttonByText('Cancel job')
    expect(confirm).toBeDefined()
    confirm!.click()
    await flushPromises()

    expect(mocks.cancelJob).toHaveBeenCalledExactlyOnceWith(JOB_ID)
  })

  it('offers no cancel for a finished job', async () => {
    mocks.jobs = [job('completed')]
    const wrapper = await mountPage()

    expect(wrapper.find('button[aria-label="Cancel job"]').exists()).toBe(false)
  })
})
