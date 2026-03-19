<script setup lang="ts">
import type { TopConnection } from '@entities/stats'
import { formatNumber, formatRelativeTime } from '@shared/lib/format'
import SparklineChart from './SparklineChart.vue'

defineProps<{
  connections: TopConnection[]
}>()
</script>

<template>
  <div class="overflow-x-auto">
    <table class="w-full min-w-[600px]">
      <thead>
        <tr class="border-b border-border">
          <th class="px-6 py-3 text-xs font-medium text-text-muted uppercase tracking-wider bg-surface-raised text-left">
            Name
          </th>
          <th class="px-6 py-3 text-xs font-medium text-text-muted uppercase tracking-wider bg-surface-raised text-right">
            Records
          </th>
          <th class="px-6 py-3 text-xs font-medium text-text-muted uppercase tracking-wider bg-surface-raised text-right">
            Bytes
          </th>
          <th class="px-6 py-3 text-xs font-medium text-text-muted uppercase tracking-wider bg-surface-raised text-right">
            Last Sync
          </th>
          <th class="px-6 py-3 text-xs font-medium text-text-muted uppercase tracking-wider bg-surface-raised text-right">
            Trend
          </th>
        </tr>
      </thead>
      <tbody class="divide-y divide-border">
        <tr v-for="conn in connections" :key="conn.connectionId" class="hover:bg-surface-hover/50 transition-colors">
          <td class="px-6 py-3.5 text-sm">
            <RouterLink :to="`/connections/${conn.connectionId}`" class="text-primary hover:text-primary-hover font-medium">
              {{ conn.connectionName }}
            </RouterLink>
          </td>
          <td class="px-6 py-3.5 text-sm text-right text-text-secondary">
            {{ formatNumber(conn.recordsSynced) }}
          </td>
          <td class="px-6 py-3.5 text-sm text-right text-text-secondary">
            {{ formatNumber(conn.bytesSynced) }}
          </td>
          <td class="px-6 py-3.5 text-sm text-right text-text-secondary">
            {{ conn.lastSyncAt ? formatRelativeTime(conn.lastSyncAt) : '-' }}
          </td>
          <td class="px-6 py-3.5 text-right">
            <div class="flex justify-end">
              <SparklineChart v-if="conn.sparklineValues.length > 0" :values="conn.sparklineValues" />
              <span v-else class="text-sm text-text-muted">-</span>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
