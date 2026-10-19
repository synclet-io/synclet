<script setup lang="ts">
import type { JobAttempt } from '@entities/job'
import { downloadJobLogs, isActiveJobStatus, useJob, useJobLogs } from '@entities/job'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { formatDuration, statusVariant } from '@shared/lib/format'
import { PageHeader, SAlert, SBadge, SCard, SLogViewer, SSkeleton, useToast } from '@shared/ui'
import { CheckCircle2, ChevronDown, ChevronRight, Clock, Download, Loader2, PlayCircle, XCircle } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'

const LOG_TIMESTAMP_RE = /^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?/

const route = useRoute()
const jobId = computed(() => route.params.id as string)

const { data: job, isLoading, error } = useJob(jobId)

const latestAttempt = computed(() => {
  if (!job.value?.attempts?.length)
    return undefined
  return job.value.attempts.at(-1)
})

const isActive = computed(() => isActiveJobStatus(job.value?.status))

const { lines, droppedLines, isLoading: logsLoading } = useJobLogs(jobId, isActive)

const totalDropped = computed(() => Object.values(droppedLines.value).reduce((sum, n) => sum + n, 0))

function droppedLabel(n: number): string {
  return `${n.toLocaleString()} ${n === 1 ? 'line' : 'lines'} dropped`
}

const toast = useToast()
const downloading = ref<Record<number, boolean>>({})

async function downloadAttemptLogs(attempt: number) {
  downloading.value[attempt] = true
  try {
    const blob = await downloadJobLogs(jobId.value, attempt)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `job-${jobId.value}-attempt-${attempt}.log`
    a.click()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  }
  catch (e: unknown) {
    toast.error(getErrorMessage(e) || 'Failed to download logs')
  }
  finally {
    downloading.value[attempt] = false
  }
}

const expanded = ref<Record<string, boolean>>({})
function toggle(id: string) {
  expanded.value[id] = !expanded.value[id]
}

function formatDate(d: Date | undefined): string {
  return d ? d.toLocaleString() : '-'
}

function durationBetween(start: Date | undefined, end: Date | undefined): string {
  if (!start)
    return '-'
  const endTime = end?.getTime() ?? Date.now()
  return formatDuration(endTime - start.getTime())
}

function attemptStatus(att: JobAttempt): { label: string, variant: 'success' | 'danger' | 'info' | 'gray' } {
  if (att.error)
    return { label: 'Failed', variant: 'danger' }
  if (att.completedAt)
    return { label: 'Succeeded', variant: 'success' }
  return { label: 'Running', variant: 'info' }
}

interface TimelineEvent {
  id: string
  label: string
  at: Date | undefined
  duration?: string
  icon: typeof Clock
  iconColor: string
  details?: string
  attempt?: JobAttempt
}

const timeline = computed<TimelineEvent[]>(() => {
  if (!job.value)
    return []
  const events: TimelineEvent[] = []

  if (job.value.scheduledAt) {
    events.push({
      id: 'scheduled',
      label: 'Scheduled',
      at: job.value.scheduledAt,
      duration: job.value.startedAt ? durationBetween(job.value.scheduledAt, job.value.startedAt) : undefined,
      icon: Clock,
      iconColor: 'text-text-muted',
      details: job.value.startedAt ? 'Waited in queue' : 'Waiting in queue…',
    })
  }

  if (job.value.startedAt) {
    events.push({
      id: 'started',
      label: 'Job started',
      at: job.value.startedAt,
      icon: PlayCircle,
      iconColor: 'text-primary',
    })
  }

  for (const att of job.value.attempts ?? []) {
    const status = attemptStatus(att)
    events.push({
      id: `attempt-${att.id}`,
      label: `Attempt ${att.attemptNumber}`,
      at: att.startedAt,
      duration: att.startedAt ? durationBetween(att.startedAt, att.completedAt) : undefined,
      icon: status.variant === 'success' ? CheckCircle2 : status.variant === 'danger' ? XCircle : Loader2,
      iconColor: status.variant === 'success'
        ? 'text-success'
        : status.variant === 'danger' ? 'text-danger' : 'text-primary',
      details: att.error || (att.syncStats ? `${att.syncStats.recordsRead.toLocaleString()} records, ${formatDuration(att.syncStats.durationMs)}` : 'In progress…'),
      attempt: att,
    })
  }

  if (job.value.completedAt) {
    const status = job.value.status
    events.push({
      id: 'completed',
      label: status === 'failed' ? 'Job failed' : status === 'cancelled' ? 'Job cancelled' : 'Job completed',
      at: job.value.completedAt,
      duration: job.value.startedAt ? durationBetween(job.value.startedAt, job.value.completedAt) : undefined,
      icon: status === 'completed' ? CheckCircle2 : XCircle,
      iconColor: status === 'completed' ? 'text-success' : 'text-danger',
      details: job.value.error || undefined,
    })
  }

  return events
})

// Logs aren't tagged by attempt at the storage layer, so we approximate by time.
function logsForAttempt(attempt: JobAttempt): string[] {
  if (!attempt.startedAt)
    return []
  const start = attempt.startedAt.getTime()
  const end = attempt.completedAt?.getTime() ?? Date.now()
  const result: string[] = []
  for (const line of lines.value) {
    const ts = parseLogTimestamp(line)
    if (ts === null) {
      result.push(line)
      continue
    }
    if (ts >= start && ts <= end)
      result.push(line)
  }
  return result
}

