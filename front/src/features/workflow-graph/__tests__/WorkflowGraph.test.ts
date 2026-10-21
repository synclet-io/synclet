import type { WorkflowStepInput } from '@entities/workflow'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import WorkflowGraph from '../WorkflowGraph.vue'

interface Point { x: number, y: number }
interface Box { x: number, y: number, width: number, height: number }

function mountGraph(steps: WorkflowStepInput[]) {
  return mount(WorkflowGraph, { props: { steps, connectionName: (id: string) => id.toUpperCase() } })
}

function box(w: ReturnType<typeof mountGraph>, connectionId: string): Box {
  const rect = w.get(`g[data-step="${connectionId}"] rect`)
  return {
    x: Number(rect.attributes('x')),
    y: Number(rect.attributes('y')),
    width: Number(rect.attributes('width')),
    height: Number(rect.attributes('height')),
  }
}

function centreY(b: Box): number {
  return b.y + b.height / 2
}

/** Samples an SVG path of M, L, Q and C commands with absolute coordinates into a polyline. */
function samplePath(d: string): Point[] {
  const tokens = d.match(/[MLQC]|-?\d+(?:\.\d+)?(?:e-?\d+)?/gi)!
  const points: Point[] = []
  let current: Point = { x: 0, y: 0 }
  let i = 0
  const num = () => Number(tokens[i++])
  const pt = () => ({ x: num(), y: num() })
  let command = ''
  while (i < tokens.length) {
    if (/[MLQC]/i.test(tokens[i]!))
      command = tokens[i++]!
    if (command === 'M' || command === 'L') {
      current = pt()
      points.push(current)
    }
    else if (command === 'Q') {
      const c = pt()
      const end = pt()
      for (let s = 1; s <= 20; s++) {
        const t = s / 20
        const u = 1 - t
        points.push({ x: u * u * current.x + 2 * u * t * c.x + t * t * end.x, y: u * u * current.y + 2 * u * t * c.y + t * t * end.y })
      }
      current = end
    }
    else if (command === 'C') {
      const c1 = pt()
      const c2 = pt()
      const end = pt()
      for (let s = 1; s <= 20; s++) {
        const t = s / 20
        const u = 1 - t
        points.push({
          x: u ** 3 * current.x + 3 * u * u * t * c1.x + 3 * u * t * t * c2.x + t ** 3 * end.x,
          y: u ** 3 * current.y + 3 * u * u * t * c1.y + 3 * u * t * t * c2.y + t ** 3 * end.y,
        })
      }
      current = end
    }
    else {
      throw new Error(`unsupported path command ${command}`)
    }
  }
  return points
}

function edgePolyline(w: ReturnType<typeof mountGraph>, from: string, to: string): Point[] {
  return samplePath(w.get(`path[data-edge="${from}-${to}"]`).attributes('d')!)
}

function segmentsCross(a: Point, b: Point, c: Point, d: Point): boolean {
  const cross = (p: Point, q: Point, r: Point) => (q.x - p.x) * (r.y - p.y) - (q.y - p.y) * (r.x - p.x)
  const d1 = cross(c, d, a)
  const d2 = cross(c, d, b)
  const d3 = cross(a, b, c)
  const d4 = cross(a, b, d)
  if (d1 === 0 && d2 === 0 && d3 === 0 && d4 === 0)
    return false
  return d1 * d2 <= 0 && d3 * d4 <= 0
}

function crossings(lines: Point[][]): number {
  let count = 0
  for (let i = 0; i < lines.length; i++) {
    for (let j = i + 1; j < lines.length; j++) {
      const a = lines[i]!
      const b = lines[j]!
      for (let s = 1; s < a.length; s++) {
        for (let t = 1; t < b.length; t++) {
          if (segmentsCross(a[s - 1]!, a[s]!, b[t - 1]!, b[t]!))
            count++
        }
      }
    }
  }
  return count
}

function inside(p: Point, b: Box): boolean {
  return p.x > b.x && p.x < b.x + b.width && p.y > b.y && p.y < b.y + b.height
}

describe('workflowGraph layout', () => {
  it('centres the first and last step of a diamond between its branches', () => {
    const w = mountGraph([
      { connectionId: 'a', afterConnectionIds: [] },
      { connectionId: 'b', afterConnectionIds: ['a'] },
      { connectionId: 'c', afterConnectionIds: ['a'] },
      { connectionId: 'd', afterConnectionIds: ['b', 'c'] },
    ])

    const middle = (centreY(box(w, 'b')) + centreY(box(w, 'c'))) / 2
    expect(centreY(box(w, 'b'))).not.toBe(centreY(box(w, 'c')))
    expect(centreY(box(w, 'a'))).toBeCloseTo(middle, 0)
    expect(centreY(box(w, 'd'))).toBeCloseTo(middle, 0)
    expect(box(w, 'a').x).toBeLessThan(box(w, 'b').x)
    expect(box(w, 'b').x).toBeLessThan(box(w, 'd').x)
  })

  it('lays out edges whose list order would cross without crossings', () => {
    const w = mountGraph([
      { connectionId: 'a1', afterConnectionIds: [] },
      { connectionId: 'a2', afterConnectionIds: [] },
      { connectionId: 'b1', afterConnectionIds: ['a2'] },
      { connectionId: 'b2', afterConnectionIds: ['a1'] },
    ])

    expect(crossings([edgePolyline(w, 'a1', 'b2'), edgePolyline(w, 'a2', 'b1')])).toBe(0)
  })

  it('routes a long edge around the step it skips', () => {
    const w = mountGraph([
      { connectionId: 'a', afterConnectionIds: [] },
      { connectionId: 'b', afterConnectionIds: ['a'] },
      { connectionId: 'c', afterConnectionIds: ['a', 'b'] },
    ])

    const b = box(w, 'b')
    expect(edgePolyline(w, 'a', 'c').some(p => inside(p, b))).toBe(false)
  })
})
