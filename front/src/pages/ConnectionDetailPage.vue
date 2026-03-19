<script setup lang="ts">
import type { TimeRange } from '@entities/stats'
import type { Column } from '@shared/ui'
import { useConnection } from '@entities/connection'
import { useDestination } from '@entities/destination'
import { useJobs } from '@entities/job'
import { useSource } from '@entities/source'
import { useConnectionStats } from '@entities/stats'
import ConnectionStatsCards from '@features/connection-stats/ConnectionStatsCards.vue'
import RecordsChart from '@features/connection-stats/RecordsChart.vue'
import SyncDurationChart from '@features/connection-stats/SyncDurationChart.vue'
import TimeRangeSelector from '@features/dashboard/TimeRangeSelector.vue'
import { statusVariant } from '@shared/lib/format'
import { SBadge, SCard, SEmptyState, SPagination, SSkeleton, STable } from '@shared/ui'
import { ArrowRight, ExternalLink } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

const route = useRoute()
const id = route.params.id as string

const PAGE_SIZE = 25
const currentPage = ref(1)
const { data: connection } = useConnection(id)
const { data: source } = useSource(computed(() => connection.value?.sourceId ?? ''))
const { data: destination } = useDestination(computed(() => connection.value?.destinationId ?? ''))
const { data: jobs } = useJobs(id, { page: currentPage, pageSize: PAGE_SIZE })
const timeRange = ref<TimeRange>('24h')
const { data: connStats, isLoading: statsLoading } = useConnectionStats(id, timeRange)

const jobColumns: Column[] = [
  { key: 'status', label: 'Status' },
  { key: 'type', label: 'Type' },
  { key: 'started', label: 'Started' },
  { key: 'duration', label: 'Duration' },
  { key: 'attempt', label: 'Attempt' },
  { key: 'actions', label: '', align: 'right' },
]
</script>

<template>
  <template v-if="connection">
    <SCard class="mb-6">
      <div class="flex items-center justify-between p-4">
        <RouterLink
          :to="{ name: 'source-detail', params: { id: connection.sourceId } }"
          class="flex items-center gap-2 text-sm font-medium text-primary hover:underline"
        >
          {{ source?.name ?? 'Source' }}
        </RouterLink>
        <ArrowRight class="w-4 h-4 text-text-muted" />
        <RouterLink
          :to="{ name: 'destination-detail', params: { id: connection.destinationId } }"
          class="flex items-center gap-2 text-sm font-medium text-primary hover:underline"
        >
          {{ destination?.name ?? 'Destination' }}
        </RouterLink>
      </div>
    </SCard>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
      <SCard>
        <p class="text-xs text-text-secondary uppercase mb-1">
          Status
        </p>
        <SBadge :variant="statusVariant(connection.status)" dot>
          {{ connection.status }}
        </SBadge>
      </SCard>
      <SCard>
        <p class="text-xs text-text-secondary uppercase mb-1">
          Schedule
        </p>
        <p class="text-sm font-medium text-text-primary">
          {{ connection.schedule || 'Manual' }}
        </p>
      </SCard>
      <SCard>
        <p class="text-xs text-text-secondary uppercase mb-1">
          Schema Policy
        </p>
        <p class="text-sm font-medium text-text-primary">
          {{ connection.schemaChangePolicy }}
        </p>
      </SCard>
    </div>

    <!-- Replication Stats section -->
    <div class="mb-6">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-[15px] font-semibold text-heading">
          Replication Stats
        </h3>
        <TimeRangeSelector v-model="timeRange" />
      </div>

      <div v-if="statsLoading" class="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <SSkeleton v-for="i in 4" :key="i" variant="rect" height="90px" />
      </div>
      <ConnectionStatsCards v-else-if="connStats" :stats="connStats" class="mb-6" />

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <SCard title="Recent Sync Duration">
          <SSkeleton v-if="statsLoading" variant="rect" height="256px" />
          <SyncDurationChart v-else-if="connStats?.durationChart?.length" :syncs="connStats.durationChart" />
          <SEmptyState v-else title="No sync history" description="This connection has not completed any syncs yet. Trigger a manual sync or wait for the next scheduled run." />
        </SCard>
        <SCard title="Records Over Time">
          <SSkeleton v-if="statsLoading" variant="rect" height="256px" />
          <RecordsChart
            v-else-if="connStats?.recordsChart?.length"
            :labels="connStats.recordsChart.map(p => p.label)"
            :read="connStats.recordsChart.map(p => p.recordsRead)"
          />
          <SEmptyState v-else title="No sync history" description="This connection has not completed any syncs yet. Trigger a manual sync or wait for the next scheduled run." />
        </SCard>
      </div>
    </div>

    <SCard title="Sync History" :padding="false">
      <STable :columns="jobColumns" :data="jobs?.items" empty-text="No sync jobs yet">
        <template #cell-status="{ row }">
          <SBadge :variant="statusVariant(row.status)" dot>
            {{ row.status }}
          </SBadge>
        </template>
        <template #cell-type="{ row }">
          <span class="text-sm text-text-secondary">{{ row.jobType }}</span>
        </template>
        <template #cell-started="{ row }">
          <span class="text-sm text-text-secondary">{{ row.startedAt?.toLocaleString() ?? '-' }}</span>
        </template>
        <template #cell-duration="{ row }">
          <span class="text-sm text-text-secondary">
            {{ row.attempts?.[row.attempts.length - 1]?.syncStats?.durationMs
              ? `${(row.attempts[row.attempts.length - 1].syncStats!.durationMs / 1000).toFixed(1)}s`
              : '-' }}
          </span>
        </template>
        <template #cell-attempt="{ row }">
          <span class="text-sm text-text-secondary">{{ row.attempt }}/{{ row.maxAttempts }}</span>
        </template>
        <template #cell-actions="{ row }">
          <RouterLink :to="{ name: 'job-detail', params: { id: row.id } }" class="p-1.5 text-text-muted hover:text-primary transition-colors" title="View Details">
            <ExternalLink class="w-4 h-4" />
          </RouterLink>
        </template>
      </STable>
      <SPagination
        :total="jobs?.total ?? 0"
        :page-size="PAGE_SIZE"
        :current-page="currentPage"
        class="mt-4"
        @page-change="currentPage = $event"
      />
    </SCard>
  </template>
</template>
