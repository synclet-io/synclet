import type { OnboardingState } from '../useOnboardingState'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { useOnboardingState } from '../useOnboardingState'

const listJobs = vi.hoisted(() => vi.fn())

vi.mock('@entities/auth', () => ({
  useAuth: () => ({ currentWorkspaceId: ref('ws-1'), user: ref({ id: 'user-1' }) }),
}))
vi.mock('@entities/source/api', () => ({ listSources: async () => ({ items: [], total: 1 }) }))
vi.mock('@entities/destination/api', () => ({ listDestinations: async () => ({ items: [], total: 1 }) }))
vi.mock('@entities/connection/api', () => ({ listConnections: async () => ({ items: [], total: 1 }) }))
vi.mock('@entities/stats/api', () => ({ getWorkspaceStats: async () => ({ totalSyncs: 0, failedSyncs: 0 }) }))
vi.mock('@entities/job/api', () => ({ listJobs }))

async function mountState(): Promise<OnboardingState> {
  let state: OnboardingState | undefined
  mount(defineComponent({
    setup() {
      state = useOnboardingState()
      return () => null
    },
  }), {
    global: { plugins: [[VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]] },
  })
  await flushPromises()
  return state!
}

function firstSyncDone(state: OnboardingState): boolean | undefined {
  return state.steps.value.find(s => s.id === 'first-sync')?.done
}

describe('useOnboardingState first sync', () => {
  it('is done once a sync has completed, before the stats rollup counts it', async () => {
    listJobs.mockResolvedValue({ items: [{ id: 'job-1', status: 'completed' }], total: 1 })

    const state = await mountState()

    expect(firstSyncDone(state)).toBe(true)
    expect(listJobs).toHaveBeenCalledWith('', expect.objectContaining({ statuses: ['completed'] }))
  })

  it('is not done while no sync has completed', async () => {
    listJobs.mockResolvedValue({ items: [], total: 0 })

    const state = await mountState()

    expect(firstSyncDone(state)).toBe(false)
  })
})
