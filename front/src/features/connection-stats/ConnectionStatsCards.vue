<script setup lang="ts">
import type { ConnectionStats } from '@entities/stats'
import { formatDuration, formatNumber, formatPercent, formatRelativeTime } from '@shared/lib/format'
import { SStatCard } from '@shared/ui'
import StatTrend from './StatTrend.vue'

defineProps<{
  stats: ConnectionStats
}>()
</script>

<template>
  <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
    <SStatCard label="Avg Duration" :value="formatDuration(stats.avgDurationMs)">
      <StatTrend :delta="stats.avgDurationDelta" lower-is-better />
    </SStatCard>

    <SStatCard label="Success Rate" :value="formatPercent(stats.successRate)">
      <StatTrend :delta="stats.successRateDelta" />
    </SStatCard>

    <SStatCard label="Records Synced" :value="formatNumber(stats.totalRecords)">
      <StatTrend :delta="stats.totalRecordsDelta" />
    </SStatCard>

    <SStatCard label="Last Sync" :value="stats.lastSyncAt ? formatRelativeTime(stats.lastSyncAt) : '--'" />
  </div>
</template>
