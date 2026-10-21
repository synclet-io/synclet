<script setup lang="ts">
import type { WorkflowRun, WorkflowStepInput } from '@entities/workflow'
import type { StepDisplayStatus } from './graph'
import { computed } from 'vue'
import { edgePath, layoutWorkflowGraph, stepDisplayStatus, stepNodeColor } from './graph'

const props = defineProps<{
  steps: WorkflowStepInput[]
  run?: WorkflowRun
  connectionName: (connectionId: string) => string
}>()

const emit = defineEmits<{
  openJob: [jobId: string]
}>()

const NODE_WIDTH = 200
const NODE_HEIGHT = 56

const layout = computed(() => layoutWorkflowGraph(props.steps, { width: NODE_WIDTH, height: NODE_HEIGHT }))

interface Node {
  connectionId: string
  name: string
  status: StepDisplayStatus
  jobId: string
  error: string
  x: number
  y: number
}

const nodes = computed<Node[]>(() => {
  const byConnection = new Map(props.steps.map(s => [s.connectionId, s]))
  return layout.value.steps.map((p) => {
    const stepRun = props.run?.stepRuns.find(s => s.connectionId === p.connectionId)
    return {
      connectionId: p.connectionId,
      name: props.connectionName(p.connectionId),
      status: stepDisplayStatus(props.run, byConnection.get(p.connectionId)!),
      jobId: stepRun?.jobId ?? '',
      error: stepRun?.error ?? '',
      x: p.x,
      y: p.y,
    }
  })
})

const edges = computed(() => layout.value.edges.map(e => ({ key: `${e.from}-${e.to}`, d: edgePath(e.points) })))

const width = computed(() => Math.max(1, layout.value.width))
const height = computed(() => Math.max(1, layout.value.height))

function hint(node: Node): string {
  const label = `${node.name}: ${node.status}`
  return node.status === 'failed' && node.error ? `${label}\n${node.error}` : label
}

function openJob(node: Node) {
  if (node.jobId)
    emit('openJob', node.jobId)
}

function truncate(name: string): string {
  return name.length > 26 ? `${name.slice(0, 25)}…` : name
}
</script>

<template>
  <div class="overflow-auto">
    <svg
      :width="width"
      :height="height"
      :viewBox="`0 0 ${width} ${height}`"
      class="block overflow-visible"
      role="img"
      aria-label="Workflow steps graph"
    >
      <defs>
        <marker id="workflow-arrow" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="8" markerHeight="8" orient="auto-start-reverse">
          <path d="M 0 0 L 10 5 L 0 10 z" class="fill-current text-border-strong" />
        </marker>
      </defs>
      <g>
        <path
          v-for="edge in edges"
          :key="edge.key"
          :data-edge="edge.key"
          :d="edge.d"
          fill="none"
          class="stroke-border-strong"
          stroke-width="1.5"
          marker-end="url(#workflow-arrow)"
        />
      </g>
      <g
        v-for="node in nodes"
        :key="node.connectionId"
        :data-step="node.connectionId"
        :data-status="node.status"
        :role="node.jobId ? 'link' : undefined"
        :tabindex="node.jobId ? 0 : undefined"
        :aria-label="node.jobId ? `Open the job of ${node.name}` : undefined"
        :class="{ 'cursor-pointer focus:outline-none focus-visible:[&>rect]:stroke-primary': node.jobId }"
        @click="openJob(node)"
        @keydown.enter="openJob(node)"
      >
        <title>{{ hint(node) }}</title>
        <rect
          :x="node.x"
          :y="node.y"
          :width="NODE_WIDTH"
          :height="NODE_HEIGHT"
          rx="8"
          :class="[stepNodeColor(node.status).fill, stepNodeColor(node.status).stroke]"
          stroke-width="1.5"
        />
        <text
          :x="node.x + 12"
          :y="node.y + 22"
          class="text-xs font-medium fill-current text-heading"
        >
          {{ truncate(node.name) }}
        </text>
        <text
          :x="node.x + 12"
          :y="node.y + 40"
          class="text-[11px] fill-current"
          :class="stepNodeColor(node.status).text"
        >
          {{ node.status }}
        </text>
      </g>
    </svg>
  </div>
</template>
