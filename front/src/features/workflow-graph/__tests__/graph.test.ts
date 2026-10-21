import type { WorkflowRun, WorkflowStepRunStatus } from '@entities/workflow'
import { describe, expect, it } from 'vitest'
import { edgePath, layoutWorkflowGraph, sortStepRunsByGraph, stepDisplayStatus, wouldCreateCycle } from '../graph'

const diamond = [
  { connectionId: 'a', afterConnectionIds: [] },
  { connectionId: 'b', afterConnectionIds: ['a'] },
  { connectionId: 'c', afterConnectionIds: ['a'] },
  { connectionId: 'd', afterConnectionIds: ['b', 'c'] },
]

function run(status: WorkflowRun['status'], steps: Record<string, WorkflowStepRunStatus>): WorkflowRun {
  return {
    id: 'r1',
    workflowId: 'w1',
    status,
    trigger: 'manual',
    startedAt: undefined,
    completedAt: undefined,
    stepRuns: Object.entries(steps).map(([connectionId, s]) => ({
      id: `sr-${connectionId}`,
      stepId: `s-${connectionId}`,
      connectionId,
      status: s,
      jobId: '',
      error: '',
      startedAt: undefined,
      completedAt: undefined,
    })),
  }
}

describe('layoutWorkflowGraph', () => {
  it('lays the steps out left to right after the steps they run after', () => {
    const layout = layoutWorkflowGraph([
      ...diamond,
      { connectionId: 'e', afterConnectionIds: ['a', 'd'] },
    ])

    const x = Object.fromEntries(layout.steps.map(s => [s.connectionId, s.x]))
    expect(x.a).toBeLessThan(x.b!)
    expect(x.b).toBe(x.c)
    expect(x.b).toBeLessThan(x.d!)
    expect(x.d).toBeLessThan(x.e!)
    expect(layout.edges.map(e => `${e.from}-${e.to}`).sort()).toEqual(['a-b', 'a-c', 'a-e', 'b-d', 'c-d', 'd-e'])
    expect(layout.edges.every(e => e.points.length >= 2)).toBe(true)
    for (const s of layout.steps) {
      expect(s.x + s.width).toBeLessThanOrEqual(layout.width)
      expect(s.y + s.height).toBeLessThanOrEqual(layout.height)
    }
  })

  it('leaves out edges to connections that are not steps', () => {
    const layout = layoutWorkflowGraph([{ connectionId: 'a', afterConnectionIds: ['gone'] }])
    expect(layout.edges).toEqual([])
    expect(layout.steps.map(s => s.connectionId)).toEqual(['a'])
  })

  it('lays out an empty workflow', () => {
    expect(layoutWorkflowGraph([])).toEqual({ steps: [], edges: [], width: 0, height: 0 })
  })
})

describe('edgePath', () => {
  it('draws a straight route as a line and bends as curves', () => {
    expect(edgePath([{ x: 0, y: 0 }, { x: 10, y: 0 }])).toBe('M 0,0 L 5,0 L 10,0')
    expect(edgePath([{ x: 0, y: 0 }, { x: 10, y: 0 }, { x: 10, y: 10 }])).toBe('M 0,0 L 5,0 Q 10,0 10,5 L 10,10')
  })
})

describe('wouldCreateCycle', () => {
  it('rejects running a step after itself or after a step downstream of it', () => {
    expect(wouldCreateCycle(diamond, 'a', 'a')).toBe(true)
    expect(wouldCreateCycle(diamond, 'a', 'd')).toBe(true)
    expect(wouldCreateCycle(diamond, 'b', 'd')).toBe(true)
  })

  it('allows an edge between independent steps', () => {
    expect(wouldCreateCycle(diamond, 'c', 'b')).toBe(false)
    expect(wouldCreateCycle(diamond, 'd', 'a')).toBe(false)
  })
})

describe('stepDisplayStatus', () => {
  it('shows the ready pending steps of a running run as starting and the others as waiting', () => {
    const r = run('running', { a: 'pending', b: 'pending', c: 'pending', d: 'pending' })
    expect(stepDisplayStatus(r, diamond[0]!)).toBe('starting')
    expect(stepDisplayStatus(r, diamond[1]!)).toBe('waiting')
  })

  it('shows a pending step whose upstream steps all finished as starting', () => {
    const r = run('running', { a: 'succeeded', b: 'succeeded', c: 'queued', d: 'pending' })
    expect(stepDisplayStatus(r, diamond[3]!)).toBe('waiting')
    const later = run('running', { a: 'succeeded', b: 'succeeded', c: 'succeeded', d: 'pending' })
    expect(stepDisplayStatus(later, diamond[3]!)).toBe('starting')
  })

  it('shows the stored status otherwise', () => {
    const r = run('failed', { a: 'failed', b: 'skipped', c: 'skipped', d: 'skipped' })
    expect(stepDisplayStatus(r, diamond[0]!)).toBe('failed')
    expect(stepDisplayStatus(r, diamond[3]!)).toBe('skipped')
  })

  it('shows a step without a step run as not run', () => {
    expect(stepDisplayStatus(undefined, diamond[0]!)).toBe('not run')
    expect(stepDisplayStatus(run('succeeded', { a: 'succeeded' }), diamond[1]!)).toBe('not run')
  })
})

describe('sortStepRunsByGraph', () => {
  it('orders step runs by graph layer, then by the step list, with removed steps last', () => {
    const stepRuns = ['removed', 'd', 'c', 'gone', 'a', 'b'].map(connectionId => ({ connectionId }))

    expect(sortStepRunsByGraph(stepRuns, diamond).map(s => s.connectionId)).toEqual(['a', 'b', 'c', 'd', 'removed', 'gone'])
  })

  it('orders steps of the same layer as in the step list', () => {
    const steps = [
      { connectionId: 'c', afterConnectionIds: ['a'] },
      { connectionId: 'b', afterConnectionIds: [] },
      { connectionId: 'a', afterConnectionIds: [] },
    ]
    const stepRuns = ['c', 'a', 'b'].map(connectionId => ({ connectionId }))

    expect(sortStepRunsByGraph(stepRuns, steps).map(s => s.connectionId)).toEqual(['b', 'a', 'c'])
  })
})
