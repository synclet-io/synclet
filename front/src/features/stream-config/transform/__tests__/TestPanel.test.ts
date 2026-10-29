import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import TestPanel from '../TestPanel.vue'

describe('testPanel', () => {
  it('generates a sample record from the column types', async () => {
    const wrapper = mount(TestPanel, {
      props: {
        columns: [
          { name: 'created_at', type: 'timestamp' },
          { name: 'local_at', type: 'datetime' },
          { name: 'day', type: 'date' },
        ],
        steps: [],
        samples: [],
        modelValue: '',
        parseError: '',
        loading: false,
        blocked: false,
      },
    })
    await wrapper.get('[data-testid="test-generate"]').trigger('click')

    const [value] = wrapper.emitted('update:modelValue')!.at(-1) as [string]
    expect(JSON.parse(value)).toEqual({
      created_at: '2026-01-01T12:00:00Z',
      local_at: '2026-01-01T12:00:00',
      day: '2026-01-01',
    })
  })
})
