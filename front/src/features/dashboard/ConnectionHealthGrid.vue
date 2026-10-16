<script setup lang="ts">
import type { ConnectionHealth, HealthStatus } from '@entities/stats'
import { formatRelativeTime } from '@shared/lib/format'
import { RouterLink } from 'vue-router'

defineProps<{
  connections: ConnectionHealth[]
}>()

const tileStyle: Record<HealthStatus, { tile: string, dot: string, label: string }> = {
  healthy: { tile: 'bg-success-bg border-success-border', dot: 'bg-success', label: 'Healthy' },
  warning: { tile: 'bg-warning-bg border-warning-border', dot: 'bg-warning', label: 'Warning' },
  failing: { tile: 'bg-danger-bg border-danger-border', dot: 'bg-danger', label: 'Failing' },
  disabled: { tile: 'bg-surface-raised border-border', dot: 'bg-text-disabled', label: 'Disabled' },
}

function lastSync(conn: ConnectionHealth): string {
  return conn.lastSyncAt ? formatRelativeTime(conn.lastSyncAt) : 'Never'
}
</script>

<template>
  <ul class="flex flex-wrap gap-2">
    <li v-for="conn in connections" :key="conn.connectionId">
      <RouterLink
        :to="`/connections/${conn.connectionId}`"
        class="flex flex-col justify-between w-28 h-16 px-2 py-1.5 rounded-lg border transition-colors hover:border-border-strong focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        :class="tileStyle[conn.health].tile"
        :title="`${conn.connectionName}: ${tileStyle[conn.health].label}, last sync ${lastSync(conn)}`"
      >
        <span class="text-xs font-medium text-heading truncate">{{ conn.connectionName }}</span>
        <span class="flex items-center gap-1.5 text-xs text-text-secondary">
          <span class="w-1.5 h-1.5 rounded-full shrink-0" :class="tileStyle[conn.health].dot" aria-hidden="true" />
          {{ tileStyle[conn.health].label }}
          <span class="sr-only">, last sync {{ lastSync(conn) }}</span>
        </span>
      </RouterLink>
    </li>
  </ul>
</template>
