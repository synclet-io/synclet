<script setup lang="ts">
import type { Source } from '@entities/source'
import { useManagedConnectors } from '@entities/connector'
import { useSources } from '@entities/source'
import { groupOf } from '@features/executor-group'
import { SButton, SEmptyState, SSkeleton } from '@shared/ui'
import { computed } from 'vue'
import { formatConnector } from './connectorLabel'

const props = defineProps<{
  sourceId: string
  group?: string
}>()

const emit = defineEmits<{
  'update:sourceId': [value: string]
  'update:source': [value: Source]
  'auto-advance': []
}>()

const { data: sources, isLoading } = useSources()
const offered = computed(() => (sources.value?.items ?? []).filter(item => !props.group || groupOf(item) === props.group))
const { data: connectors } = useManagedConnectors()
const connectorsById = computed(() => new Map((connectors.value ?? []).map(c => [c.id, c])))

function selectSource(s: Source) {
  emit('update:sourceId', s.id)
  emit('update:source', s)
  setTimeout(emit, 300, 'auto-advance')
}
</script>

<template>
  <div>
    <!-- Loading skeletons -->
    <div v-if="isLoading" class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <div v-for="i in 4" :key="i" class="rounded-xl border border-border p-4">
        <SSkeleton width="60%" height="16px" />
        <SSkeleton width="40%" height="12px" class="mt-2" />
      </div>
    </div>

    <!-- Empty state -->
    <SEmptyState
      v-else-if="sources && sources.items.length > 0 && offered.length === 0"
      :title="`No sources in executor group &quot;${group}&quot;`"
      description="A connection pairs a source and a destination of the same executor group."
    >
      <SButton to="/sources/new">
        Add Source
      </SButton>
    </SEmptyState>

    <SEmptyState
      v-else-if="sources && sources.items.length === 0"
      title="No sources configured"
      description="Create a source before setting up a connection."
    >
      <SButton to="/sources/new">
        Add Source
      </SButton>
    </SEmptyState>

    <!-- Source cards grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <button
        v-for="s in offered"
        :key="s.id"
        class="text-left rounded-xl p-4 transition-all"
        :class="s.id === sourceId
          ? 'ring-2 ring-primary border-primary border'
          : 'border border-border hover:border-primary/50 hover:shadow-soft cursor-pointer'"
        @click="selectSource(s)"
      >
        <div class="text-sm font-medium text-heading">
          {{ s.name }}
        </div>
        <div class="text-xs text-text-muted mt-1">
          {{ formatConnector(connectorsById.get(s.managedConnectorId)) }}
        </div>
      </button>
    </div>
  </div>
</template>
