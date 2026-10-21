import type { Workflow, WorkflowRun } from '@entities/workflow'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import WorkflowPage from '../WorkflowPage.vue'

const mocks = vi.hoisted(() => ({
  role: 'editor',
  params: { id: 'w1' } as Record<string, string>,
  workflow: undefined as unknown,
  runs: [] as unknown[],
  update: vi.fn(),
  create: vi.fn(),
  run: vi.fn(),
  rerun: vi.fn(),
  push: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: mocks.params }),
  useRouter: () => ({ push: mocks.push }),
  RouterLink: { props: ['to'], template: '<a :data-to="JSON.stringify(to)"><slot /></a>' },
}))
vi.mock('@entities/auth', () => ({
  useAuth: () => ({
    currentWorkspaceId: ref('ws1'),
    workspaces: ref([{ workspaceId: 'ws1', role: mocks.role }]),
  }),
}))
vi.mock('@entities/connection', () => ({
  useConnections: () => ({
    data: ref({ items: [{ id: 'a', name: 'Orders' }, { id: 'b', name: 'Customers' }, { id: 'c', name: 'Revenue' }], total: 3 }),
  }),
}))
vi.mock('@entities/workflow', () => ({
  useWorkflow: () => ({ data: ref(mocks.workflow), error: ref(null) }),
  useWorkflowRuns: () => ({ data: ref(mocks.runs) }),
  useCreateWorkflow: () => ({ mutateAsync: mocks.create, isPending: ref(false) }),
  useUpdateWorkflow: () => ({ mutateAsync: mocks.update, isPending: ref(false) }),
  useDeleteWorkflow: () => ({ mutateAsync: vi.fn(), isPending: ref(false) }),
  useRunWorkflow: () => ({ mutateAsync: mocks.run, isPending: ref(false) }),
  useRerunWorkflowRun: () => ({ mutateAsync: mocks.rerun, isPending: ref(false) }),
}))
vi.mock('@shared/ui', async importOriginal => ({
  ...(await importOriginal<typeof import('@shared/ui')>()),
  useToast: () => ({ success: vi.fn(), error: vi.fn() }),
}))

const workflow: Workflow = {
  id: 'w1',
  workspaceId: 'ws1',
  name: 'Nightly',
  schedule: '0 2 * * *',
  nextScheduledAt: undefined,
  steps: [
    { id: 's1', connectionId: 'a', afterConnectionIds: [] },
    { id: 's2', connectionId: 'b', afterConnectionIds: ['a'] },
  ],
  createdAt: undefined,
  updatedAt: undefined,
}

type StepRun = WorkflowRun['stepRuns'][number]

function run(id: string, status: WorkflowRun['status'], a: string, b: string, extra: Record<string, Partial<StepRun>> = {}): WorkflowRun {
  const stepRun = (connectionId: string, s: string): StepRun => ({
    id: `${id}-${connectionId}`,
    stepId: `s-${connectionId}`,
    connectionId,
    status: s as StepRun['status'],
    jobId: '',
    error: '',
    startedAt: undefined,
    completedAt: undefined,
    ...extra[connectionId],
  })
  return { id, workflowId: 'w1', status, trigger: 'manual', startedAt: new Date('2026-10-21T10:00:00Z'), completedAt: undefined, stepRuns: [stepRun('a', a), stepRun('b', b)] }
}

const failedRun = run('r2', 'failed', 'failed', 'skipped', {
  a: { jobId: 'j-a', error: 'source check failed: bad password', startedAt: new Date('2026-10-21T10:00:00Z'), completedAt: new Date('2026-10-21T10:01:30Z') },
})
const succeededRun = run('r1', 'succeeded', 'succeeded', 'succeeded', {
  a: { jobId: 'j-a1', startedAt: new Date('2026-10-20T10:00:00Z'), completedAt: new Date('2026-10-20T10:00:05Z') },
  b: { jobId: 'j-b1', startedAt: new Date('2026-10-20T10:00:05Z'), completedAt: new Date('2026-10-20T10:00:07Z') },
})

function stepRows(w: ReturnType<typeof mountPage>) {
  return w.findAll('[data-testid="step-runs"] tbody tr')
}

function mountPage() {
  return mount(WorkflowPage, {
    global: {
      stubs: { SCronEditor: true },
    },
    attachTo: document.body,
  })
}

