import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import SButton from '../SButton.vue'
import SCronEditor from '../SCronEditor.vue'
import SDropdown from '../SDropdown.vue'
import SInput from '../SInput.vue'
import SModal from '../SModal.vue'
import SPagination from '../SPagination.vue'
import SSelect from '../SSelect.vue'
import STabs from '../STabs.vue'
import SToast from '../SToast.vue'
import { TOAST_DURATIONS, useToast } from '../useToast'
import { expectNoAxeViolations } from './axe'

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { render: () => null } },
      { path: '/a', name: 'a', component: { render: () => null } },
      { path: '/b', name: 'b', component: { render: () => null } },
    ],
  })
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('sInput', () => {
  it('associates label, hint and error with the input', async () => {
    const w = mount(SInput, {
      props: { label: 'Email', hint: 'Work address' },
      attachTo: document.body,
    })
    const input = w.get('input')
    const label = w.get('label')
    expect(label.attributes('for')).toBe(input.attributes('id'))
    const hintId = input.attributes('aria-describedby')
    expect(w.get(`#${hintId}`).text()).toBe('Work address')
    expect(input.attributes('aria-invalid')).toBeUndefined()

    await w.setProps({ error: 'Required' })
    expect(input.attributes('aria-invalid')).toBe('true')
    expect(w.get(`#${input.attributes('aria-describedby')}`).text()).toBe('Required')
    await expectNoAxeViolations()
    w.unmount()
  })

  it('forwards undeclared attributes to the input and keeps class on the wrapper', () => {
    const w = mount(SInput, {
      props: { label: 'Retries', type: 'number' },
      attrs: { 'readonly': true, 'min': '0', 'max': '10', 'class': 'col-span-2', 'data-test': 'x' },
    })
    const input = w.get('input')
    expect(input.attributes('readonly')).toBeDefined()
    expect(input.attributes('min')).toBe('0')
    expect(input.attributes('max')).toBe('10')
    expect(input.attributes('data-test')).toBe('x')
    expect(input.classes()).not.toContain('col-span-2')
    expect(w.classes()).toContain('col-span-2')
    expect(w.attributes('min')).toBeUndefined()
  })

  it('uses a caller-supplied id', () => {
    const w = mount(SInput, { props: { label: 'Name' }, attrs: { id: 'name' } })
    expect(w.get('input').attributes('id')).toBe('name')
    expect(w.get('label').attributes('for')).toBe('name')
  })
})

describe('sSelect', () => {
  it('associates label and error with the select and forwards attributes', async () => {
    const w = mount(SSelect, {
      props: { label: 'Policy', options: [{ label: 'A', value: 'a' }], error: 'Pick one' },
      attrs: { name: 'policy', class: 'w-40' },
      attachTo: document.body,
    })
    const select = w.get('select')
    expect(w.get('label').attributes('for')).toBe(select.attributes('id'))
    expect(select.attributes('aria-invalid')).toBe('true')
    expect(w.get(`#${select.attributes('aria-describedby')}`).text()).toBe('Pick one')
    expect(select.attributes('name')).toBe('policy')
    expect(w.classes()).toContain('w-40')
    await expectNoAxeViolations()
    w.unmount()
  })
})

describe('sCronEditor', () => {
  it('labels the preset select and the custom input', async () => {
    const w = mount(SCronEditor, {
      props: { label: 'Schedule', modelValue: 'not a cron' },
      attachTo: document.body,
    })
    expect(w.get('label').attributes('for')).toBe(w.get('select').attributes('id'))
    const input = w.get('input')
    expect(input.attributes('aria-label')).toBe('Cron expression')
    expect(input.attributes('aria-invalid')).toBe('true')
    expect(w.find(`#${input.attributes('aria-describedby')}`).exists()).toBe(true)
    await expectNoAxeViolations()
    w.unmount()
  })
})

