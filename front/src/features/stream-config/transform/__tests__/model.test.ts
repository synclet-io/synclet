import type { StepJson } from '../model'
import { describe, expect, it } from 'vitest'
import { addStep, fromStepJson, moveStep, removeStep, toStepJson, updateStep } from '../model'

function roundTrip(json: StepJson): StepJson {
  return toStepJson(fromStepJson(json))
}

describe('step model', () => {
  it.each<[string, StepJson]>([
    ['rename', { rename: { cust_id: 'customer_id', amt: 'amount' } }],
    ['cast', { cast: { amount: 'number', created_at: 'timestamp' }, on_error: 'null' }],
    ['set', { set: { currency: '\'USD\'', total_cents: 'int(r.amount * 100.0)' }, on_error: 'skip' }],
    ['hash', { hash: ['email', 'phone'] }],
    ['mask', { mask: { card: { keep_last: 4 }, phone: { keep_first: 2, keep_last: 2 } } }],
    ['filter', { filter: 'r.status != \'test\'', on_error: 'skip' }],
  ])('round-trips a %s step', (_kind, json) => {
    expect(roundTrip(json)).toEqual(json)
  })

  it('round-trips a cast with every option', () => {
    const json = {
      cast: {
        created_at: {
          to: 'timestamp',
          format: 'dd.MM.yyyy HH:mm',
          timezone: 'Europe/Kyiv',
          decimal_separator: ',',
          thousands_separator: ' ',
          trim: true,
          strip_prefix: '$',
          strip_suffix: '%',
          true_values: ['yes', 'Y'],
          false_values: ['no', 'N'],
          source: 'r.code.split(\'-\')[0]',
          on_error: 'null',
        },
      },
    }
    expect(roundTrip(json)).toEqual(json)
  })

  it('writes a cast with only a target as a bare string and leaves empty options out', () => {
    const step = fromStepJson({ cast: { amount: { to: 'number' } } })
    step.entries[0].cast = { ...step.entries[0].cast!, format: '', trim: false, trueValues: [], source: '' }
    expect(toStepJson(step)).toEqual({ cast: { amount: 'number' } })
  })

  it('adds, moves, updates and removes steps without changing the input list', () => {
    const steps = addStep(addStep([], 'rename'), 'filter')
    expect(steps.map(s => s.kind)).toEqual(['rename', 'filter'])

    const moved = moveStep(steps, 1, -1)
    expect(moved.map(s => s.kind)).toEqual(['filter', 'rename'])
    expect(steps.map(s => s.kind)).toEqual(['rename', 'filter'])
    expect(moveStep(moved, 0, -1)).toEqual(moved)

    const updated = updateStep(moved, 0, { ...moved[0], filter: 'r.ok' })
    expect(toStepJson(updated[0])).toEqual({ filter: 'r.ok' })

    expect(removeStep(updated, 0).map(s => s.kind)).toEqual(['rename'])
  })
})
