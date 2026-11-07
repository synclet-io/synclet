<script setup lang="ts">
import type { Destination } from '@entities/destination'
import { useManagedConnectors } from '@entities/connector'
import { useDestinations } from '@entities/destination'
import { groupOf } from '@features/executor-group'
import { SButton, SEmptyState, SSkeleton } from '@shared/ui'
import { computed } from 'vue'
import { formatConnector } from './connectorLabel'

const props = defineProps<{
  destinationId: string
  group?: string
}>()

const emit = defineEmits<{
  'update:destinationId': [value: string]
  'update:destination': [value: Destination]
  'auto-advance': []
}>()

const { data: destinations, isLoading } = useDestinations()
const offered = computed(() => (destinations.value?.items ?? []).filter(item => !props.group || groupOf(item) === props.group))
const { data: connectors } = useManagedConnectors()
const connectorsById = computed(() => new Map((connectors.value ?? []).map(c => [c.id, c])))

function selectDestination(d: Destination) {
  emit('update:destinationId', d.id)
  emit('update:destination', d)
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
      v-else-if="destinations && destinations.items.length > 0 && offered.length === 0"
      :title="`No destinations in executor group &quot;${group}&quot;`"
      description="A connection pairs a source and a destination of the same executor group."
    >
      <SButton to="/destinations/new">
        Add Destination
      </SButton>
    </SEmptyState>

    <SEmptyState
      v-else-if="destinations && destinations.items.length === 0"
      title="No destinations configured"
      description="Create a destination before setting up a connection."
    >
      <SButton to="/destinations/new">
        Add Destination
      </SButton>
    </SEmptyState>

    <!-- Destination cards grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <button
        v-for="d in offered"
        :key="d.id"
        class="text-left rounded-xl p-4 transition-all"
        :class="d.id === destinationId
          ? 'ring-2 ring-primary border-primary border'
          : 'border border-border hover:border-primary/50 hover:shadow-soft cursor-pointer'"
        @click="selectDestination(d)"
      >
        <div class="text-sm font-medium text-heading">
          {{ d.name }}
        </div>
        <div class="text-xs text-text-muted mt-1">
          {{ formatConnector(connectorsById.get(d.managedConnectorId)) }}
        </div>
      </button>
    </div>
  </div>
</template>
