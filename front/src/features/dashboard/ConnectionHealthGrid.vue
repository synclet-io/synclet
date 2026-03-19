<script setup lang="ts">
import type { ConnectionHealth, HealthStatus } from '@entities/stats'
import { formatRelativeTime } from '@shared/lib/format'
import { RouterLink } from 'vue-router'

defineProps<{
  connections: ConnectionHealth[]
}>()

const tileColor: Record<HealthStatus, string> = {
  healthy: 'bg-green-500',
  warning: 'bg-amber-500',
  failing: 'bg-red-500',
  disabled: 'bg-slate-300',
}

const textColor: Record<HealthStatus, string> = {
  healthy: 'text-white',
  warning: 'text-white',
  failing: 'text-white',
  disabled: 'text-text-secondary',
}

function tileTooltip(conn: ConnectionHealth): string {
  const lastSync = conn.lastSyncAt ? formatRelativeTime(conn.lastSyncAt) : 'Never'
  return `${conn.connectionName} -- Last sync: ${lastSync}, Status: ${conn.health}`
}
</script>

<template>
  <div class="flex flex-wrap gap-2">
    <RouterLink
      v-for="conn in connections"
      :key="conn.connectionId"
      :to="`/connections/${conn.connectionId}`"
      class="w-16 h-16 rounded-lg flex items-center justify-center cursor-pointer transition-opacity hover:opacity-80"
      :class="[tileColor[conn.health], textColor[conn.health]]"
      :title="tileTooltip(conn)"
    >
      <span class="text-xs font-medium truncate px-1">{{ conn.connectionName }}</span>
    </RouterLink>
  </div>
</template>
