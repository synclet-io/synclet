<script setup lang="ts">
import type { Source } from '@entities/source'
import { useSources } from '@entities/source'
import { SButton, SEmptyState, SSkeleton } from '@shared/ui'

defineProps<{
  sourceId: string
}>()

const emit = defineEmits<{
  'update:sourceId': [value: string]
  'update:source': [value: Source]
  'auto-advance': []
}>()

const { data: sources, isLoading } = useSources()

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
        v-for="s in sources?.items"
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
          {{ s.managedConnectorId }}
        </div>
      </button>
    </div>
  </div>
</template>
