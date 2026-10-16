<script setup lang="ts">
import type { Job } from '@entities/job'
import type { JobStatusFilterValue } from '@entities/job/statusFilters'
import type { Column } from '@shared/ui'
import { useAuth } from '@entities/auth'
import { useConnections } from '@entities/connection'
import { isActiveJobStatus, useCancelJob } from '@entities/job'
import { listJobs } from '@entities/job/api'
import { JOB_STATUS_FILTERS } from '@entities/job/statusFilters'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { formatDuration, statusVariant } from '@shared/lib/format'
import { jobKeys } from '@shared/lib/queryKeys'
import { PageHeader, SAlert, SBadge, SButton, SConfirmDialog, SPagination, SSelect, STable, useToast } from '@shared/ui'
import { useQuery } from '@tanstack/vue-query'
import { XCircle } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'

const PAGE_SIZE = 25

const route = useRoute()
const router = useRouter()

function queryString(key: string): string {
  const value = route.query[key]
  return typeof value === 'string' ? value : ''
}

function setQuery(patch: Record<string, string | undefined>, resetPage = true) {
  const query: Record<string, string> = {}
  for (const [key, value] of Object.entries({ ...route.query, ...(resetPage ? { page: undefined } : {}), ...patch })) {
    if (typeof value === 'string' && value !== '')
      query[key] = value
  }
  router.replace({ query })
}

const connectionId = computed<string>({
  get: () => queryString('connection'),
  set: value => setQuery({ connection: value || undefined }),
})

const statusFilter = computed(() => {
  const value = queryString('status')
  return JOB_STATUS_FILTERS.find(f => f.value === value) ?? JOB_STATUS_FILTERS[0]
})

function setStatusFilter(value: JobStatusFilterValue) {
  setQuery({ status: value || undefined })
}

const page = computed(() => {
  const n = Number.parseInt(queryString('page'), 10)
  return Number.isInteger(n) && n > 1 ? n : 1
})

function setPage(n: number) {
  setQuery({ page: n > 1 ? String(n) : undefined }, false)
}

const { currentWorkspaceId } = useAuth()
const { data: connections } = useConnections()

const connectionNames = computed(() => new Map((connections.value?.items ?? []).map(c => [c.id, c.name])))

const connectionOptions = computed(() => [
  { label: 'All connections', value: '' },
  ...(connections.value?.items ?? []).map(c => ({ label: c.name, value: c.id })),
])

function connectionName(id: string): string {
  return connectionNames.value.get(id) ?? id
}

const { data, isLoading: loading, error } = useQuery({
  queryKey: computed(() => [
    ...jobKeys.list(currentWorkspaceId.value ?? '', connectionId.value || 'all'),
    statusFilter.value.value,
    page.value,
    PAGE_SIZE,
  ]),
  queryFn: () => listJobs(connectionId.value, {
    statuses: statusFilter.value.statuses,
    pageSize: PAGE_SIZE,
    offset: (page.value - 1) * PAGE_SIZE,
  }),
  enabled: computed(() => !!currentWorkspaceId.value),
  refetchInterval: (query) => {
    const hasActive = query.state.data?.items.some(j => isActiveJobStatus(j.status))
    return hasActive ? 2_000 : 10_000
  },
})

const hasFilters = computed(() => !!connectionId.value || statusFilter.value.value !== '')

const emptyText = computed(() => {
  if (hasFilters.value)
    return 'No jobs match these filters'
  return 'No jobs yet'
})

const emptyDescription = computed(() => {
  if (hasFilters.value)
    return 'Try another connection or status.'
  return 'Jobs appear here once a connection syncs.'
})

function lastAttemptStats(job: Job) {
  return job.attempts.at(-1)?.syncStats
}

function openJob(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (target.closest('a, button'))
    return
  const row = target.closest('tbody tr') as HTMLTableRowElement | null
  const job = row ? data.value?.items[row.sectionRowIndex] : undefined
  if (job)
    router.push({ name: 'job-detail', params: { id: job.id } })
}

const cancelJobMutation = useCancelJob()
const toast = useToast()
const jobToCancel = ref<Job | null>(null)
const cancelError = ref('')

const cancelMessage = computed(() => {
  const job = jobToCancel.value
  if (!job)
    return ''
  return `Cancel job ${job.id.slice(0, 8)} on "${connectionName(job.connectionId)}"? Its sync stops and it will not be retried.`
})

function requestCancel(job: Job) {
  cancelError.value = ''
  jobToCancel.value = job
}

