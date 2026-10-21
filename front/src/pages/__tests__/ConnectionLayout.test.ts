import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import ConnectionLayout from '../ConnectionLayout.vue'

const mocks = vi.hoisted(() => ({
  connection: null as Record<string, unknown> | null,
  triggerSync: vi.fn(),
}))

vi.mock('@entities/connection', () => ({
  useConnection: () => ({ data: ref(mocks.connection), isLoading: ref(false), error: ref(null) }),
  useEnableConnection: () => ({ mutateAsync: vi.fn(), isPending: ref(false) }),
  useDisableConnection: () => ({ mutateAsync: vi.fn(), isPending: ref(false) }),
}))
vi.mock('@entities/job', () => ({
  useActiveJob: () => ({ data: ref(null) }),
  useCancelJob: () => ({ mutateAsync: vi.fn(), isPending: ref(false) }),
  useTriggerSync: () => ({ mutateAsync: mocks.triggerSync, isPending: ref(false) }),
}))
vi.mock('@entities/stats', () => ({
  useConnectionStats: () => ({ data: ref(null) }),
}))

const Empty = defineComponent({ template: '<div />' })

async function mountLayout(streamsConfigured: boolean) {
  mocks.connection = { id: 'c1', name: 'PG to BQ', status: 'active', streamsConfigured }
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/connections', name: 'connections', component: Empty },
      { path: '/connections/:id', name: 'connection-detail', component: ConnectionLayout },
      { path: '/connections/:id/schema-changes', name: 'connection-schema-changes', component: Empty },
      { path: '/connections/:id/settings', name: 'connection-settings', component: Empty },
      { path: '/connections/:id/notifications', name: 'connection-notifications', component: Empty },
      { path: '/:rest(.*)*', component: Empty },
    ],
  })
  await router.push('/connections/c1')
  const wrapper = mount(ConnectionLayout, { global: { plugins: [router] } })
  await flushPromises()
  return wrapper
}

function syncNow(wrapper: Awaited<ReturnType<typeof mountLayout>>) {
  return wrapper.findAll('button').find(b => b.text().includes('Sync Now'))!
}

describe('connectionLayout streams', () => {
  beforeEach(() => {
    mocks.triggerSync.mockReset()
  })

  it('shows "Streams not configured" with a link and disables Sync Now', async () => {
    const wrapper = await mountLayout(false)

    const badge = wrapper.find('a[href="/connections/c1/streams"][title]')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toBe('Streams not configured')
    expect(syncNow(wrapper).attributes('disabled')).toBeDefined()
    const tooltip = wrapper.get('[data-testid="sync-now"]')
    expect(tooltip.attributes('title')).toBe('Configure streams first')
    expect(tooltip.element.tagName).not.toBe('BUTTON')
    expect(tooltip.attributes('disabled')).toBeUndefined()

    await syncNow(wrapper).trigger('click')
    expect(mocks.triggerSync).not.toHaveBeenCalled()
  })

  it('enables Sync Now once streams are configured', async () => {
    const wrapper = await mountLayout(true)

    expect(wrapper.text()).not.toContain('Streams not configured')
    expect(syncNow(wrapper).attributes('disabled')).toBeUndefined()
    expect(wrapper.get('[data-testid="sync-now"]').attributes('title')).toBeUndefined()
  })
})
