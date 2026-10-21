import type { StreamRow } from '../../catalogStreams'
import type { MapperKind } from '../model'
import { validateMappers } from '@entities/connection'
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import TransformTab from '../TransformTab.vue'
import { setCel } from './celInput'

vi.mock('@entities/connection', () => ({
  validateMappers: vi.fn(async () => ({
    errors: [],
    warnings: [],
    outputSchema: { properties: {} },
    outputPrimaryKey: [],
    sampleOutput: undefined,
    sampleKept: true,
    sampleError: '',
  })),
}))

const schema = {
  properties: {
    id: { type: 'integer' },
    cust_id: { type: 'integer' },
    email: { type: 'string' },
    card: { type: 'string' },
    amount: { type: 'string' },
    created_at: { type: 'string' },
    status: { type: 'string' },
  },
}

function row(partial: Partial<StreamRow> = {}): StreamRow {
  return {
    name: 'orders',
    namespace: 'public',
    syncMode: 'full_refresh',
    destinationSyncMode: 'overwrite',
    enabled: true,
    supportedSyncModes: ['full_refresh'],
    cursorField: [],
    primaryKey: [['id']],
    selectedFields: [],
    jsonSchema: schema,
    sourceDefinedCursor: false,
    defaultCursorField: [],
    sourceDefinedPrimaryKey: [],
    mappers: [],
    ...partial,
  }
}

function mountTab(stream: StreamRow) {
  const wrapper = mount(TransformTab, {
    props: {
      'stream': stream,
      'target': { connectionId: 'conn-1' },
      'onUpdate:stream': (s: StreamRow) => wrapper.setProps({ stream: s }),
    },
    attachTo: document.body,
  })
  return wrapper
}

type Wrapper = ReturnType<typeof mountTab>

function lastMappers(wrapper: Wrapper) {
  const events = wrapper.emitted('update:stream') as [StreamRow][]
  return events.at(-1)![0].mappers
}

const labels: Record<MapperKind, string> = { rename: 'Rename', cast: 'Cast', set: 'Set', hash: 'Hash', mask: 'Mask', filter: 'Filter', drop: 'Drop' }

async function addStep(wrapper: Wrapper, kind: MapperKind) {
  await wrapper.get('[data-testid="add-step"]').trigger('click')
  const item = wrapper.findAll('[role="menuitem"]').find(b => b.text() === labels[kind])!
  await item.trigger('click')
}

function card(wrapper: Wrapper, index: number) {
  return wrapper.findAll('[data-testid="step-card"]')[index]
}

const CARD_H = 100
const CARD_GAP = 8
const PITCH = CARD_H + CARD_GAP

function rect(top: number, height: number): DOMRect {
  return { top, bottom: top + height, left: 0, right: 400, height, width: 400, x: 0, y: top, toJSON: () => ({}) } as DOMRect
}

function layoutSteps(wrapper: Wrapper) {
  const slots = wrapper.findAll('[data-testid="step-slot"]')
  wrapper.get('[data-testid="step-list"]').element.getBoundingClientRect = () => rect(0, slots.length * PITCH - CARD_GAP)
  slots.forEach((s, i) => {
    s.element.getBoundingClientRect = () => rect(i * PITCH, CARD_H)
  })
}

function pointer(type: string, clientY: number, clientX = 100) {
  window.dispatchEvent(new PointerEvent(type, { clientX, clientY, button: 0, bubbles: true }))
}

async function grab(wrapper: Wrapper, index: number) {
  layoutSteps(wrapper)
  await card(wrapper, index).get('[data-testid="step-drag"]').trigger('pointerdown', { button: 0, clientX: 100, clientY: index * PITCH + CARD_H / 2 })
}

async function moveTo(wrapper: Wrapper, clientY: number, clientX = 100) {
  pointer('pointermove', clientY, clientX)
  await wrapper.vm.$nextTick()
}

async function release(wrapper: Wrapper, clientY: number, clientX = 100) {
  pointer('pointerup', clientY, clientX)
  await wrapper.vm.$nextTick()
}

function slotStyle(wrapper: Wrapper, index: number) {
  return wrapper.findAll('[data-testid="step-slot"]')[index].attributes('style') ?? ''
}

function cardOrder(wrapper: Wrapper) {
  return wrapper.findAll('[data-testid="step-card"]').map(c => [c.get('header .tabular-nums').text(), c.attributes('data-kind')])
}