function closeCancel() {
  jobToCancel.value = null
  cancelError.value = ''
}

async function confirmCancel() {
  const job = jobToCancel.value
  if (!job)
    return
  try {
    await cancelJobMutation.mutateAsync(job.id)
    toast.success('Job cancelled')
    closeCancel()
  }
  catch (e: unknown) {
    cancelError.value = getErrorMessage(e)
  }
}

const columns = computed<Column[]>(() => [
  { key: 'status', label: 'Status' },
  ...(connectionId.value ? [] : [{ key: 'connection', label: 'Connection' }]),
  { key: 'type', label: 'Type' },
  { key: 'started', label: 'Started' },
  { key: 'duration', label: 'Duration' },
  { key: 'records', label: 'Records' },
  { key: 'attempt', label: 'Attempt' },
  { key: 'actions', label: '', align: 'right' },
])
</script>

<template>
  <PageHeader title="Jobs" description="Sync job history across your connections" />

  <div class="mb-4 flex flex-wrap items-center gap-3">
    <div class="w-full sm:w-64">
      <SSelect v-model="connectionId" :options="connectionOptions" aria-label="Connection" />
    </div>
    <div class="inline-flex flex-wrap gap-1 bg-surface-raised rounded-lg p-1" role="group" aria-label="Status">
      <button
        v-for="option in JOB_STATUS_FILTERS"
        :key="option.value"
        type="button"
        class="rounded-md px-3 py-1 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
        :class="statusFilter.value === option.value
          ? 'bg-primary-solid text-on-primary'
          : 'text-text-secondary hover:text-text-primary'"
        :aria-pressed="statusFilter.value === option.value"
        @click="setStatusFilter(option.value)"
      >
        {{ option.label }}
      </button>
    </div>
  </div>

  <SAlert v-if="error" variant="danger" class="mb-4">
    {{ getErrorMessage(error) }}
  </SAlert>

  <div class="[&_tbody_tr]:cursor-pointer" @click="openJob">
    <STable
      :columns="columns"
      :data="data?.items"
      :loading="loading"
      :empty-text="emptyText"
      :empty-description="emptyDescription"
    >
      <template #cell-status="{ row }">
        <RouterLink :to="{ name: 'job-detail', params: { id: row.id } }" class="focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 rounded-full">
          <SBadge :variant="statusVariant(row.status)" dot>
            {{ row.status }}
          </SBadge>
        </RouterLink>
      </template>
      <template #cell-connection="{ row }">
        <RouterLink
          :to="{ name: 'connection-detail', params: { id: row.connectionId } }"
          class="text-sm text-text-secondary hover:text-primary transition-colors"
        >
          {{ connectionName(row.connectionId) }}
        </RouterLink>
      </template>
      <template #cell-type="{ row }">
        <span class="text-sm text-text-secondary">{{ row.jobType }}</span>
      </template>
      <template #cell-started="{ row }">
        <span class="text-sm text-text-secondary">{{ row.startedAt?.toLocaleString() ?? '-' }}</span>
      </template>
      <template #cell-duration="{ row }">
        <span class="text-sm text-text-secondary">{{ formatDuration(lastAttemptStats(row)?.durationMs) }}</span>
      </template>
      <template #cell-records="{ row }">
        <span class="text-sm text-text-secondary">{{ lastAttemptStats(row)?.recordsRead?.toLocaleString() ?? '-' }}</span>
      </template>
      <template #cell-attempt="{ row }">
        <span class="text-sm text-text-secondary">{{ row.attempt }}/{{ row.maxAttempts }}</span>
      </template>
      <template #cell-actions="{ row }">
        <SButton
          v-if="isActiveJobStatus(row.status)"
          variant="ghost"
          size="icon"
          title="Cancel job"
          aria-label="Cancel job"
          @click="requestCancel(row)"
        >
          <XCircle class="w-4 h-4" />
        </SButton>
      </template>
    </STable>
  </div>

  <SPagination
    :total="data?.total ?? 0"
    :page-size="PAGE_SIZE"
    :current-page="page"
    class="mt-4"
    @page-change="setPage"
  />

  <SConfirmDialog
    :open="!!jobToCancel"
    title="Cancel job"
    :message="cancelMessage"
    confirm-text="Cancel job"
    cancel-text="Keep running"
    :loading="cancelJobMutation.isPending.value"
    :error="cancelError"
    @confirm="confirmCancel"
    @cancel="closeCancel"
  />
</template>
