import type { WorkflowRun, WorkflowStepInput, WorkflowStepRunStatus } from '@entities/workflow'
import dagre from '@dagrejs/dagre'

export interface Point {
  x: number
  y: number
}

/** A step's box; x and y are its top-left corner. */
export interface PlacedStep {
  connectionId: string
  x: number
  y: number
  width: number
  height: number
}

export interface GraphEdge {
  from: string
  to: string
  /** Route from the upstream box to the downstream box, around the boxes between them. */
  points: Point[]
}

export interface WorkflowGraphLayout {
  steps: PlacedStep[]
  edges: GraphEdge[]
  width: number
  height: number
}

export interface NodeSize {
  width: number
  height: number
}

/**
 * Lays the steps out left to right in layers (dagre's Sugiyama layout): each
 * step comes after the steps it runs after, the order within a layer keeps
 * edges from crossing, and a step sits centred against its neighbours. Edges
 * to connections that are not steps are left out.
 */
export function layoutWorkflowGraph(steps: WorkflowStepInput[], size: NodeSize = { width: 200, height: 56 }): WorkflowGraphLayout {
  if (steps.length === 0)
    return { steps: [], edges: [], width: 0, height: 0 }

  const graph = new dagre.graphlib.Graph()
  graph.setGraph({ rankdir: 'LR', nodesep: 24, ranksep: 64, edgesep: 16, marginx: 2, marginy: 2 })
  graph.setDefaultEdgeLabel(() => ({}))

  const ids = new Set(steps.map(s => s.connectionId))
  for (const step of steps)
    graph.setNode(step.connectionId, { width: size.width, height: size.height })

  const edges: { from: string, to: string }[] = []
  for (const step of steps) {
    for (const upstream of step.afterConnectionIds) {
      if (ids.has(upstream) && upstream !== step.connectionId) {
        graph.setEdge(upstream, step.connectionId)
        edges.push({ from: upstream, to: step.connectionId })
      }
    }
  }

  dagre.layout(graph)

  const placed = steps.map((step) => {
    const node = graph.node(step.connectionId)
    return {
      connectionId: step.connectionId,
      x: node.x - size.width / 2,
      y: node.y - size.height / 2,
      width: size.width,
      height: size.height,
    }
  })

  const label = graph.graph()
  return {
    steps: placed,
    edges: edges.map(e => ({ ...e, points: (graph.edge(e.from, e.to)?.points ?? []).map((p: Point) => ({ x: p.x, y: p.y })) })),
    width: label.width ?? 0,
    height: label.height ?? 0,
  }
}

/**
 * The step runs in the order the graph shows their steps: by layer from left
 * to right, then by position in the step list. Step runs whose step is not in
 * steps come last, in their given order.
 */
export function sortStepRunsByGraph<T extends { connectionId: string }>(stepRuns: T[], steps: WorkflowStepInput[]): T[] {
  const placed = new Map(layoutWorkflowGraph(steps).steps.map((s, index) => [s.connectionId, { x: s.x, index }]))
  const known = stepRuns.filter(s => placed.has(s.connectionId))
  const removed = stepRuns.filter(s => !placed.has(s.connectionId))
  known.sort((a, b) => {
    const pa = placed.get(a.connectionId)!
    const pb = placed.get(b.connectionId)!
    return pa.x - pb.x || pa.index - pb.index
  })
  return [...known, ...removed]
}

/**
 * A smooth SVG path along a route: straight to the middle of the first leg,
 * quadratic curves around each bend, straight into the last point.
 */
export function edgePath(points: Point[]): string {
  if (points.length === 0)
    return ''
  const [first] = points
  if (points.length === 1)
    return `M ${first!.x},${first!.y}`
  const mid = (a: Point, b: Point) => ({ x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 })
  let d = `M ${first!.x},${first!.y}`
  const start = mid(points[0]!, points[1]!)
  d += ` L ${start.x},${start.y}`
  for (let i = 1; i < points.length - 1; i++) {
    const bend = points[i]!
    const next = mid(bend, points[i + 1]!)
    d += ` Q ${bend.x},${bend.y} ${next.x},${next.y}`
  }
  const last = points.at(-1)!
  d += ` L ${last.x},${last.y}`
  return d
}

/** True when the step of connectionId would run after candidate and candidate already runs after it. */
export function wouldCreateCycle(steps: WorkflowStepInput[], connectionId: string, candidate: string): boolean {
  if (connectionId === candidate)
    return true
  const byConnection = new Map(steps.map(s => [s.connectionId, s]))
  const seen = new Set<string>()
  const stack = [candidate]
  while (stack.length > 0) {
    const current = stack.pop()!
    if (current === connectionId)
      return true
    if (seen.has(current))
      continue
    seen.add(current)
    stack.push(...(byConnection.get(current)?.afterConnectionIds ?? []))
  }
  return false
}

export type StepDisplayStatus = WorkflowStepRunStatus | 'starting' | 'waiting' | 'not run'

/**
 * The status a step shows for a run. A pending step of a running run whose
 * upstream steps all succeeded is "starting": the run's next jobs are created
 * asynchronously, a moment after it starts or a step finishes. A pending step
 * still waiting for an upstream step is "waiting".
 */
export function stepDisplayStatus(run: WorkflowRun | undefined, step: WorkflowStepInput): StepDisplayStatus {
  const stepRun = run?.stepRuns.find(s => s.connectionId === step.connectionId)
  if (!run || !stepRun)
    return 'not run'
  if (stepRun.status !== 'pending' || run.status !== 'running')
    return stepRun.status
  const upstreamDone = step.afterConnectionIds.every((upstream) => {
    const status = run.stepRuns.find(s => s.connectionId === upstream)?.status
    return status !== 'pending' && status !== 'queued'
  })
  return upstreamDone ? 'starting' : 'waiting'
}

export function stepStatusVariant(status: StepDisplayStatus): 'success' | 'danger' | 'warning' | 'info' | 'gray' {
  switch (status) {
    case 'succeeded': return 'success'
    case 'failed': return 'danger'
    case 'queued': return 'info'
    case 'starting': return 'warning'
    default: return 'gray'
  }
}

export function runStatusVariant(status: WorkflowRun['status']): 'success' | 'danger' | 'info' | 'gray' {
  switch (status) {
    case 'succeeded': return 'success'
    case 'failed': return 'danger'
    case 'running': return 'info'
    default: return 'gray'
  }
}

export function stepNodeColor(status: StepDisplayStatus): { fill: string, stroke: string, text: string } {
  switch (status) {
    case 'succeeded':
      return { fill: 'fill-green-500/15', stroke: 'stroke-green-500', text: 'text-green-700 dark:text-green-200' }
    case 'failed':
      return { fill: 'fill-red-500/15', stroke: 'stroke-red-500', text: 'text-red-700 dark:text-red-200' }
    case 'queued':
      return { fill: 'fill-blue-500/15', stroke: 'stroke-blue-500', text: 'text-blue-700 dark:text-blue-200' }
    case 'starting':
      return { fill: 'fill-amber-500/15', stroke: 'stroke-amber-500', text: 'text-amber-700 dark:text-amber-200' }
    case 'skipped':
      return { fill: 'fill-slate-400/15', stroke: 'stroke-slate-400', text: 'text-slate-700 dark:text-slate-200' }
    default:
      return { fill: 'fill-slate-300/15', stroke: 'stroke-slate-300', text: 'text-slate-600 dark:text-slate-300' }
  }
}