// Try to parse an ISO-ish timestamp from the start of a log line.
function parseLogTimestamp(line: string): number | null {
  const match = line.match(LOG_TIMESTAMP_RE)
  if (!match)
    return null
  const t = Date.parse(match[0])
  return Number.isNaN(t) ? null : t
}
</script>

<template>
  <div>
    <PageHeader
      :title="job ? `Job ${job.id.slice(0, 8)}...` : 'Job Detail'"
      :back-to="job?.connectionId ? { name: 'connection-detail', params: { id: job.connectionId } } : { name: 'jobs' }"
      back-label="Back to connection"
    />

    <div v-if="isLoading" class="space-y-4">
      <SSkeleton variant="rect" class="h-32" />
      <SSkeleton variant="rect" class="h-64" />
    </div>

    <SAlert v-else-if="error" variant="danger">
      Failed to load job details.
    </SAlert>

    <div v-else-if="job" class="space-y-6">
      <!-- Metadata grid -->
      <div class="grid grid-cols-1 lg:grid-cols-4 gap-4">
        <SCard>
          <div class="text-sm text-text-muted mb-1">
            Status
          </div>
          <SBadge :variant="statusVariant(job.status)" dot>
            {{ job.status }}
          </SBadge>
        </SCard>
        <SCard>
          <div class="text-sm text-text-muted mb-1">
            Total duration
          </div>
          <div class="text-heading font-semibold">
            {{ durationBetween(job.startedAt, job.completedAt) }}
          </div>
        </SCard>
        <SCard>
          <div class="text-sm text-text-muted mb-1">
            Records (latest)
          </div>
          <div class="text-heading font-semibold">
            {{ latestAttempt?.syncStats?.recordsRead?.toLocaleString() ?? '-' }}
          </div>
        </SCard>
        <SCard>
          <div class="text-sm text-text-muted mb-1">
            Attempt
          </div>
          <div class="text-heading font-semibold">
            {{ job.attempt }} / {{ job.maxAttempts }}
          </div>
        </SCard>
      </div>

      <SAlert v-if="job.status === 'failed' && job.error" variant="danger">
        {{ job.error }}
      </SAlert>

      <!-- Timeline -->
      <SCard title="Timeline">
        <ol class="space-y-1">
          <li v-for="ev in timeline" :key="ev.id" class="relative">
            <div class="flex items-start gap-3 py-2">
              <div class="mt-0.5 shrink-0">
                <component :is="ev.icon" class="w-5 h-5" :class="ev.iconColor" />
              </div>
              <div class="flex-1 min-w-0">
                <div class="flex items-baseline gap-3 flex-wrap">
                  <span class="font-medium text-heading text-sm">{{ ev.label }}</span>
                  <span v-if="ev.duration" class="text-xs text-text-muted">{{ ev.duration }}</span>
                  <span
                    v-if="ev.attempt && droppedLines[ev.attempt.attemptNumber]"
                    class="text-xs text-warning"
                    :data-testid="`attempt-${ev.attempt.attemptNumber}-dropped`"
                  >{{ droppedLabel(droppedLines[ev.attempt.attemptNumber]) }}</span>
                  <button
                    v-if="ev.attempt"
                    type="button"
                    class="ml-auto inline-flex items-center gap-1 text-xs text-primary hover:underline disabled:opacity-50"
                    :disabled="downloading[ev.attempt.attemptNumber]"
                    :data-testid="`attempt-${ev.attempt.attemptNumber}-download`"
                    @click="downloadAttemptLogs(ev.attempt.attemptNumber)"
                  >
                    <component :is="downloading[ev.attempt.attemptNumber] ? Loader2 : Download" class="w-3.5 h-3.5" :class="{ 'animate-spin': downloading[ev.attempt.attemptNumber] }" />
                    Download logs
                  </button>
                  <button
                    v-if="ev.attempt"
                    type="button"
                    class="inline-flex items-center gap-1 text-xs text-primary hover:underline"
                    @click="toggle(ev.id)"
                  >
                    <component :is="expanded[ev.id] ? ChevronDown : ChevronRight" class="w-3.5 h-3.5" />
                    {{ expanded[ev.id] ? 'Hide attempt logs' : 'View attempt logs' }}
                  </button>
                </div>
                <div class="text-xs text-text-muted mt-0.5">
                  {{ formatDate(ev.at) }}<span v-if="ev.details"> · {{ ev.details }}</span>
                </div>

                <div v-if="ev.attempt && expanded[ev.id]" class="mt-3">
                  <SLogViewer
                    :lines="logsForAttempt(ev.attempt)"
                    :loading="logsLoading && logsForAttempt(ev.attempt).length === 0"
                  />
                </div>
              </div>
            </div>
            <div v-if="ev !== timeline[timeline.length - 1]" class="absolute left-[9px] top-7 w-px h-[calc(100%-12px)] bg-border" />
          </li>
        </ol>
      </SCard>

      <!-- Full unified log viewer -->
      <SCard title="All logs">
        <p class="text-xs text-text-muted mb-3">
          Combined log stream across all attempts. Use the timeline above to expand or download a specific attempt.
          <span v-if="totalDropped > 0" class="text-warning" data-testid="all-logs-dropped">{{ droppedLabel(totalDropped) }}</span>
        </p>
        <SLogViewer
          :lines="lines"
          :loading="logsLoading"
        />
      </SCard>
    </div>
  </div>
</template>
