<script setup lang="ts">
import type { TopologyFilters, TopologyNode } from '@features/topology/types'
import { nodeHealthColor } from '@features/topology/buildTopology'
import { useTopology } from '@features/topology/useTopology'
import { formatRelativeTime } from '@shared/lib/format'
import { PageHeader, SBadge, SButton, SEmptyState, SSkeleton } from '@shared/ui'
import { ArrowRightLeft, Database, Server } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const filters = ref<TopologyFilters>({ failingOnly: false, enabledOnly: false })

const { data, isLoading } = useTopology(filters)

const NODE_HEIGHT = 56
const NODE_VGAP = 12
const NODE_WIDTH = 200
const COLUMN_GAP = 80

const columnPositions = computed(() => ({
  source: 0,
  connection: NODE_WIDTH + COLUMN_GAP,
  destination: (NODE_WIDTH + COLUMN_GAP) * 2,
}))

interface PlacedNode {
  node: TopologyNode
  x: number
  y: number
}

function layoutColumn(nodes: TopologyNode[], x: number): PlacedNode[] {
  return nodes.map((node, i) => ({
    node,
    x,
    y: i * (NODE_HEIGHT + NODE_VGAP),
  }))
}

const placedSources = computed(() => layoutColumn(data.value.sources, columnPositions.value.source))
const placedConnections = computed(() => layoutColumn(data.value.connections, columnPositions.value.connection))
const placedDestinations = computed(() => layoutColumn(data.value.destinations, columnPositions.value.destination))

const positionById = computed(() => {
  const map = new Map<string, PlacedNode>()
  for (const p of placedSources.value) map.set(p.node.id, p)
  for (const p of placedConnections.value) map.set(p.node.id, p)
  for (const p of placedDestinations.value) map.set(p.node.id, p)
  return map
})

const svgHeight = computed(() => {
  const counts = [data.value.sources.length, data.value.connections.length, data.value.destinations.length]
  const max = Math.max(...counts, 1)
  return max * (NODE_HEIGHT + NODE_VGAP)
})

const svgWidth = computed(() => columnPositions.value.destination + NODE_WIDTH)

interface RenderedEdge {
  d: string
  failing: boolean
}

const renderedEdges = computed<RenderedEdge[]>(() => {
  const result: RenderedEdge[] = []
  for (const e of data.value.edges) {
    const from = positionById.value.get(e.from)
    const to = positionById.value.get(e.to)
    if (!from || !to)
      continue
    const x1 = from.x + NODE_WIDTH
    const y1 = from.y + NODE_HEIGHT / 2
    const x2 = to.x
    const y2 = to.y + NODE_HEIGHT / 2
    const cx = (x1 + x2) / 2
    const d = `M ${x1},${y1} C ${cx},${y1} ${cx},${y2} ${x2},${y2}`
    const conn = data.value.connections.find(c => c.id === e.connectionId)
    const failing = conn?.health === 'failing' || conn?.health === 'warning'
    result.push({ d, failing })
  }
  return result
})

const hovered = ref<PlacedNode | null>(null)
const tooltipPos = ref({ x: 0, y: 0 })

function onHover(p: PlacedNode, e: MouseEvent) {
  hovered.value = p
  tooltipPos.value = { x: e.clientX, y: e.clientY }
}

function onLeave() {
  hovered.value = null
}

function navigateTo(node: TopologyNode) {
  if (node.kind === 'source')
    router.push(`/sources/${node.id}`)
  else if (node.kind === 'destination')
    router.push(`/destinations/${node.id}`)
  else
    router.push(`/connections/${node.id}`)
}

function clearFilters() {
  filters.value = { failingOnly: false, enabledOnly: false }
}

const isEmpty = computed(() => data.value.sources.length === 0 && data.value.destinations.length === 0 && data.value.connections.length === 0)
</script>

