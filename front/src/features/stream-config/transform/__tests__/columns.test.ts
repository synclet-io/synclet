import type { DestinationSyncMode, SelectedField } from '@entities/connection'
import type { Column } from '../columns'
import type { StepJson } from '../model'
import { describe, expect, it } from 'vitest'
import { schemaColumns, walkColumns } from '../columns'
import { fromStepJson } from '../model'

const input: Column[] = [
  { name: 'id', type: 'integer' },
  { name: 'cust_id', type: 'integer' },
  { name: 'email', type: 'string' },
  { name: 'amount', type: 'string' },
  { name: 'status', type: 'string' },
]

function walk(steps: StepJson[], opts: { selected?: SelectedField[], pk?: string[][], mode?: DestinationSyncMode, columns?: Column[] } = {}) {
  return walkColumns(opts.columns ?? input, steps.map(fromStepJson), opts.selected ?? [], opts.pk ?? [['id']], opts.mode ?? 'overwrite')
}

function errors(steps: StepJson[], opts: Parameters<typeof walk>[1] = {}) {
  return walk(steps, opts).issues.filter(i => i.severity === 'error').map(({ step, field, message }) => ({ step, field, message }))
}

describe('schemaColumns', () => {
  it('reads top-level properties with their types', () => {
    expect(schemaColumns({
      properties: {
        id: { type: ['null', 'integer'] },
        created_at: { type: 'string', format: 'date-time' },
        day: { type: 'string', format: 'date' },
        meta: { type: 'object' },
        raw: {},
      },
    })).toEqual([
      { name: 'id', type: 'integer' },
      { name: 'created_at', type: 'timestamp' },
      { name: 'day', type: 'date' },
      { name: 'meta', type: 'object' },
      { name: 'raw', type: 'any' },
    ])
  })
})

describe('walkColumns', () => {
  it('walks a clean chain and marks new, renamed, retyped and dropped columns', () => {
    const result = walk([
      { rename: { cust_id: 'customer_id' } },
      { cast: { amount: 'number' } },
      { set: { currency: '\'USD\'' } },
      { hash: ['email'] },
      { filter: 'r.status != \'test\'' },
    ], { selected: [{ fieldPath: ['id'] }, { fieldPath: ['cust_id'] }, { fieldPath: ['amount'] }, { fieldPath: ['email'] }] })

    expect(result.issues).toEqual([])
    expect(result.stepInputs[1].map(c => c.name)).toEqual(['id', 'customer_id', 'email', 'amount', 'status'])
    const byName = Object.fromEntries(result.output.map(c => [c.name, c]))
    expect(byName.customer_id).toMatchObject({ type: 'integer', marks: ['renamed'], renamedFrom: 'cust_id' })
    expect(byName.amount).toMatchObject({ type: 'number', marks: ['retyped'] })
    expect(byName.currency.marks).toEqual(['new'])
    expect(byName.email).toMatchObject({ type: 'string', marks: [] })
    expect(byName.status.marks).toEqual(['dropped'])
    expect(byName.id).toMatchObject({ marks: [], primaryKey: true })
  })

  it('flags two steps creating the same field', () => {
    expect(errors([{ set: { total: '1' } }, { set: { total: '2' } }]))
      .toEqual([{ step: 1, field: 'total', message: 'field "total" already exists (created by step 1)' }])
  })

  it('flags a rename onto a field that exists', () => {
    expect(errors([{ rename: { cust_id: 'email' } }]))
      .toEqual([{ step: 0, field: 'email', message: 'field "email" already exists' }])
  })

  it('allows renames that swap two fields in one step', () => {
    expect(errors([{ rename: { cust_id: 'email', email: 'cust_id' } }])).toEqual([])
  })

  it('ignores a renamed field named inside a string literal', () => {
    expect(errors([
      { rename: { cust_id: 'customer_id' } },
      { set: { r: '\'r.cust_id\'', note: '"see r.cust_id" + \'it\\\'s r.cust_id\'' } },
      { filter: 'r.status != \'r.cust_id\'' },
    ])).toEqual([])
  })

  it('flags the use of a field an earlier step renamed away', () => {
    expect(errors([{ rename: { cust_id: 'customer_id' } }, { hash: ['cust_id'] }, { filter: 'r.cust_id > 0' }])).toEqual([
      { step: 1, field: 'cust_id', message: 'field "cust_id" was renamed by step 1' },
      { step: 2, field: 'cust_id', message: 'field "cust_id" was renamed by step 1' },
    ])
  })

  it.each<[string, StepJson]>([
    ['rename', { rename: { nope: 'x' } }],
    ['cast', { cast: { nope: 'integer' } }],
    ['hash', { hash: ['nope'] }],
    ['mask', { mask: { nope: { keep_last: 4 } } }],
  ])('flags a %s of a field missing at that step', (_kind, step) => {
    expect(errors([step])).toEqual([{ step: 0, field: 'nope', message: 'field "nope" does not exist' }])
  })

  it('flags an unchecked primary key in a dedup mode and shows a renamed one', () => {
    expect(errors([], { mode: 'append_dedup', selected: [{ fieldPath: ['email'] }] }))
      .toEqual([{ step: -1, field: 'id', message: 'primary key field "id" is not selected' }])

    const renamed = walk([{ rename: { id: 'order_id' } }], { mode: 'append_dedup', selected: [{ fieldPath: ['id'] }] })
    expect(renamed.issues).toEqual([])
    expect(renamed.output.find(c => c.name === 'order_id')).toMatchObject({ primaryKey: true, marks: ['renamed'] })
  })

  it.each([
    ['', 'field name is empty'],
    [' padded', 'invalid field name " padded"'],
    ['a\u0007b', 'invalid field name "a\u0007b"'],
    ['x'.repeat(256), `invalid field name "${'x'.repeat(256)}"`],
  ])('flags an invalid destination field name %j', (name, message) => {
    expect(errors([{ set: { [name]: '1' } }])).toEqual([{ step: 0, field: name, message }])
  })

  it('warns about a filter on a dedup stream and on a CDC stream', () => {
    const warning = { step: 0, field: '', message: 'Rows filtered out later stay in the destination', severity: 'warning' }
    expect(walk([{ filter: 'true' }], { mode: 'append_dedup' }).issues).toEqual([warning])
    expect(walk([{ filter: 'true' }], { columns: [...input, { name: '_ab_cdc_lsn', type: 'integer' }] }).issues).toEqual([warning])
    expect(walk([{ filter: 'true' }]).issues).toEqual([])
  })
})
