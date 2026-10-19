import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import ConnectorStarToggle from '../ConnectorStarToggle.vue'

const setStar = vi.hoisted(() => vi.fn())
const role = vi.hoisted(() => ({ value: 'editor' }))

vi.mock('@entities/auth', () => ({
  useAuth: () => ({ currentWorkspaceId: ref('ws-1'), workspaces: ref([{ workspaceId: 'ws-1', role: role.value }]) }),
}))
vi.mock('@entities/connector', () => ({
  useSetConnectorStar: () => ({ mutate: setStar, isPending: ref(false) }),
}))

beforeEach(() => {
  setStar.mockClear()
  role.value = 'editor'
})

describe('connectorStarToggle', () => {
  it('stars an unstarred connector', async () => {
    const wrapper = mount(ConnectorStarToggle, { props: { connector: { id: 'c1', name: 'Faker', starred: false } } })
    const button = wrapper.get('button')
    expect(button.attributes('aria-label')).toBe('Star Faker')
    expect(button.attributes('aria-pressed')).toBe('false')
    await button.trigger('click')
    expect(setStar.mock.calls[0]![0]).toEqual({ id: 'c1', starred: true })
  })

  it('unstars a starred connector', async () => {
    const wrapper = mount(ConnectorStarToggle, { props: { connector: { id: 'c1', name: 'Faker', starred: true } } })
    const button = wrapper.get('button')
    expect(button.attributes('aria-label')).toBe('Unstar Faker')
    expect(button.attributes('aria-pressed')).toBe('true')
    await button.trigger('click')
    expect(setStar.mock.calls[0]![0]).toEqual({ id: 'c1', starred: false })
  })

  it('is disabled for a viewer', async () => {
    role.value = 'viewer'
    const wrapper = mount(ConnectorStarToggle, { props: { connector: { id: 'c1', name: 'Faker', starred: true } } })
    const button = wrapper.get('button')
    expect(button.attributes('disabled')).toBeDefined()
    await button.trigger('click')
    expect(setStar).not.toHaveBeenCalled()
  })
})
