<script setup lang="ts">
import type { Column } from '@shared/ui'
import { useConnections } from '@entities/connection'
import { useCancelJob, useJobs } from '@entities/job'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { formatDuration, statusVariant } from '@shared/lib/format'
import { PageHeader, SAlert, SBadge, SPagination, SSelect, STable, useToast } from '@shared/ui'
import { ExternalLink, XCircle } from 'lucide-vue-next'
import { ref, watch } from 'vue'
import { RouterLink } from 'vue-router'

const { data: connections } = useConnections()
const selectedConnection = ref('')
const currentPage = ref(1)
const PAGE_SIZE = 25
const cancelJobMutation = useCancelJob()
const toast = useToast()

watch(connections, (conns) => {
  if (conns && conns.items.length > 0 && !selectedConnection.value) {
    selectedConnection.value = conns.items[0].id
  }
}, { immediate: true })

// Reset page when connection changes
watch(selectedConnection, () => {
  currentPage.value = 1
})

const { data, isLoading: loading, error } = useJobs(selectedConnection, { page: currentPage, pageSize: PAGE_SIZE })

function connectionOptions() {
  return (connections.value?.items ?? []).map(c => ({ label: c.name, value: c.id }))
}

async function cancelJob(jobId: string) {
  try {
    await cancelJobMutation.mutateAsync(jobId)
    toast.success('Job cancelled')
  }
  catch (e: unknown) {
    toast.error(`Error: ${getErrorMessage(e)}`)
  }
}

const columns: Column[] = [
  { key: 'status', label: 'Status' },
  { key: 'type', label: 'Type' },
  { key: 'started', label: 'Started' },
  { key: 'duration', label: 'Duration' },
  { key: 'records', label: 'Records' },
  { key: 'attempt', label: 'Attempt' },
  { key: 'actions', label: 'Actions', align: 'right' },
]
</script>

<template>
  <PageHeader title="Jobs" description="View sync job history and status" />

  <div class="mb-4 max-w-xs">
    <SSelect
      v-model="selectedConnection"
      :options="connectionOptions()"
      placeholder="Select a connection"
    />
  </div>

  <SAlert v-if="error" variant="danger" class="mb-4">
    {{ error.message }}
  </SAlert>

  <STable :columns="columns" :data="data?.items" :loading="loading" empty-text="No jobs found for this connection">
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
      <span class="text-sm text-text-secondary">{{ formatDuration(row.attempts?.[row.attempts.length - 1]?.syncStats?.durationMs) }}</span>
    </template>
    <template #cell-records="{ row }">
      <span class="text-sm text-text-secondary">
        {{ row.attempts?.[row.attempts.length - 1]?.syncStats?.recordsRead != null
          ? row.attempts[row.attempts.length - 1].syncStats!.recordsRead.toLocaleString()
          : '-' }}
      </span>
    </template>
    <template #cell-attempt="{ row }">
      <span class="text-sm text-text-secondary">{{ row.attempt }}/{{ row.maxAttempts }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="flex items-center justify-end gap-1">
        <RouterLink :to="{ name: 'job-detail', params: { id: row.id } }" class="p-1.5 text-text-muted hover:text-primary transition-colors" title="View Logs">
          <ExternalLink class="w-4 h-4" />
        </RouterLink>
        <button
          v-if="row.status === 'running' || row.status === 'scheduled' || row.status === 'starting'"
          class="p-1.5 text-text-muted hover:text-danger transition-colors" title="Cancel" @click="cancelJob(row.id)"
        >
          <XCircle class="w-4 h-4" />
        </button>
      </div>
    </template>
  </STable>

  <SPagination
    :total="data?.total ?? 0"
    :page-size="PAGE_SIZE"
    :current-page="currentPage"
    class="mt-4"
    @page-change="currentPage = $event"
  />
</template>
