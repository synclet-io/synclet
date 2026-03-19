<script setup lang="ts">
import type { TimeRange } from '@entities/stats'
import { useSyncTimeline, useWorkspaceStats } from '@entities/stats'
import ConnectionHealthGrid from '@features/dashboard/ConnectionHealthGrid.vue'
import DurationTrendsChart from '@features/dashboard/DurationTrendsChart.vue'
import FailureBreakdownChart from '@features/dashboard/FailureBreakdownChart.vue'
import SyncTimelineChart from '@features/dashboard/SyncTimelineChart.vue'
import ThroughputChart from '@features/dashboard/ThroughputChart.vue'
import TimeRangeSelector from '@features/dashboard/TimeRangeSelector.vue'
import TopConnectionsTable from '@features/dashboard/TopConnectionsTable.vue'
import { formatNumber, formatPercent, formatTrend } from '@shared/lib/format'
import { PageHeader, SCard, SEmptyState, SSkeleton, SStatCard } from '@shared/ui'
import { Activity, ArrowRightLeft, CheckCircle, Database, XCircle } from 'lucide-vue-next'
import { ref } from 'vue'

const timeRange = ref<TimeRange>('24h')
const { data: stats, isLoading, isError: statsError } = useWorkspaceStats(timeRange)
const { data: timeline, isLoading: timelineLoading, isError: timelineError } = useSyncTimeline(timeRange)

function trendColorClass(color: 'green' | 'red' | 'gray') {
  if (color === 'green')
    return 'text-green-600'
  if (color === 'red')
    return 'text-red-600'
  return 'text-text-muted'
}
</script>

<template>
  <div>
    <PageHeader title="Dashboard" description="Overview of your data synchronization platform">
      <template #actions>
        <TimeRangeSelector v-model="timeRange" />
      </template>
    </PageHeader>

    <!-- Loading skeleton for stat cards -->
    <div v-if="isLoading" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4 mb-8">
      <SSkeleton v-for="i in 5" :key="i" variant="rect" height="90px" />
    </div>

    <!-- Error state -->
    <div v-else-if="statsError || timelineError" class="rounded-lg border border-red-200 bg-red-50 p-6 mb-8">
      <div class="flex items-center gap-3">
        <XCircle class="h-5 w-5 text-red-500 shrink-0" />
        <div>
          <p class="font-medium text-red-800">Failed to load dashboard data</p>
          <p class="text-sm text-red-600 mt-1">Please try refreshing the page. If the problem persists, check your connection.</p>
        </div>
      </div>
    </div>

    <!-- Empty state when no stats -->
    <SEmptyState
      v-else-if="!stats"
      title="No sync data yet"
      description="Stats will appear here after your first sync completes. Create a connection to get started."
      class="mb-8"
    />

    <!-- Dashboard content -->
    <div v-else>
      <!-- 5 Stat Cards Row -->
      <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4 mb-8">
        <SStatCard label="Total Syncs (24h)" :value="stats.totalSyncs" :icon="Activity" color="blue">
          <template #default>
            <p class="text-xs mt-1" :class="trendColorClass(formatTrend(stats.totalSyncsDelta).color)">
              {{ formatTrend(stats.totalSyncsDelta).text }}
            </p>
          </template>
        </SStatCard>

        <SStatCard label="Success Rate" :value="formatPercent(stats.successRate)" :icon="CheckCircle" color="green">
          <template #default>
            <p class="text-xs mt-1" :class="trendColorClass(formatTrend(stats.successRateDelta).color)">
              {{ formatTrend(stats.successRateDelta).text }}
            </p>
          </template>
        </SStatCard>

        <SStatCard label="Records Synced (24h)" :value="formatNumber(stats.recordsSynced)" :icon="Database" color="purple">
          <template #default>
            <p class="text-xs mt-1" :class="trendColorClass(formatTrend(stats.recordsSyncedDelta).color)">
              {{ formatTrend(stats.recordsSyncedDelta).text }}
            </p>
          </template>
        </SStatCard>

        <SStatCard label="Active Connections" :value="stats.activeConnections" :icon="ArrowRightLeft" color="blue" />

        <SStatCard
          label="Failed Syncs (24h)"
          :value="stats.failedSyncs"
          :icon="XCircle"
          color="amber"
        >
          <template #default>
            <p class="text-xs mt-1" :class="trendColorClass(formatTrend(stats.failedSyncsDelta).color)">
              {{ formatTrend(stats.failedSyncsDelta).text }}
            </p>
          </template>
        </SStatCard>
      </div>

      <!-- Chart Grid (2x2) -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <SCard title="Sync Timeline">
          <SSkeleton v-if="timelineLoading" variant="rect" height="256px" />
          <SEmptyState v-else-if="!timeline || timeline.points.length === 0" title="No timeline data" description="Timeline will appear after syncs complete." />
          <SyncTimelineChart
            v-else
            :labels="timeline.points.map(p => p.label)"
            :succeeded="timeline.points.map(p => p.succeeded)"
            :failed="timeline.points.map(p => p.failed)"
          />
        </SCard>

        <SCard title="Throughput">
          <SSkeleton v-if="timelineLoading" variant="rect" height="256px" />
          <SEmptyState v-else-if="!timeline || timeline.throughput.length === 0" title="No throughput data" description="Throughput data will appear after syncs complete." />
          <ThroughputChart
            v-else
            :labels="timeline.throughput.map(p => p.label)"
            :records-read="timeline.throughput.map(p => p.recordsRead)"
          />
        </SCard>

        <SCard title="Duration Trends">
          <SSkeleton v-if="timelineLoading" variant="rect" height="256px" />
          <SEmptyState v-else-if="!timeline || timeline.durations.length === 0" title="No duration data" description="Duration trends will appear after syncs complete." />
          <DurationTrendsChart
            v-else
            :labels="timeline.durations.map(p => p.label)"
            :durations="timeline.durations.map(p => p.avgDurationMs)"
          />
        </SCard>

        <SCard title="Failure Breakdown">
          <SEmptyState
            v-if="!stats.failureBreakdown || stats.failureBreakdown.length === 0 || stats.failureBreakdown.every(f => f.count === 0)"
            title="No failures"
            description="No failure data to display. All syncs completed successfully."
          />
          <FailureBreakdownChart
            v-else
            :labels="stats.failureBreakdown.map(f => f.category)"
            :counts="stats.failureBreakdown.map(f => f.count)"
          />
        </SCard>
      </div>

      <!-- Top Connections -->
      <SCard title="Top Connections" :padding="false" class="mb-6">
        <SEmptyState v-if="!stats.topConnections || stats.topConnections.length === 0" title="No connections" description="Top connections will appear after syncs complete." />
        <TopConnectionsTable v-else :connections="stats.topConnections" />
      </SCard>

      <!-- Connection Health Grid -->
      <SCard title="Connection Health" class="mb-6">
        <SEmptyState v-if="!stats.connectionHealths || stats.connectionHealths.length === 0" title="No connections" description="Connection health grid will appear after adding connections." />
        <ConnectionHealthGrid v-else :connections="stats.connectionHealths" />
      </SCard>
    </div>
  </div>
</template>