describe('transformTab', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.mocked(validateMappers).mockClear()
  })

  afterEach(() => {
    vi.useRealTimers()
    document.body.innerHTML = ''
  })

  it('links the expression reference from the steps and the filter', async () => {
    const wrapper = mountTab(row())
    await addStep(wrapper, 'filter')

    const links = wrapper.findAll('a[data-testid="expression-help"]')
    expect(links).toHaveLength(2)
    for (const link of links) {
      expect(link.attributes('href')).toBe('https://synclet-io.github.io/synclet-site/docs/guides/transforms/')
      expect(link.attributes('target')).toBe('_blank')
      expect(link.attributes('rel')).toContain('noopener')
    }
  })

  it('adds a step of every mapper kind and saves its form to the stream', async () => {
    const wrapper = mountTab(row())
    for (const kind of ['rename', 'cast', 'set', 'hash', 'mask', 'filter', 'drop'] as MapperKind[])
      await addStep(wrapper, kind)

    await card(wrapper, 0).get('input[data-testid="entry-field"]').setValue('cust_id')
    await card(wrapper, 0).get('input[data-testid="entry-target"]').setValue('customer_id')
    await card(wrapper, 1).get('input[data-testid="entry-field"]').setValue('amount')
    await card(wrapper, 1).get('select[data-testid="entry-cast-to"]').setValue('number')
    await card(wrapper, 2).get('input[data-testid="entry-field"]').setValue('currency')
    setCel(card(wrapper, 2).get('[data-testid="entry-expr"]'), '\'USD\'')
    await nextTick()
    await card(wrapper, 3).get('input[data-testid="entry-field"]').setValue('email')
    await card(wrapper, 4).get('input[data-testid="entry-field"]').setValue('card')
    await card(wrapper, 4).get('input[data-testid="mask-keep-last"]').setValue('4')
    setCel(card(wrapper, 5).get('[data-testid="filter-expr"]'), 'r.status != \'test\'')
    await nextTick()
    await card(wrapper, 5).get('select[data-testid="step-on-error"]').setValue('skip')
    await card(wrapper, 6).get('input[data-testid="entry-field"]').setValue('status')
    expect(card(wrapper, 6).find('select[data-testid="step-on-error"]').exists()).toBe(false)

    expect(lastMappers(wrapper)).toEqual([
      { rename: { cust_id: 'customer_id' } },
      { cast: { amount: 'number' } },
      { set: { currency: '\'USD\'' } },
      { hash: ['email'] },
      { mask: { card: { keep_last: 4 } } },
      { filter: 'r.status != \'test\'', on_error: 'skip' },
      { drop: ['status'] },
    ])
  })

  it('writes a timestamp cast with its format and timezone', async () => {
    const wrapper = mountTab(row())
    await addStep(wrapper, 'cast')
    const c = card(wrapper, 0)
    await c.get('input[data-testid="entry-field"]').setValue('created_at')
    await c.get('select[data-testid="entry-cast-to"]').setValue('timestamp')
    await c.get('input[data-testid="cast-format"]').setValue('dd.MM.yyyy HH:mm')
    await c.get('input[data-testid="cast-timezone"]').setValue('Europe/Kyiv')

    expect(lastMappers(wrapper)).toEqual([
      { cast: { created_at: { to: 'timestamp', format: 'dd.MM.yyyy HH:mm', timezone: 'Europe/Kyiv' } } },
    ])
  })

  it('moves a step up and removes one', async () => {
    const wrapper = mountTab(row({ mappers: [{ hash: ['email'] }, { filter: 'true' }, { rename: { cust_id: 'customer_id' } }] }))
    await card(wrapper, 1).get('[data-testid="step-up"]').trigger('click')
    expect(lastMappers(wrapper)).toEqual([{ filter: 'true' }, { hash: ['email'] }, { rename: { cust_id: 'customer_id' } }])

    await card(wrapper, 0).get('[data-testid="step-remove"]').trigger('click')
    expect(lastMappers(wrapper)).toEqual([{ hash: ['email'] }, { rename: { cust_id: 'customer_id' } }])
    expect(wrapper.findAll('[data-testid="step-card"]')).toHaveLength(2)
  })

  it('shows the slot a dragged step will land in while the other cards move aside', async () => {
    const wrapper = mountTab(row({ mappers: [{ hash: ['email'] }, { filter: 'true' }, { rename: { cust_id: 'customer_id' } }] }))
    await grab(wrapper, 0)
    await moveTo(wrapper, CARD_H / 2 + 2 * PITCH + 10)

    const placeholder = wrapper.get('[data-testid="drop-placeholder"]')
    expect(placeholder.attributes('style')).toContain(`top: ${2 * PITCH}px`)
    expect(placeholder.attributes('style')).toContain(`height: ${CARD_H}px`)
    expect(slotStyle(wrapper, 0)).toContain(`translateY(${2 * PITCH + 10}px)`)
    expect(slotStyle(wrapper, 1)).toContain(`translateY(-${PITCH}px)`)
    expect(slotStyle(wrapper, 2)).toContain(`translateY(-${PITCH}px)`)
    expect(wrapper.findAll('[data-testid="step-card"]')).toHaveLength(3)
    expect(card(wrapper, 0).classes()).toContain('ring-primary')
    expect(wrapper.emitted('update:stream')).toBeUndefined()
  })

  it('drops a dragged step into the last slot', async () => {
    const wrapper = mountTab(row({ mappers: [{ hash: ['email'] }, { filter: 'true' }, { rename: { cust_id: 'customer_id' } }] }))
    await grab(wrapper, 0)
    await moveTo(wrapper, CARD_H / 2 + 2 * PITCH + 10)
    await release(wrapper, CARD_H / 2 + 2 * PITCH + 10)

    expect(lastMappers(wrapper)).toEqual([{ filter: 'true' }, { rename: { cust_id: 'customer_id' } }, { hash: ['email'] }])
    expect(cardOrder(wrapper)).toEqual([['1', 'filter'], ['2', 'rename'], ['3', 'hash']])
    expect(wrapper.find('[data-testid="drop-placeholder"]').exists()).toBe(false)
    expect(slotStyle(wrapper, 0)).toBe('')
  })

  it('drops a dragged step into the first slot and the columns and checks follow', async () => {
    const wrapper = mountTab(row({ mappers: [{ filter: 'true' }, { hash: ['email'] }, { rename: { email: 'contact' } }] }))
    expect(card(wrapper, 1).findAll('[data-testid="step-issue"]')).toHaveLength(0)

    await grab(wrapper, 2)
    await moveTo(wrapper, 2 * PITCH + CARD_H / 2 - 2 * PITCH - 10)
    await release(wrapper, 10)

    expect(lastMappers(wrapper)).toEqual([{ rename: { email: 'contact' } }, { filter: 'true' }, { hash: ['email'] }])
    expect(cardOrder(wrapper)).toEqual([['1', 'rename'], ['2', 'filter'], ['3', 'hash']])
    const outputs = wrapper.findAll('[data-testid="output-column"] .font-mono.text-heading').map(c => c.text())
    expect(outputs).toContain('contact')
    expect(outputs).not.toContain('email')
    expect(card(wrapper, 2).findAll('[data-testid="step-issue"]').map(i => i.attributes('data-severity'))).toContain('error')
  })

  it('leaves the order unchanged on Escape or a drop outside the list', async () => {
    const wrapper = mountTab(row({ mappers: [{ hash: ['email'] }, { filter: 'true' }, { rename: { cust_id: 'customer_id' } }] }))
    await grab(wrapper, 0)
    await moveTo(wrapper, CARD_H / 2 + 2 * PITCH + 10)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="drop-placeholder"]').exists()).toBe(false)
    expect(slotStyle(wrapper, 0)).toBe('')
    await release(wrapper, CARD_H / 2 + 2 * PITCH + 10)
    expect(wrapper.emitted('update:stream')).toBeUndefined()

    await grab(wrapper, 0)
    await moveTo(wrapper, CARD_H / 2 + 2 * PITCH + 10)
    await moveTo(wrapper, 3 * PITCH + 200)
    expect(wrapper.find('[data-testid="drop-placeholder"]').exists()).toBe(false)
    await release(wrapper, 3 * PITCH + 200)
    expect(wrapper.emitted('update:stream')).toBeUndefined()
    expect(wrapper.find('[data-testid="drop-placeholder"]').exists()).toBe(false)
    expect(cardOrder(wrapper)).toEqual([['1', 'hash'], ['2', 'filter'], ['3', 'rename']])
  })

  it('drops into the shown slot when the pointer drifts sideways off the list', async () => {
    const wrapper = mountTab(row({ mappers: [{ hash: ['email'] }, { filter: 'true' }, { rename: { cust_id: 'customer_id' } }] }))
    await grab(wrapper, 0)
    await moveTo(wrapper, CARD_H / 2 + 2 * PITCH + 10, 1000)
    expect(wrapper.get('[data-testid="drop-placeholder"]').attributes('style')).toContain(`top: ${2 * PITCH}px`)
    await release(wrapper, CARD_H / 2 + 2 * PITCH + 10, 1000)

    expect(lastMappers(wrapper)).toEqual([{ filter: 'true' }, { rename: { cust_id: 'customer_id' } }, { hash: ['email'] }])
  })

  it('starts a drag from the handle only', async () => {
    const wrapper = mountTab(row({ mappers: [{ rename: { cust_id: 'customer_id' } }, { filter: 'true' }] }))
    layoutSteps(wrapper)
    await card(wrapper, 0).get('input[data-testid="entry-field"]').trigger('pointerdown', { button: 0, clientY: 50 })
    expect(wrapper.find('[data-testid="drop-placeholder"]').exists()).toBe(false)

    expect(card(wrapper, 0).get('[data-testid="step-drag"]').attributes('tabindex')).toBe('-1')
    await card(wrapper, 0).get('[data-testid="step-drag"]').trigger('pointerdown', { button: 2, clientY: 50 })
    expect(wrapper.find('[data-testid="drop-placeholder"]').exists()).toBe(false)

    await card(wrapper, 0).get('[data-testid="step-drag"]').trigger('pointerdown', { button: 0, clientY: 50 })
    expect(wrapper.find('[data-testid="drop-placeholder"]').exists()).toBe(true)
    await release(wrapper, 50)
  })

  it('flags a rename onto an existing field on its card without calling the server', async () => {
    const wrapper = mountTab(row({ mappers: [{ hash: ['card'] }, { rename: { cust_id: 'email' } }] }))
    await vi.advanceTimersByTimeAsync(1000)

    const issues = card(wrapper, 1).findAll('[data-testid="step-issue"]')
    expect(issues.map(i => [i.attributes('data-severity'), i.text()])).toEqual([['error', 'field "email" already exists']])
    expect(card(wrapper, 0).findAll('[data-testid="step-issue"]')).toHaveLength(0)
    expect(validateMappers).not.toHaveBeenCalled()
  })

  it('marks the stream as having transform errors when one step lists a field twice', async () => {
    const wrapper = mountTab(row())
    await addStep(wrapper, 'rename')
    const c = card(wrapper, 0)
    await c.get('[data-testid="entry-add"]').trigger('click')
    const fields = c.findAll('input[data-testid="entry-field"]')
    const targets = c.findAll('input[data-testid="entry-target"]')
    await fields[0].setValue('cust_id')
    await targets[0].setValue('customer_id')
    await fields[1].setValue('cust_id')
    await targets[1].setValue('client_id')

    const events = wrapper.emitted('update:stream') as [StreamRow][]
    expect(events.at(-1)![0].transformErrors).toBe(1)
    expect(c.findAll('[data-testid="step-issue"]').map(i => i.text())).toEqual(['field "cust_id" is listed twice in this step'])

    await c.findAll('[data-testid="entry-remove"]')[1].trigger('click')
    expect((wrapper.emitted('update:stream') as [StreamRow][]).at(-1)![0].transformErrors).toBe(0)
  })

  it('warns on a filter card of a dedup stream', async () => {
    const wrapper = mountTab(row({ destinationSyncMode: 'append_dedup', mappers: [{ hash: ['email'] }, { filter: 'r.id > 0' }] }))
    const issues = card(wrapper, 1).findAll('[data-testid="step-issue"]')
    expect(issues.map(i => [i.attributes('data-severity'), i.text()])).toEqual([['warning', 'Rows filtered out later stay in the destination']])
    expect(card(wrapper, 0).findAll('[data-testid="step-issue"]')).toHaveLength(0)
  })

  it('shows a server warning on its step card', async () => {
    const message = 'field "status" is not selected; sources that honour field selection will not send it — select it and drop it after use'
    vi.mocked(validateMappers).mockResolvedValue({
      errors: [],
      warnings: [{ step: 0, field: 'status', message }],
      outputSchema: { properties: {} },
      outputPrimaryKey: [],
      sampleOutput: undefined,
      sampleKept: true,
      sampleError: '',
    })
    const wrapper = mountTab(row({ selectedFields: [{ fieldPath: ['id'] }], mappers: [{ set: { x: 'r.status' } }, { drop: ['x'] }] }))
    await vi.advanceTimersByTimeAsync(1000)

    const issues = card(wrapper, 0).findAll('[data-testid="step-issue"]')
    expect(issues.map(i => [i.attributes('data-severity'), i.text()])).toEqual([['warning', message]])
    expect(card(wrapper, 1).findAll('[data-testid="step-issue"]')).toHaveLength(0)
  })

  it('flags a drop of the source-defined cursor of an incremental stream on its card', async () => {
    const wrapper = mountTab(row({ syncMode: 'incremental', sourceDefinedCursor: true, defaultCursorField: ['created_at'], mappers: [{ drop: ['created_at'] }] }))
    const issues = card(wrapper, 0).findAll('[data-testid="step-issue"]')
    expect(issues.map(i => [i.attributes('data-severity'), i.text()])).toEqual([
      ['error', 'cursor field "created_at" cannot be dropped in incremental mode; choose another cursor in the stream settings or keep the field'],
    ])
  })
})
