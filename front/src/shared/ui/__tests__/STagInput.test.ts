import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import STagInput from '../STagInput.vue'
import { expectNoAxeViolations } from './axe'

function mountTags(props: Record<string, unknown> = {}) {
  return mount(STagInput, {
    props: { label: 'Tags', modelValue: [], suggestions: ['prod', 'eu', 'staging'], ...props },
    attachTo: document.body,
  })
}

function emitted(w: ReturnType<typeof mountTags>): string[][] {
  return (w.emitted('update:modelValue') ?? []).map(e => e[0] as string[])
}

async function type(w: ReturnType<typeof mountTags>, text: string, key = 'Enter') {
  const input = w.get('input')
  await input.setValue(text)
  await input.trigger('keydown', { key })
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('sTagInput', () => {
  it('adds typed text on Enter, trimmed and lowercased', async () => {
    const w = mountTags({ modelValue: ['eu'] })
    await type(w, '  Prod ')
    expect(emitted(w)).toEqual([['eu', 'prod']])
    expect((w.get('input').element as HTMLInputElement).value).toBe('')
  })

  it('adds on comma', async () => {
    const w = mountTags()
    await type(w, 'eu', ',')
    expect(emitted(w)).toEqual([['eu']])
  })

  it('does not add a tag that is already chosen', async () => {
    const w = mountTags({ modelValue: ['prod'] })
    await type(w, 'PROD')
    expect(emitted(w)).toEqual([])
  })

  it('suggests tags not chosen yet, filtered by the typed text, and picks one on click', async () => {
    const w = mountTags({ modelValue: ['eu'] })
    await w.get('input').trigger('focus')
    expect(w.findAll('li[role="option"]').map(o => o.text())).toEqual(['prod', 'staging'])

    await w.get('input').setValue('sta')
    const options = w.findAll('li[role="option"]')
    expect(options.map(o => o.text())).toEqual(['staging'])
    await options[0]!.trigger('click')
    expect(emitted(w)).toEqual([['eu', 'staging']])
  })

  it('removes a tag with its remove button', async () => {
    const w = mountTags({ modelValue: ['eu', 'prod'] })
    await w.get('button[aria-label="Remove tag eu"]').trigger('click')
    expect(emitted(w)).toEqual([['prod']])
  })

  it('removes the last tag on Backspace in an empty input', async () => {
    const w = mountTags({ modelValue: ['eu', 'prod'] })
    await w.get('input').trigger('keydown', { key: 'Backspace' })
    expect(emitted(w)).toEqual([['eu']])
  })

  it('refuses free text when allowCreate is false but accepts a suggestion', async () => {
    const w = mountTags({ allowCreate: false })
    await type(w, 'unknown')
    expect(emitted(w)).toEqual([])
    await type(w, 'pro')
    expect(emitted(w)).toEqual([['prod']])
  })

  it('renders chips only when readonly', () => {
    const w = mountTags({ modelValue: ['eu', 'prod'], readonly: true })
    expect(w.text()).toContain('eu')
    expect(w.text()).toContain('prod')
    expect(w.find('input').exists()).toBe(false)
    expect(w.find('button[aria-label^="Remove tag"]').exists()).toBe(false)
  })

  it('shows the error as an alert described by the input', async () => {
    const w = mountTags({ error: 'too many tags' })
    const alert = w.get('[role="alert"]')
    expect(alert.text()).toBe('too many tags')
    expect(w.get('input').attributes('aria-describedby')).toBe(alert.attributes('id'))
    expect(w.get('input').attributes('aria-invalid')).toBe('true')
    await expectNoAxeViolations(w.element)
  })
})
