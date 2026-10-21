import type { WorkflowStepInput } from '@entities/workflow'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { removeStep } from '../steps'
import WorkflowStepsEditor from '../WorkflowStepsEditor.vue'

const connections = [
  { id: 'a', name: 'Orders' },
  { id: 'b', name: 'Customers' },
  { id: 'c', name: 'Revenue' },
]

function mountEditor(steps: WorkflowStepInput[], readonly = false) {
  return mount(WorkflowStepsEditor, { props: { modelValue: steps, connections, readonly } })
}

function lastEmitted(w: ReturnType<typeof mountEditor>): WorkflowStepInput[] {
  const events = w.emitted('update:modelValue')!
  return events.at(-1)![0] as WorkflowStepInput[]
}

describe('workflowStepsEditor', () => {
  it('lists the steps with a multi-select of the steps each runs after', () => {
    const w = mountEditor([
      { connectionId: 'a', afterConnectionIds: [] },
      { connectionId: 'b', afterConnectionIds: [] },
      { connectionId: 'c', afterConnectionIds: ['a', 'b'] },
    ])

    const revenue = w.get('[data-step="c"]')
    const boxes = revenue.findAll('input[type="checkbox"]')
    expect(boxes.map(b => b.attributes('data-after'))).toEqual(['a', 'b'])
    expect(boxes.every(b => (b.element as HTMLInputElement).checked)).toBe(true)
  })

  it('sets the steps a step runs after', async () => {
    const w = mountEditor([
      { connectionId: 'a', afterConnectionIds: [] },
      { connectionId: 'b', afterConnectionIds: [] },
    ])

    await w.get('[data-step="b"] input[data-after="a"]').setValue(true)

    expect(lastEmitted(w)).toEqual([
      { connectionId: 'a', afterConnectionIds: [] },
      { connectionId: 'b', afterConnectionIds: ['a'] },
    ])
  })

  it('disables an upstream choice that would make a cycle', () => {
    const w = mountEditor([
      { connectionId: 'a', afterConnectionIds: [] },
      { connectionId: 'b', afterConnectionIds: ['a'] },
    ])

    expect(w.get('[data-step="a"] input[data-after="b"]').attributes('disabled')).toBeDefined()
    expect(w.get('[data-step="b"] input[data-after="a"]').attributes('disabled')).toBeUndefined()
  })

  it('adds a step from the connections not yet in the workflow', async () => {
    const w = mountEditor([{ connectionId: 'a', afterConnectionIds: [] }])

    const options = w.get('select').findAll('option').map(o => o.attributes('value'))
    expect(options).toEqual(['', 'b', 'c'])

    await w.get('select').setValue('c')
    await w.findAll('button').find(b => b.text() === 'Add step')!.trigger('click')

    expect(lastEmitted(w)).toEqual([
      { connectionId: 'a', afterConnectionIds: [] },
      { connectionId: 'c', afterConnectionIds: [] },
    ])
  })

  it('removes a step and the edges to it', async () => {
    const w = mountEditor([
      { connectionId: 'a', afterConnectionIds: [] },
      { connectionId: 'b', afterConnectionIds: ['a'] },
    ])

    await w.get('button[aria-label="Remove step Orders"]').trigger('click')

    expect(lastEmitted(w)).toEqual([{ connectionId: 'b', afterConnectionIds: [] }])
    expect(removeStep([{ connectionId: 'a', afterConnectionIds: [] }], 'x')).toHaveLength(1)
  })

  it('shows a viewer the steps without controls', () => {
    const w = mountEditor([
      { connectionId: 'a', afterConnectionIds: [] },
      { connectionId: 'b', afterConnectionIds: ['a'] },
    ], true)

    expect(w.find('input').exists()).toBe(false)
    expect(w.find('select').exists()).toBe(false)
    expect(w.find('button').exists()).toBe(false)
    expect(w.get('[data-step="b"]').text()).toContain('Orders')
  })
})
