import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent, ref } from 'vue'
import STextarea from '../STextarea.vue'

describe('sTextarea', () => {
  it('reflects and updates v-model', async () => {
    const Host = defineComponent({
      components: { STextarea },
      setup: () => ({ text: ref('{}') }),
      template: '<STextarea v-model="text" label="Config" />',
    })
    const w = mount(Host)
    const textarea = w.find<HTMLTextAreaElement>('textarea')
    expect(textarea.element.value).toBe('{}')

    await textarea.setValue('{"a":1}')
    expect((w.vm as unknown as { text: string }).text).toBe('{"a":1}')
  })

  it('associates the label with the textarea', () => {
    const w = mount(STextarea, { props: { label: 'Tolerations' } })
    const id = w.find('textarea').attributes('id')
    expect(id).toBeTruthy()
    expect(w.find('label').attributes('for')).toBe(id)
    expect(w.find('label').text()).toBe('Tolerations')
  })

  it('uses a caller-provided id', () => {
    const w = mount(STextarea, { props: { label: 'State' }, attrs: { id: 'state-json' } })
    expect(w.find('textarea').attributes('id')).toBe('state-json')
    expect(w.find('label').attributes('for')).toBe('state-json')
  })

  it('shows the error text and links it to the textarea', () => {
    const w = mount(STextarea, { props: { label: 'Affinity', error: 'Invalid JSON', hint: 'Optional' } })
    const textarea = w.find('textarea')
    const message = w.find('p')
    expect(message.text()).toBe('Invalid JSON')
    expect(message.classes()).toContain('text-danger')
    expect(textarea.attributes('aria-invalid')).toBe('true')
    expect(textarea.attributes('aria-describedby')).toBe(message.attributes('id'))
    expect(w.text()).not.toContain('Optional')
  })

  it('shows the hint without marking the field invalid', () => {
    const w = mount(STextarea, { props: { hint: 'JSON object' } })
    expect(w.find('p').text()).toBe('JSON object')
    expect(w.find('textarea').attributes('aria-invalid')).toBeUndefined()
  })
})
