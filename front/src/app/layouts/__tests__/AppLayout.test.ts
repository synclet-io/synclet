import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import AppLayout from '../AppLayout.vue'

vi.mock('@vueuse/core', async importOriginal => ({
  ...await importOriginal<typeof import('@vueuse/core')>(),
  useMediaQuery: () => ref(false),
}))
vi.mock('@entities/auth', () => ({
  useAuth: () => ({
    user: ref({ id: 'u1', name: 'Ada', email: 'ada@example.com' }),
    workspaces: ref([]),
    currentWorkspaceId: ref('w1'),
    logout: vi.fn(),
    switchWorkspace: vi.fn(),
  }),
}))
vi.mock('@entities/system', () => ({ useSystemInfo: () => ({ data: ref(undefined) }) }))
vi.mock('@features/global-search', () => ({ GlobalSearchModal: defineComponent({ render: () => null }) }))

const Empty = defineComponent({ render: () => null })

async function mountLayout(path = '/') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: Empty },
      { path: '/sources/:id?', component: Empty },
      { path: '/settings/general', component: Empty },
      { path: '/:rest(.*)*', component: Empty },
    ],
  })
  await router.push(path)
  const wrapper = mount(AppLayout, { global: { plugins: [router] }, attachTo: document.body })
  await flushPromises()
  return wrapper
}

describe('appLayout', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('keeps the closed drawer out of the tab order', async () => {
    const wrapper = await mountLayout()
    expect((wrapper.get('#app-sidebar').element as HTMLElement).inert).toBe(true)
    expect(wrapper.get('[aria-label="Open menu"]').attributes('aria-expanded')).toBe('false')
    wrapper.unmount()
  })

  it('opens the drawer with focus inside and closes it on Escape', async () => {
    const wrapper = await mountLayout()
    const menu = wrapper.get('[aria-label="Open menu"]')
    await menu.trigger('click')
    await flushPromises()

    const sidebar = wrapper.get('#app-sidebar').element as HTMLElement
    expect(menu.attributes('aria-expanded')).toBe('true')
    expect(sidebar.inert).toBe(false)
    expect(sidebar.contains(document.activeElement)).toBe(true)
    expect((wrapper.get('main').element as HTMLElement).inert).toBe(true)

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }))
    await flushPromises()
    expect(menu.attributes('aria-expanded')).toBe('false')
    expect(document.activeElement).toBe(menu.element)
    wrapper.unmount()
  })

  it('highlights the section on detail pages and settings sub-pages', async () => {
    const sources = await mountLayout('/sources/123')
    expect(sources.get('a[href="/sources"]').attributes('aria-current')).toBe('page')
    sources.unmount()

    const settings = await mountLayout('/settings/general')
    expect(settings.get('a[href="/settings"]').attributes('aria-current')).toBe('page')
    settings.unmount()
  })
})
