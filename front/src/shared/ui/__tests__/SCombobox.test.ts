import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import SCombobox from '../SCombobox.vue'
import { expectNoAxeViolations } from './axe'

const options = [
  { label: 'Postgres', value: 'pg' },
  { label: 'MySQL', value: 'my' },
  { label: 'BigQuery', value: 'bq' },
]

function mountCombobox(props: Record<string, unknown> = {}, attrs: Record<string, unknown> = {}) {
  return mount(SCombobox, {
    props: { label: 'Source', options, modelValue: '', placeholder: 'All sources', ...props },
    attrs,
    attachTo: document.body,
  })
}

afterEach(() => {
  document.body.innerHTML = ''
  document.documentElement.classList.remove('dark')
})

describe('sCombobox', () => {
  it('associates the label with a combobox input that controls a listbox', () => {
    const w = mountCombobox()
    const input = w.get('input')
    expect(input.attributes('role')).toBe('combobox')
    expect(w.get('label').attributes('for')).toBe(input.attributes('id'))
    expect(input.attributes('aria-expanded')).toBe('false')
    const listbox = w.get(`#${input.attributes('aria-controls')}`)
    expect(listbox.attributes('role')).toBe('listbox')
    expect(listbox.attributes('aria-labelledby')).toBe(w.get('label').attributes('id'))
  })

  it('uses a caller-supplied id and keeps class on the wrapper', () => {
    const w = mountCombobox({}, { id: 'src', class: 'w-56' })
    expect(w.get('input').attributes('id')).toBe('src')
    expect(w.get('label').attributes('for')).toBe('src')
    expect(w.classes()).toContain('w-56')
    expect(w.get('input').classes()).not.toContain('w-56')
  })

  it('shows the selected label when closed', () => {
    const w = mountCombobox({ modelValue: 'my' })
    expect((w.get('input').element as HTMLInputElement).value).toBe('MySQL')
  })

  it('opens on ArrowDown and moves the active option with the arrows', async () => {
    const w = mountCombobox({ modelValue: 'my' })
    const input = w.get('input')
    await input.trigger('keydown', { key: 'ArrowDown' })
    expect(input.attributes('aria-expanded')).toBe('true')
    const opts = w.findAll('[role="option"]')
    expect(opts).toHaveLength(3)
    expect(opts.map(o => o.attributes('aria-selected'))).toEqual(['false', 'true', 'false'])
    expect(input.attributes('aria-activedescendant')).toBe(opts[1].attributes('id'))

    await input.trigger('keydown', { key: 'ArrowDown' })
    expect(input.attributes('aria-activedescendant')).toBe(opts[2].attributes('id'))
    await input.trigger('keydown', { key: 'ArrowDown' })
    expect(input.attributes('aria-activedescendant')).toBe(opts[0].attributes('id'))
    await input.trigger('keydown', { key: 'ArrowUp' })
    expect(input.attributes('aria-activedescendant')).toBe(opts[2].attributes('id'))
  })

  it('filters by typed text and picks the active option with Enter', async () => {
    const w = mountCombobox()
    const input = w.get('input')
    await input.setValue('big')
    expect(input.attributes('aria-expanded')).toBe('true')
    const opts = w.findAll('[role="option"]')
    expect(opts.map(o => o.text())).toEqual(['BigQuery'])
    expect(input.attributes('aria-activedescendant')).toBe(opts[0].attributes('id'))
    await input.trigger('keydown', { key: 'Enter' })
    expect(w.emitted('update:modelValue')).toEqual([['bq']])
    expect(input.attributes('aria-expanded')).toBe('false')
  })

  it('picks an option on click', async () => {
    const w = mountCombobox()
    await w.get('input').trigger('click')
    await w.findAll('[role="option"]')[2].trigger('click')
    expect(w.emitted('update:modelValue')).toEqual([['bq']])
  })

  it('closes on Escape without changing the value', async () => {
    const w = mountCombobox()
    const input = w.get('input')
    await input.trigger('keydown', { key: 'ArrowDown' })
    await input.trigger('keydown', { key: 'Escape' })
    expect(input.attributes('aria-expanded')).toBe('false')
    expect(input.attributes('aria-activedescendant')).toBeUndefined()
    expect(w.emitted('update:modelValue')).toBeUndefined()
  })

  it('shows the empty text when nothing matches', async () => {
    const w = mountCombobox({ emptyText: 'No sources found' })
    const input = w.get('input')
    await input.setValue('zzz')
    expect(w.findAll('[role="option"]')).toHaveLength(0)
    expect(w.text()).toContain('No sources found')
    expect(input.attributes('aria-activedescendant')).toBeUndefined()
    await input.trigger('keydown', { key: 'Enter' })
    expect(w.emitted('update:modelValue')).toBeUndefined()
  })

  it('clears the value with the clear button', async () => {
    const w = mountCombobox({ modelValue: 'pg' })
    const clear = w.get('button[aria-label="Clear Source"]')
    await clear.trigger('click')
    expect(w.emitted('update:modelValue')).toEqual([['']])
    await w.setProps({ modelValue: '' })
    expect(w.find('button[aria-label="Clear Source"]').exists()).toBe(false)
  })

  it('matches SSelect sizing and chevron inset', () => {
    const md = mountCombobox()
    expect(md.get('input').classes()).toEqual(expect.arrayContaining(['h-10', 'pl-3.5', 'text-sm']))
    expect(md.get('svg.pointer-events-none').classes()).toEqual(expect.arrayContaining(['right-3', 'w-4', 'h-4']))
    const sm = mountCombobox({ size: 'sm' })
    expect(sm.get('input').classes()).toEqual(expect.arrayContaining(['h-7', 'pl-2', 'text-xs']))
    expect(sm.get('svg.pointer-events-none').classes()).toEqual(expect.arrayContaining(['right-2', 'w-3.5', 'h-3.5']))
  })

  for (const theme of ['light', 'dark'] as const) {
    it(`has no axe violations closed, open and empty in the ${theme} theme`, async () => {
      document.documentElement.classList.toggle('dark', theme === 'dark')
      const w = mountCombobox({ modelValue: 'pg', emptyText: 'No sources found' })
      await expectNoAxeViolations()
      const input = w.get('input')
      await input.trigger('keydown', { key: 'ArrowDown' })
      await expectNoAxeViolations()
      await input.setValue('zzz')
      await expectNoAxeViolations()
      w.unmount()
    })
  }
})