describe('sModal', () => {
  function mountModal(open = true) {
    const opener = document.createElement('button')
    opener.textContent = 'Open'
    document.body.appendChild(opener)
    opener.focus()
    const w = mount(SModal, {
      props: { open, title: 'Delete source' },
      slots: {
        default: () => h('input', { 'aria-label': 'Name', 'id': 'first' }),
        footer: () => h('button', { id: 'last' }, 'Confirm'),
      },
      attachTo: document.body,
    })
    return { w, opener }
  }

  it('exposes dialog semantics labelled by the title', async () => {
    const { w } = mountModal()
    await nextTick()
    const dialog = document.querySelector('[role="dialog"]')!
    expect(dialog.getAttribute('aria-modal')).toBe('true')
    const titleId = dialog.getAttribute('aria-labelledby')!
    expect(document.getElementById(titleId)?.textContent?.trim()).toBe('Delete source')
    expect(document.querySelector('button[aria-label="Close"]')?.getAttribute('type')).toBe('button')
    await expectNoAxeViolations()
    w.unmount()
  })

  it('moves focus in, traps Tab and restores focus on close', async () => {
    const { w, opener } = mountModal()
    await nextTick()
    await nextTick()
    const first = document.getElementById('first')!
    const last = document.getElementById('last')!
    const close = document.querySelector<HTMLElement>('button[aria-label="Close"]')!
    expect(document.activeElement).toBe(first)

    last.focus()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true }))
    expect(document.activeElement).toBe(close)

    close.focus()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true }))
    expect(document.activeElement).toBe(last)

    await w.setProps({ open: false })
    expect(document.activeElement).toBe(opener)
    w.unmount()
  })

  it('closes on Escape, only the topmost modal', async () => {
    const outer = mount(SModal, { props: { open: true, title: 'Outer' }, attachTo: document.body })
    const inner = mount(SModal, { props: { open: true, title: 'Inner' }, attachTo: document.body })
    await nextTick()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    expect(inner.emitted('close')).toHaveLength(1)
    expect(outer.emitted('close')).toBeUndefined()
    inner.unmount()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    expect(outer.emitted('close')).toHaveLength(1)
    outer.unmount()
  })
})

describe('sDropdown', () => {
  const Host = defineComponent({
    components: { SDropdown },
    setup() {
      const picked = ref('')
      const items = [
        { label: 'Alpha', active: true, onClick: () => { picked.value = 'Alpha' } },
        { label: 'Beta', onClick: () => { picked.value = 'Beta' } },
      ]
      return { items, picked }
    },
    template: `
      <SDropdown :items="items">
        <template #trigger="{ attrs }">
          <button id="trigger" v-bind="attrs">Workspace</button>
        </template>
      </SDropdown>
    `,
  })

  it('exposes menu semantics and supports keyboard navigation', async () => {
    const w = mount(Host, { attachTo: document.body })
    const trigger = w.get('#trigger')
    expect(trigger.attributes('aria-haspopup')).toBe('menu')
    expect(trigger.attributes('aria-expanded')).toBe('false')

    ;(trigger.element as HTMLElement).focus()
    await trigger.trigger('keydown', { key: 'ArrowDown' })
    await nextTick()
    expect(trigger.attributes('aria-expanded')).toBe('true')
    const menu = w.get('[role="menu"]')
    expect(trigger.attributes('aria-controls')).toBe(menu.attributes('id'))
    const items = w.findAll('[role="menuitem"]')
    expect(items).toHaveLength(2)
    expect(items[0].attributes('aria-current')).toBe('true')
    expect(document.activeElement).toBe(items[0].element)

    await items[0].trigger('keydown', { key: 'ArrowDown' })
    expect(document.activeElement).toBe(items[1].element)
    await items[1].trigger('keydown', { key: 'ArrowDown' })
    expect(document.activeElement).toBe(items[0].element)
    await items[0].trigger('keydown', { key: 'End' })
    expect(document.activeElement).toBe(items[1].element)
    await expectNoAxeViolations()

    await items[1].trigger('keydown', { key: 'Escape' })
    expect(w.find('[role="menu"]').exists()).toBe(false)
    expect(document.activeElement).toBe(trigger.element)
    w.unmount()
  })
})

