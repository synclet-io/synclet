import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent, ref } from 'vue'
import SCheckbox from '../SCheckbox.vue'

describe('sCheckbox', () => {
  it('reflects and updates v-model', async () => {
    const Host = defineComponent({
      components: { SCheckbox },
      setup: () => ({ checked: ref(false) }),
      template: '<SCheckbox v-model="checked" label="Enabled" />',
    })
    const w = mount(Host)
    const input = w.find<HTMLInputElement>('input[type="checkbox"]')
    expect(input.element.checked).toBe(false)

    await input.setValue(true)
    expect((w.vm as unknown as { checked: boolean }).checked).toBe(true)

    await input.setValue(false)
    expect((w.vm as unknown as { checked: boolean }).checked).toBe(false)
  })

  it('sets the indeterminate DOM state from the prop', async () => {
    const w = mount(SCheckbox, { props: { modelValue: false, indeterminate: true } })
    const input = w.find<HTMLInputElement>('input').element
    expect(input.indeterminate).toBe(true)

    await w.setProps({ indeterminate: false })
    expect(input.indeterminate).toBe(false)
  })

  it('keeps indeterminate after a click until the prop changes', async () => {
    const w = mount(SCheckbox, { props: { modelValue: false, indeterminate: true }, attachTo: document.body })
    const input = w.find<HTMLInputElement>('input')
    await input.trigger('click')
    expect(input.element.indeterminate).toBe(true)
    expect(w.emitted('update:modelValue')?.[0]).toEqual([true])
    w.unmount()
  })

  it('toggles when the label text is clicked', async () => {
    const w = mount(SCheckbox, { props: { modelValue: false, label: 'Enabled' }, attachTo: document.body })
    expect(w.element.tagName).toBe('LABEL')
    await w.find('span.text-sm').trigger('click')
    expect(w.emitted('update:modelValue')?.[0]).toEqual([true])
    w.unmount()
  })

  it('forwards attributes to the input and class to the root', () => {
    const w = mount(SCheckbox, { attrs: { 'aria-label': 'Select row', 'class': 'mt-2' } })
    expect(w.find('input').attributes('aria-label')).toBe('Select row')
    expect(w.classes()).toContain('mt-2')
    expect(w.find('input').classes()).not.toContain('mt-2')
  })

  it('does not emit when disabled', async () => {
    const w = mount(SCheckbox, { props: { modelValue: false, disabled: true, label: 'Enabled' }, attachTo: document.body })
    expect(w.find('input').attributes('disabled')).toBeDefined()
    await w.find('span.text-sm').trigger('click')
    expect(w.emitted('update:modelValue')).toBeUndefined()
    w.unmount()
  })
})
