import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import ExecutorGroupField from '../ExecutorGroupField.vue'

const data = vi.hoisted(() => ({
  executors: [] as Array<{ executorGroup: string }>,
  sources: [] as Array<{ executorGroup: string }>,
}))

vi.mock('@entities/system', () => ({
  useExecutors: () => ({ data: ref(data.executors), isLoading: ref(false) }),
}))
vi.mock('@entities/source', () => ({
  useSources: () => ({ data: ref({ items: data.sources }), isLoading: ref(false) }),
}))
vi.mock('@entities/destination', () => ({
  useDestinations: () => ({ data: ref({ items: [] }), isLoading: ref(false) }),
}))

beforeEach(() => {
  data.executors = [{ executorGroup: 'default' }]
  data.sources = []
})

describe('executorGroupField', () => {
  it('renders nothing when default is the only group', () => {
    const wrapper = mount(ExecutorGroupField, { props: { modelValue: 'default' } })
    expect(wrapper.find('select').exists()).toBe(false)
    expect(wrapper.text()).toBe('')
  })

  it('flags a used group without live executors', () => {
    data.sources = [{ executorGroup: 'us' }]
    const wrapper = mount(ExecutorGroupField, { props: { modelValue: 'us' } })
    const options = wrapper.findAll('option').map(o => o.text())
    expect(options).toEqual(['default', 'us (no live executors)'])
    expect(wrapper.text()).toContain('No live executor in group "us": its syncs and checks will not run until an executor of this group connects.')
  })

  it('shows no warning for a group with a live executor', () => {
    data.sources = [{ executorGroup: 'us' }]
    const wrapper = mount(ExecutorGroupField, { props: { modelValue: 'default' } })
    expect(wrapper.text()).not.toContain('No live executor')
  })

  it('leaves out the excluded group', () => {
    data.executors = [{ executorGroup: 'default' }, { executorGroup: 'eu' }]
    const wrapper = mount(ExecutorGroupField, { props: { modelValue: 'eu', exclude: 'default' } })
    expect(wrapper.findAll('option').map(o => o.text())).toEqual(['eu'])
  })

  it('emits the selected group', async () => {
    data.executors = [{ executorGroup: 'default' }, { executorGroup: 'eu' }]
    const wrapper = mount(ExecutorGroupField, { props: { modelValue: 'default' } })
    await wrapper.find('select').setValue('eu')
    expect(wrapper.emitted('update:modelValue')).toEqual([['eu']])
  })
})
