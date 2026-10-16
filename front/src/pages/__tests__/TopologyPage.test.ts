import type { TopologyData } from '@features/topology/types'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import TopologyPage from '../TopologyPage.vue'

const push = vi.fn()
vi.mock('vue-router', async importOriginal => ({
  ...await importOriginal<typeof import('vue-router')>(),
  useRouter: () => ({ push }),
}))

const data = ref<TopologyData>({
  sources: [{ id: 's1', kind: 'source', name: 'Postgres', health: 'healthy', enabled: true }],
  connections: [{ id: 'c1', kind: 'connection', name: 'PG to BQ', health: 'failing', enabled: true }],
  destinations: [{ id: 'd1', kind: 'destination', name: 'BigQuery', health: 'healthy', enabled: true }],
  edges: [],
})
vi.mock('@features/topology/useTopology', () => ({
  useTopology: () => ({ data, isLoading: ref(false) }),
}))

describe('topologyPage', () => {
  it('renders every node as a focusable link with an accessible name', () => {
    const wrapper = mount(TopologyPage, { global: { stubs: { RouterLink: true } } })
    const nodes = wrapper.findAll('g[role="link"]')
    expect(nodes).toHaveLength(3)
    for (const node of nodes)
      expect(node.attributes('tabindex')).toBe('0')
    expect(nodes[1].attributes('aria-label')).toBe('connection PG to BQ, failing')
  })

  it('opens the node on Enter', async () => {
    const wrapper = mount(TopologyPage, { global: { stubs: { RouterLink: true } } })
    const nodes = wrapper.findAll('g[role="link"]')
    await nodes[0].trigger('keydown', { key: 'Enter' })
    expect(push).toHaveBeenCalledWith('/sources/s1')
    await nodes[2].trigger('keydown', { key: 'Enter' })
    expect(push).toHaveBeenCalledWith('/destinations/d1')
  })

  it('shows the tooltip while a node has focus', async () => {
    const wrapper = mount(TopologyPage, { global: { stubs: { RouterLink: true } } })
    const node = wrapper.findAll('g[role="link"]')[1]
    await node.trigger('focus')
    expect(wrapper.text()).toContain('Health:')
    await node.trigger('blur')
    expect(wrapper.text()).not.toContain('Health:')
  })
})