describe('sTabs', () => {
  const tabs = [
    { name: 'Sources', value: 'source' },
    { name: 'Destinations', value: 'destination' },
    { name: 'Repositories', value: 'repositories' },
  ]

  it('renders value tabs as a tablist with roving focus', async () => {
    const w = mount(STabs, {
      props: {
        'tabs': tabs,
        'modelValue': 'source',
        'onUpdate:modelValue': (v: string) => w.setProps({ modelValue: v }),
      },
      attachTo: document.body,
    })
    expect(w.find('[role="tablist"]').exists()).toBe(true)
    const buttons = w.findAll('[role="tab"]')
    expect(buttons.map(b => b.attributes('aria-selected'))).toEqual(['true', 'false', 'false'])
    expect(buttons.map(b => b.attributes('tabindex'))).toEqual(['0', '-1', '-1'])

    await buttons[0].trigger('keydown', { key: 'ArrowLeft' })
    await nextTick()
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual(['repositories'])
    expect(document.activeElement).toBe(buttons[2].element)
    expect(buttons[2].attributes('aria-selected')).toBe('true')
    await expectNoAxeViolations()
    w.unmount()
  })

  it('marks the active route link with aria-current', async () => {
    const router = makeRouter()
    router.push('/b')
    await router.isReady()
    const w = mount(STabs, {
      props: { tabs: [{ name: 'A', to: { name: 'a' } }, { name: 'B', to: { name: 'b' } }] },
      global: { plugins: [router] },
      attachTo: document.body,
    })
    const links = w.findAll('a')
    expect(links.map(l => l.attributes('aria-current'))).toEqual([undefined, 'page'])
    expect(w.find('[role="tablist"]').exists()).toBe(false)
    await expectNoAxeViolations()
    w.unmount()
  })
})

describe('sToast', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    const { toasts, dismiss } = useToast()
    toasts.value.forEach(t => dismiss(t.id))
    vi.useRealTimers()
  })

  it('announces toasts through a live region with alert role for errors', async () => {
    const w = mount(SToast, { attachTo: document.body })
    const toast = useToast()
    toast.success('Saved')
    toast.error('Failed to save')
    await nextTick()
    const region = document.querySelector('[aria-live]')!
    expect(region.getAttribute('aria-live')).toBe('polite')
    expect(region.querySelector('[role="status"]')?.textContent).toContain('Saved')
    expect(region.querySelector('[role="alert"]')?.textContent).toContain('Failed to save')
    expect(region.querySelectorAll('button[aria-label="Dismiss notification"]')).toHaveLength(2)
    vi.useRealTimers()
    await expectNoAxeViolations()
    w.unmount()
  })

  it('keeps errors longer than other toasts and pauses while hovered', async () => {
    const w = mount(SToast, { attachTo: document.body })
    const { toasts, success, error } = useToast()
    success('Saved')
    error('Failed')
    await nextTick()

    vi.advanceTimersByTime(TOAST_DURATIONS.success)
    expect(toasts.value.map(t => t.message)).toEqual(['Failed'])

    const el = document.querySelector<HTMLElement>('[role="alert"]')!
    el.dispatchEvent(new MouseEvent('mouseenter'))
    vi.advanceTimersByTime(TOAST_DURATIONS.error)
    expect(toasts.value.map(t => t.message)).toEqual(['Failed'])

    el.dispatchEvent(new MouseEvent('mouseleave'))
    vi.advanceTimersByTime(TOAST_DURATIONS.error)
    expect(toasts.value).toEqual([])
    w.unmount()
  })
})

describe('sButton', () => {
  it('defaults to type="button" and honours an explicit type', () => {
    expect(mount(SButton, { slots: { default: 'Go' } }).get('button').attributes('type')).toBe('button')
    expect(mount(SButton, { props: { type: 'submit' }, slots: { default: 'Go' } }).get('button').attributes('type')).toBe('submit')
  })

  it('does not submit the surrounding form by default', async () => {
    const onSubmit = vi.fn((e: Event) => e.preventDefault())
    const w = mount({
      components: { SButton },
      setup: () => ({ onSubmit }),
      template: '<form @submit="onSubmit"><SButton>Test</SButton></form>',
    }, { attachTo: document.body })
    await w.get('button').trigger('click')
    expect(onSubmit).not.toHaveBeenCalled()
    w.unmount()
  })

  it('omits type when rendered as a link', async () => {
    const router = makeRouter()
    const w = mount(SButton, { props: { to: '/a' }, slots: { default: 'Go' }, global: { plugins: [router] } })
    expect(w.get('a').attributes('type')).toBeUndefined()
  })
})

describe('sPagination', () => {
  it('names every control and marks the current page', async () => {
    const w = mount(SPagination, {
      props: { total: 100, pageSize: 10, currentPage: 5 },
      attachTo: document.body,
    })
    const nav = w.get('nav')
    expect(nav.attributes('aria-label')).toBe('Pagination')
    const labels = w.findAll('button').map(b => b.attributes('aria-label'))
    expect(labels).toEqual(expect.arrayContaining(['First page', 'Previous page', 'Next page', 'Last page']))
    expect(w.findAll('button').every(b => b.attributes('type') === 'button')).toBe(true)
    expect(w.get('[aria-current="page"]').text()).toBe('5')
    await expectNoAxeViolations()
    w.unmount()
  })
})