<template>
  <PageHeader title="Topology" description="Source → connection → destination data flow, coloured by health">
    <template #actions>
      <SButton
        :variant="filters.failingOnly ? 'primary' : 'secondary'"
        size="sm"
        @click="filters = { ...filters, failingOnly: !filters.failingOnly }"
      >
        Failing only
      </SButton>
      <SButton
        :variant="filters.enabledOnly ? 'primary' : 'secondary'"
        size="sm"
        @click="filters = { ...filters, enabledOnly: !filters.enabledOnly }"
      >
        Enabled only
      </SButton>
      <SButton v-if="filters.failingOnly || filters.enabledOnly || filters.sourceId || filters.destinationId" variant="ghost" size="sm" @click="clearFilters">
        Clear
      </SButton>
    </template>
  </PageHeader>

  <div v-if="isLoading" class="space-y-3">
    <SSkeleton variant="rect" height="200px" />
  </div>

  <div v-else-if="isEmpty">
    <SEmptyState
      title="No data flows yet"
      description="Create a source, a destination, and a connection to see the topology."
    >
      <div class="flex gap-2 justify-center">
        <SButton to="/sources/new" size="sm">
          New source
        </SButton>
        <SButton to="/connections/new" size="sm">
          New connection
        </SButton>
      </div>
    </SEmptyState>
  </div>

  <div v-else class="overflow-auto bg-surface border border-border rounded-xl p-6">
    <!-- Column headers -->
    <div class="flex mb-3 text-xs font-semibold uppercase tracking-wider text-text-muted" :style="{ width: `${svgWidth}px` }">
      <div class="flex items-center gap-2" :style="{ width: `${NODE_WIDTH}px` }">
        <Database class="w-3.5 h-3.5" /> Sources
      </div>
      <div :style="{ width: `${COLUMN_GAP}px` }" />
      <div class="flex items-center gap-2" :style="{ width: `${NODE_WIDTH}px` }">
        <ArrowRightLeft class="w-3.5 h-3.5" /> Connections
      </div>
      <div :style="{ width: `${COLUMN_GAP}px` }" />
      <div class="flex items-center gap-2" :style="{ width: `${NODE_WIDTH}px` }">
        <Server class="w-3.5 h-3.5" /> Destinations
      </div>
    </div>

    <svg
      :width="svgWidth"
      :height="svgHeight"
      :viewBox="`0 0 ${svgWidth} ${svgHeight}`"
      class="block"
    >
      <!-- Edges -->
      <g>
        <path
          v-for="(edge, i) in renderedEdges"
          :key="i"
          :d="edge.d"
          fill="none"
          :class="edge.failing ? 'stroke-red-400' : 'stroke-slate-300 dark:stroke-slate-600'"
          stroke-width="1.5"
        />
      </g>

      <!-- Nodes -->
      <g
        v-for="placed in [...placedSources, ...placedConnections, ...placedDestinations]"
        :key="`${placed.node.kind}-${placed.node.id}`"
        class="cursor-pointer"
        @mousemove="onHover(placed, $event)"
        @mouseleave="onLeave"
        @click="navigateTo(placed.node)"
      >
        <rect
          :x="placed.x"
          :y="placed.y"
          :width="NODE_WIDTH"
          :height="NODE_HEIGHT"
          rx="8"
          :class="[nodeHealthColor(placed.node.health).fill, nodeHealthColor(placed.node.health).stroke]"
          stroke-width="1.5"
        />
        <text
          :x="placed.x + 12"
          :y="placed.y + 22"
          class="text-[12px] font-medium fill-current text-heading"
        >
          {{ placed.node.name.length > 24 ? `${placed.node.name.slice(0, 24)}…` : placed.node.name }}
        </text>
        <text
          :x="placed.x + 12"
          :y="placed.y + 40"
          class="text-[10px] uppercase tracking-wider"
          :class="nodeHealthColor(placed.node.health).text"
        >
          {{ placed.node.health }}
        </text>
      </g>
    </svg>

    <!-- Hover tooltip -->
    <div
      v-if="hovered"
      class="fixed z-40 pointer-events-none bg-surface-raised border border-border rounded-lg shadow-overlay p-3 text-xs"
      :style="{ left: `${tooltipPos.x + 12}px`, top: `${tooltipPos.y + 12}px` }"
    >
      <p class="font-semibold text-heading mb-1">
        {{ hovered.node.name }}
      </p>
      <div class="flex items-center gap-2 mb-1">
        <span class="text-text-muted">Type:</span>
        <SBadge variant="gray">
          {{ hovered.node.kind }}
        </SBadge>
      </div>
      <div class="flex items-center gap-2 mb-1">
        <span class="text-text-muted">Health:</span>
        <span class="font-medium">{{ hovered.node.health }}</span>
      </div>
      <div v-if="hovered.node.lastSyncAt" class="text-text-muted">
        Last sync: {{ formatRelativeTime(hovered.node.lastSyncAt) }}
      </div>
    </div>
  </div>
</template>