function graphStatus(w: ReturnType<typeof mountPage>, connectionId: string) {
  return w.get(`svg [data-step="${connectionId}"]`).attributes('data-status')
}

function button(w: ReturnType<typeof mountPage>, text: string) {
  return w.findAll('button').find(b => b.text() === text)
}

describe('workflow page', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    mocks.role = 'editor'
    mocks.params = { id: 'w1' }
    mocks.workflow = workflow
    mocks.runs = []
    for (const fn of [mocks.update, mocks.create, mocks.run, mocks.rerun, mocks.push])
      fn.mockReset()
  })

  it('saves the edited steps as a list with their after connections', async () => {
    mocks.update.mockResolvedValue(workflow)
    const w = mountPage()
    await flushPromises()

    await w.get('select[aria-label="Connection to add"]').setValue('c')
    await button(w, 'Add step')!.trigger('click')
    await w.get('[data-step="c"] input[data-after="b"]').setValue(true)
    await button(w, 'Save workflow')!.trigger('click')
    await flushPromises()

    expect(mocks.update).toHaveBeenCalledWith({
      id: 'w1',
      name: 'Nightly',
      schedule: '0 2 * * *',
      steps: [
        { connectionId: 'a', afterConnectionIds: [] },
        { connectionId: 'b', afterConnectionIds: ['a'] },
        { connectionId: 'c', afterConnectionIds: ['b'] },
      ],
    })
  })

  it('creates a new workflow and opens it', async () => {
    mocks.params = {}
    mocks.workflow = undefined
    mocks.create.mockResolvedValue({ ...workflow, id: 'w2' })
    const w = mountPage()
    await flushPromises()

    expect(w.text()).toContain('New workflow')
    expect(button(w, 'Run now')).toBeUndefined()
    await w.get('input').setValue('Hourly')
    await button(w, 'Create workflow')!.trigger('click')
    await flushPromises()

    expect(mocks.create).toHaveBeenCalledWith({ name: 'Hourly', schedule: '', steps: [] })
    expect(mocks.push).toHaveBeenCalledWith({ name: 'workflow-detail', params: { id: 'w2' } })
  })

  it('shows the graph with the latest run and a chosen run', async () => {
    mocks.runs = [run('r2', 'failed', 'failed', 'skipped'), run('r1', 'succeeded', 'succeeded', 'succeeded')]
    const w = mountPage()
    await flushPromises()

    expect(graphStatus(w, 'a')).toBe('failed')
    expect(graphStatus(w, 'b')).toBe('skipped')
    expect(w.findAll('svg path[marker-end]')).toHaveLength(1)

    await w.get('select[aria-label="Run shown in the graph"]').setValue('r1')
    expect(graphStatus(w, 'a')).toBe('succeeded')
  })

  it('shows the run history and reruns a failed run', async () => {
    mocks.runs = [run('r2', 'failed', 'failed', 'skipped'), run('r1', 'succeeded', 'succeeded', 'succeeded')]
    mocks.rerun.mockResolvedValue(mocks.runs[0])
    const w = mountPage()
    await flushPromises()

    const rows = w.findAll('[data-testid="run-history"] tbody tr')
    expect(rows).toHaveLength(2)
    expect(rows[0]!.text()).toContain('failed')
    expect(rows[0]!.text()).toContain('0/2 succeeded')
    expect(rows[1]!.text()).not.toContain('Rerun failed steps')

    await rows[0]!.findAll('button').find(b => b.text() === 'Rerun failed steps')!.trigger('click')
    await flushPromises()
    expect(mocks.rerun).toHaveBeenCalledWith('r2')
  })

  it('shows the first steps of a just started run as starting and blocks changes while it runs', async () => {
    mocks.runs = [run('r3', 'running', 'pending', 'pending')]
    const w = mountPage()
    await flushPromises()

    expect(graphStatus(w, 'a')).toBe('starting')
    expect(graphStatus(w, 'b')).toBe('waiting')
    expect(button(w, 'Run now')!.attributes('disabled')).toBeDefined()
    expect(button(w, 'Save workflow')!.attributes('disabled')).toBeDefined()
  })

  it('starts a run', async () => {
    mocks.run.mockResolvedValue(run('r9', 'running', 'pending', 'pending'))
    const w = mountPage()
    await flushPromises()

    await button(w, 'Run now')!.trigger('click')
    await flushPromises()
    expect(mocks.run).toHaveBeenCalledWith('w1')
  })

  it('shows a viewer the workflow and its runs without edit controls', async () => {
    mocks.role = 'viewer'
    mocks.runs = [run('r2', 'failed', 'failed', 'skipped')]
    const w = mountPage()
    await flushPromises()

    expect(button(w, 'Run now')).toBeUndefined()
    expect(button(w, 'Delete')).toBeUndefined()
    expect(button(w, 'Save workflow')).toBeUndefined()
    expect(button(w, 'Rerun failed steps')).toBeUndefined()
    expect(w.get('[data-testid="schedule-readonly"]').text()).toContain('Schedule')
    expect(w.get('input').attributes('disabled')).toBeDefined()
    expect(graphStatus(w, 'a')).toBe('failed')
  })

  it('lists the steps of the latest run with their status, timing, error and job link', async () => {
    mocks.runs = [failedRun, succeededRun]
    const w = mountPage()
    await flushPromises()

    const rows = stepRows(w)
    expect(rows).toHaveLength(2)
    expect(rows[0]!.text()).toContain('Orders')
    expect(rows[0]!.text()).toContain('failed')
    expect(rows[0]!.text()).toContain(new Date('2026-10-21T10:00:00Z').toLocaleString())
    expect(rows[0]!.text()).toContain('1m 30s')
    expect(rows[0]!.text()).toContain('source check failed: bad password')
    expect(JSON.parse(rows[0]!.get('a[data-to]').attributes('data-to')!)).toEqual({ name: 'job-detail', params: { id: 'j-a' } })
    expect(rows[1]!.text()).toContain('Customers')
    expect(rows[1]!.text()).toContain('skipped')
    expect(rows[1]!.find('a[data-to]').exists()).toBe(false)
  })

  it('opens the job of a step clicked in the graph and shows a failed step\'s error on hover', async () => {
    mocks.runs = [failedRun]
    const w = mountPage()
    await flushPromises()

    expect(w.get('svg [data-step="a"] title').text()).toContain('source check failed: bad password')
    await w.get('svg [data-step="a"]').trigger('click')
    expect(mocks.push).toHaveBeenCalledWith({ name: 'job-detail', params: { id: 'j-a' } })

    mocks.push.mockReset()
    await w.get('svg [data-step="b"]').trigger('click')
    expect(mocks.push).not.toHaveBeenCalled()
  })

  it('lists the step runs in graph order with steps no longer in the workflow last', async () => {
    mocks.workflow = {
      ...workflow,
      steps: [
        { id: 's3', connectionId: 'c', afterConnectionIds: ['a'] },
        { id: 's1', connectionId: 'a', afterConnectionIds: [] },
        { id: 's2', connectionId: 'b', afterConnectionIds: ['a'] },
      ],
    }
    const base = run('r1', 'succeeded', 'succeeded', 'succeeded')
    const stepRun = (connectionId: string): StepRun => ({ ...base.stepRuns[0]!, id: `r1-${connectionId}`, connectionId })
    mocks.runs = [{ ...base, stepRuns: [stepRun('removed'), stepRun('b'), stepRun('c'), stepRun('a')] }]
    const w = mountPage()
    await flushPromises()

    expect(stepRows(w).map(r => r.find('td').text())).toEqual(['Orders', 'Revenue', 'Customers', 'removed'])
  })

  it('shows the run chosen in the run history in the graph and the step list', async () => {
    mocks.runs = [failedRun, succeededRun]
    const w = mountPage()
    await flushPromises()

    const history = w.findAll('[data-testid="run-history"] tbody tr')
    await history[1]!.findAll('button').find(b => b.text() === 'Show in graph')!.trigger('click')

    expect(graphStatus(w, 'a')).toBe('succeeded')
    expect(graphStatus(w, 'b')).toBe('succeeded')
    const rows = stepRows(w)
    expect(rows[0]!.text()).toContain('succeeded')
    expect(rows[0]!.text()).toContain('5.0s')
    expect(rows[0]!.text()).not.toContain('bad password')
    expect(JSON.parse(rows[1]!.get('a[data-to]').attributes('data-to')!)).toEqual({ name: 'job-detail', params: { id: 'j-b1' } })
  })
})
