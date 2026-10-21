import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import WorkflowsPage from '../WorkflowsPage.vue'

const mocks = vi.hoisted(() => ({ role: 'editor' }))

vi.mock('@entities/auth', () => ({
  useAuth: () => ({
    currentWorkspaceId: ref('ws1'),
    workspaces: ref([{ workspaceId: 'ws1', role: mocks.role }]),
  }),
}))
vi.mock('@entities/workflow', () => ({
  useWorkflows: () => ({
    data: ref([
      { id: 'w1', name: 'Nightly', schedule: '', nextScheduledAt: undefined, steps: [{}, {}] },
    ]),
    isLoading: ref(false),
    error: ref(null),
  }),
}))

function mountPage() {
  return mount(WorkflowsPage, {
    global: { stubs: { RouterLink: { props: ['to'], template: '<a :data-to="JSON.stringify(to)"><slot /></a>' } } },
  })
}

describe('workflows page', () => {
  it('lists the workflows with a link to each', async () => {
    const w = mountPage()
    await flushPromises()

    const row = w.get('tbody tr')
    expect(row.text()).toContain('Nightly')
    expect(row.text()).toContain('2')
    expect(row.text()).toContain('Manual')
    expect(JSON.parse(row.get('a').attributes('data-to')!)).toEqual({ name: 'workflow-detail', params: { id: 'w1' } })
    expect(w.text()).toContain('New workflow')
  })

  it('offers no new workflow to a viewer', async () => {
    mocks.role = 'viewer'
    const w = mountPage()
    await flushPromises()

    expect(w.text()).not.toContain('New workflow')
  })
})
