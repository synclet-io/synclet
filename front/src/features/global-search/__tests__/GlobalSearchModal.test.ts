import type { SearchGroup } from '../useGlobalSearch'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, nextTick, ref } from 'vue'
import GlobalSearchModal from '../GlobalSearchModal.vue'

const push = vi.fn()
vi.mock('vue-router', () => ({ useRouter: () => ({ push }) }))
vi.mock('@entities/auth', () => ({
  useAuth: () => ({ user: ref({ id: 'u1' }), currentWorkspaceId: ref('w1') }),
}))

const groups = ref<SearchGroup[]>([])
const isLoading = ref(false)
vi.mock('../useGlobalSearch', () => ({
  useGlobalSearch: () => {
    const flatResults = computed(() => groups.value.flatMap(g => g.results))
    return {
      groups,
      flatResults,
      hasResults: computed(() => flatResults.value.length > 0),
      isLoading,
      refreshAll: vi.fn(),
    }
  },
}))

const searches = ref<string[]>([])
vi.mock('../useRecentHistory', () => ({
  useRecentHistory: () => ({
    searches,
    visits: ref([]),
    rememberSearch: vi.fn(),
    rememberVisit: vi.fn(),
  }),
}))

async function openModal() {
  const wrapper = mount(GlobalSearchModal, { props: { open: false }, attachTo: document.body })
  await wrapper.setProps({ open: true })
  await nextTick()
  return wrapper
}

function press(key: string, target: EventTarget = document.body) {
  target.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
}

describe('globalSearchModal', () => {
  beforeEach(() => {
    push.mockClear()
    groups.value = []
    isLoading.value = false
    searches.value = []
    document.body.innerHTML = ''
  })

  it('closes on Escape', async () => {
    const wrapper = await openModal()
    press('Escape')
    expect(wrapper.emitted('close')).toHaveLength(1)
    wrapper.unmount()
  })

  it('shows a loading row instead of "No results" while lists load', async () => {
    isLoading.value = true
    const wrapper = await openModal()
    const input = document.querySelector('input')!
    input.value = 'abc'
    input.dispatchEvent(new Event('input'))
    await nextTick()
    expect(document.body.textContent).toContain('Loading…')
    expect(document.body.textContent).not.toContain('No results')

    isLoading.value = false
    await nextTick()
    expect(document.body.textContent).toContain('No results')
    wrapper.unmount()
  })

  it('opens the active result on Enter in the input', async () => {
    groups.value = [{ kind: 'source', title: 'Sources', results: [{ id: 's1', kind: 'source', label: 'PG', to: '/sources/s1' }] }]
    const wrapper = await openModal()
    press('Enter', document.querySelector('input')!)
    expect(push).toHaveBeenCalledWith('/sources/s1')
    wrapper.unmount()
  })

  it('lets Enter on a focused recent search act on that button', async () => {
    searches.value = ['orders']
    groups.value = [{ kind: 'source', title: 'Sources', results: [{ id: 's1', kind: 'source', label: 'PG', to: '/sources/s1' }] }]
    const wrapper = await openModal()
    const recent = Array.from(document.querySelectorAll('button')).find(b => b.textContent?.includes('orders'))!
    press('Enter', recent)
    expect(push).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('exposes a labelled modal dialog', async () => {
    const wrapper = await openModal()
    const dialog = document.querySelector('[role="dialog"]')!
    expect(dialog.getAttribute('aria-modal')).toBe('true')
    expect(dialog.getAttribute('aria-label')).toBe('Search')
    wrapper.unmount()
  })
})
