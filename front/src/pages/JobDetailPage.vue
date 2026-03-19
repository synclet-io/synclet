<script setup lang="ts">
import { useJob, useJobLogs } from '@entities/job'
import { formatDuration } from '@shared/lib/format'
import { PageHeader, SAlert, SCard, SLogViewer, SSkeleton, StatusBadge } from '@shared/ui'
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const jobId = computed(() => route.params.id as string)

const { data: job, isLoading, error } = useJob(jobId)

const latestAttempt = computed(() => {
  if (!job.value?.attempts?.length)
    return undefined
  return job.value.attempts.at(-1)
})

const isActive = computed(() => job.value?.status === 'running' || job.value?.status === 'scheduled' || job.value?.status === 'starting')

const jobIdRef = ref(jobId.value)
// Keep jobIdRef in sync with route param
watch(jobId, (val) => {
  jobIdRef.value = val
}, { immediate: true })

const { lines, isLoading: logsLoading } = useJobLogs(jobIdRef, isActive)

function formatDate(d: Date | undefined): string {
  return d ? d.toLocaleString() : '-'
}
</script>

<template>
  <div>
    <PageHeader
      :title="job ? `Job ${job.id.slice(0, 8)}...` : 'Job Detail'"
      :back-to="job?.connectionId ? { name: 'connection-detail', params: { id: job.connectionId } } : { name: 'jobs' }"
      back-label="Back to connection"
    />

    <!-- Loading state -->
    <div v-if="isLoading" class="space-y-4">
      <SSkeleton class="h-32" />
      <SSkeleton class="h-64" />
    </div>

    <!-- Error state -->
    <SAlert v-else-if="error" variant="danger">
      Failed to load job details.
    </SAlert>

    <!-- Content -->
    <div v-else-if="job" class="space-y-6">
      <!-- Metadata grid -->
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <SCard>
          <div class="text-sm text-text-muted mb-1">
            Status
          </div>
          <StatusBadge :status="job.status" />
        </SCard>
        <SCard>
          <div class="text-sm text-text-muted mb-1">
            Duration
          </div>
          <div class="text-heading font-semibold">
            {{ formatDuration(latestAttempt?.syncStats?.durationMs) }}
          </div>
        </SCard>
        <SCard>
          <div class="text-sm text-text-muted mb-1">
            Records Read
          </div>
          <div class="text-heading font-semibold">
            {{ latestAttempt?.syncStats?.recordsRead ?? '-' }}
          </div>
        </SCard>
      </div>

      <!-- Failure reason -->
      <SAlert v-if="job.status === 'failed' && job.error" variant="danger">
        {{ job.error }}
      </SAlert>

      <!-- Timeline -->
      <SCard title="Timeline">
        <div class="grid grid-cols-2 gap-4 text-sm">
          <div><span class="text-text-muted">Started:</span> {{ formatDate(job.startedAt) }}</div>
          <div><span class="text-text-muted">Completed:</span> {{ formatDate(job.completedAt) }}</div>
          <div><span class="text-text-muted">Attempt:</span> {{ job.attempt }} / {{ job.maxAttempts }}</div>
          <div><span class="text-text-muted">Type:</span> {{ job.jobType }}</div>
        </div>
      </SCard>

      <!-- Log viewer -->
      <SCard title="Logs">
        <SLogViewer
          :lines="lines"
          :loading="logsLoading"
        />
      </SCard>
    </div>
  </div>
</template>
