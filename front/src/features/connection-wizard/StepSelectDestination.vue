<script setup lang="ts">
import type { Destination } from '@entities/destination'
import { useDestinations } from '@entities/destination'
import { SButton, SEmptyState, SSkeleton } from '@shared/ui'

defineProps<{
  destinationId: string
}>()

const emit = defineEmits<{
  'update:destinationId': [value: string]
  'update:destination': [value: Destination]
  'auto-advance': []
}>()

const { data: destinations, isLoading } = useDestinations()

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
        v-for="d in destinations?.items"
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
          {{ d.managedConnectorId }}
        </div>
      </button>
    </div>
  </div>
</template>
