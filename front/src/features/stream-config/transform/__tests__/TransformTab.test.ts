import type { StreamRow } from '../../catalogStreams'
import type { MapperKind } from '../model'
import { validateMappers } from '@entities/connection'
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import TransformTab from '../TransformTab.vue'

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
      'connectionId': 'conn-1',
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

const labels: Record<MapperKind, string> = { rename: 'Rename', cast: 'Cast', set: 'Set', hash: 'Hash', mask: 'Mask', filter: 'Filter' }

async function addStep(wrapper: Wrapper, kind: MapperKind) {
  await wrapper.get('[data-testid="add-step"]').trigger('click')
  const item = wrapper.findAll('[role="menuitem"]').find(b => b.text() === labels[kind])!
  await item.trigger('click')
}

function card(wrapper: Wrapper, index: number) {
  return wrapper.findAll('[data-testid="step-card"]')[index]
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

  it('adds a step of every mapper kind and saves its form to the stream', async () => {
    const wrapper = mountTab(row())
    for (const kind of ['rename', 'cast', 'set', 'hash', 'mask', 'filter'] as MapperKind[])
      await addStep(wrapper, kind)

    await card(wrapper, 0).get('input[data-testid="entry-field"]').setValue('cust_id')
    await card(wrapper, 0).get('input[data-testid="entry-target"]').setValue('customer_id')
    await card(wrapper, 1).get('input[data-testid="entry-field"]').setValue('amount')
    await card(wrapper, 1).get('select[data-testid="entry-cast-to"]').setValue('number')
    await card(wrapper, 2).get('input[data-testid="entry-field"]').setValue('currency')
    await card(wrapper, 2).get('input[data-testid="entry-expr"]').setValue('\'USD\'')
    await card(wrapper, 3).get('input[data-testid="entry-field"]').setValue('email')
    await card(wrapper, 4).get('input[data-testid="entry-field"]').setValue('card')
    await card(wrapper, 4).get('input[data-testid="mask-keep-last"]').setValue('4')
    await card(wrapper, 5).get('textarea[data-testid="filter-expr"]').setValue('r.status != \'test\'')
    await card(wrapper, 5).get('select[data-testid="step-on-error"]').setValue('skip')

    expect(lastMappers(wrapper)).toEqual([
      { rename: { cust_id: 'customer_id' } },
      { cast: { amount: 'number' } },
      { set: { currency: '\'USD\'' } },
      { hash: ['email'] },
      { mask: { card: { keep_last: 4 } } },
      { filter: 'r.status != \'test\'', on_error: 'skip' },
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
})
