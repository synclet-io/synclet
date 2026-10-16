<script setup lang="ts">
import { useConnections } from '@entities/connection'
import { statusVariant } from '@shared/lib/format'
import { SBadge, SButton, SCard, SEmptyState, SSkeleton } from '@shared/ui'
import { ArrowRightLeft, Plus } from 'lucide-vue-next'
import { computed } from 'vue'
import { RouterLink } from 'vue-router'

const props = defineProps<{
  kind: 'source' | 'destination'
  instanceId: string
}>()

const { data, isLoading } = useConnections()

const connections = computed(() => (data.value?.items ?? []).filter(c =>
  props.kind === 'source' ? c.sourceId === props.instanceId : c.destinationId === props.instanceId,
))
</script>

<template>
  <SCard title="Used by" :description="`Connections that read from or write to this ${kind}`" class="mb-6">
    <template v-if="connections.length > 0" #header>
      <SButton variant="secondary" size="sm" :to="{ name: 'connection-new' }">
        <Plus class="w-4 h-4" /> New connection
      </SButton>
    </template>
    <SSkeleton v-if="isLoading" :count="3" />
    <SEmptyState
      v-else-if="connections.length === 0"
      :icon="ArrowRightLeft"
      title="No connections yet"
      :description="`Create a connection with this ${kind} to start syncing data.`"
    >
      <SButton size="sm" :to="{ name: 'connection-new' }">
        <Plus class="w-4 h-4" /> Create a connection
      </SButton>
    </SEmptyState>
    <ul v-else class="divide-y divide-border">
      <li v-for="connection in connections" :key="connection.id" class="flex items-center justify-between gap-3 py-2.5">
        <RouterLink
          :to="{ name: 'connection-detail', params: { id: connection.id } }"
          class="text-sm font-medium text-primary hover:underline hover:text-primary-hover truncate"
        >
          {{ connection.name }}
        </RouterLink>
        <SBadge :variant="statusVariant(connection.status)" dot>
          {{ connection.status }}
        </SBadge>
      </li>
    </ul>
  </SCard>
</template>
