import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import StepStreamConfig from '../StepStreamConfig.vue'

vi.mock('@entities/connector-task', () => ({
  useConnectorTaskResult: () => ({ data: ref(null), error: ref(null), timedOut: ref(false) }),
}))
vi.mock('@entities/source', () => ({
  discoverSourceSchema: vi.fn(),
  getSourceCatalog: vi.fn(),
}))

const userFields = ['id', 'name', 'email', 'title', 'age', 'created_at', 'updated_at']

const catalog = {
  streams: [{
    stream: {
      name: 'users',
      json_schema: { type: 'object', properties: Object.fromEntries(userFields.map(f => [f, { type: 'string' }])) },
      supported_sync_modes: ['full_refresh', 'incremental'],
    },
  }],
}

describe('wizard streams step', () => {
  it('counts every field of an enabled stream without an explicit field selection', async () => {
    const wrapper = mount(StepStreamConfig, {
      props: { sourceId: 'src-1', streams: [], discoveredCatalog: catalog },
      global: { stubs: { StreamExpandedRow: true } },
    })
    await flushPromises()

    await wrapper.find('input[aria-label="Enable stream users"]').setValue(true)

    const row = wrapper.find('tbody tr')
    expect(row.text()).toContain('7/7 fields')
  })
})
