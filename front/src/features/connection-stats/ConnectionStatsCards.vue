<script setup lang="ts">
import type { ConnectionStats } from '@entities/stats'
import { formatDuration, formatNumber, formatPercent, formatRelativeTime, formatTrend } from '@shared/lib/format'

defineProps<{
  stats: ConnectionStats
}>()

function durationTrend(delta: number) {
  // For duration, lower is better -- invert color logic
  const t = formatTrend(delta)
  if (t.color === 'green')
    return { ...t, color: 'red' as const }
  if (t.color === 'red')
    return { ...t, color: 'green' as const }
  return t
}

const trendColor: Record<string, string> = {
  green: 'text-green-600',
  red: 'text-red-600',
  gray: 'text-text-muted',
}
</script>

<template>
  <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
    <!-- Avg Duration -->
    <div class="bg-surface border border-border rounded-xl p-5">
      <p class="text-xs text-text-secondary uppercase mb-1">
        Avg Duration
      </p>
      <p class="text-2xl font-semibold text-heading mt-1">
        {{ formatDuration(stats.avgDurationMs) }}
      </p>
      <p class="text-xs mt-1" :class="trendColor[durationTrend(stats.avgDurationDelta).color]">
        {{ durationTrend(stats.avgDurationDelta).text }}
      </p>
    </div>

    <!-- Success Rate -->
    <div class="bg-surface border border-border rounded-xl p-5">
      <p class="text-xs text-text-secondary uppercase mb-1">
        Success Rate
      </p>
      <p class="text-2xl font-semibold text-heading mt-1">
        {{ formatPercent(stats.successRate) }}
      </p>
      <p class="text-xs mt-1" :class="trendColor[formatTrend(stats.successRateDelta).color]">
        {{ formatTrend(stats.successRateDelta).text }}
      </p>
    </div>

    <!-- Records Synced -->
    <div class="bg-surface border border-border rounded-xl p-5">
      <p class="text-xs text-text-secondary uppercase mb-1">
        Records Synced
      </p>
      <p class="text-2xl font-semibold text-heading mt-1">
        {{ formatNumber(stats.totalRecords) }}
      </p>
      <p class="text-xs mt-1" :class="trendColor[formatTrend(stats.totalRecordsDelta).color]">
        {{ formatTrend(stats.totalRecordsDelta).text }}
      </p>
    </div>

    <!-- Last Sync -->
    <div class="bg-surface border border-border rounded-xl p-5">
      <p class="text-xs text-text-secondary uppercase mb-1">
        Last Sync
      </p>
      <p class="text-2xl font-semibold text-heading mt-1">
        {{ stats.lastSyncAt ? formatRelativeTime(stats.lastSyncAt) : '--' }}
      </p>
    </div>
  </div>
</template>
