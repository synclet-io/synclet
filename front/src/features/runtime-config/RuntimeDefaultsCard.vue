<script setup lang="ts">
import { executorKindLabels, useRuntimeDefaults } from '@entities/system'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SButton, SCard, SSkeleton } from '@shared/ui'
import { computed } from 'vue'

const runtimeDefaults = useRuntimeDefaults()

function formatSeconds(total: number): string {
  if (total <= 0)
    return 'None'
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  return [h && `${h}h`, m && `${m}m`, s && `${s}s`].filter(Boolean).join(' ')
}

const rows = computed(() => {
  const rd = runtimeDefaults.data.value
  if (!rd)
    return []
  const notSet = 'Not set'
  return [
    { label: 'CPU request', value: rd.cpuRequest || notSet },
    { label: 'CPU limit', value: rd.cpuLimit || notSet },
    { label: 'Memory request', value: rd.memoryRequest || notSet },
    { label: 'Memory limit', value: rd.memoryLimit || notSet },
    { label: 'Service account', value: rd.serviceAccountName || notSet },
    { label: 'Max concurrent jobs', value: rd.maxConcurrentJobs > 0 ? String(rd.maxConcurrentJobs) : 'Unlimited' },
    { label: 'Idle timeout', value: formatSeconds(rd.idleTimeoutSeconds) },
    { label: 'Job log archive', value: rd.logArchiveBackend || 'none' },
    ...rd.executors.flatMap((executor) => {
      const name = executorKindLabels[executor.type]
      const res = executor.resourceDefaults
      const resource = (value: string, executorDefault: boolean) =>
        !value ? notSet : executorDefault ? `${value} (executor default)` : value
      return [
        { label: `${name} · Capacity`, value: executor.capacity > 0 ? String(executor.capacity) : 'Unlimited' },
        { label: `${name} · CPU request`, value: resource(res.cpuRequest, res.cpuExecutorDefault) },
        { label: `${name} · CPU limit`, value: resource(res.cpuLimit, res.cpuExecutorDefault) },
        { label: `${name} · Memory request`, value: resource(res.memoryRequest, res.memoryExecutorDefault) },
        { label: `${name} · Memory limit`, value: resource(res.memoryLimit, res.memoryExecutorDefault) },
      ]
    }),
  ]
})

const noExecutors = computed(() => runtimeDefaults.data.value?.executors.length === 0)
</script>

<template>
  <SCard
    title="Runtime defaults"
    description="Server-wide settings applied when a source or destination has no runtime config of its own. Set by server environment variables; read-only here."
  >
    <div v-if="runtimeDefaults.isLoading.value" class="space-y-2" data-testid="runtime-defaults-loading">
      <SSkeleton v-for="i in 4" :key="i" variant="text" />
    </div>
    <SAlert v-else-if="runtimeDefaults.isError.value" variant="danger">
      <div class="flex items-center justify-between gap-3">
        <span>{{ getErrorMessage(runtimeDefaults.error.value) || 'Failed to load runtime defaults' }}</span>
        <SButton variant="secondary" size="sm" @click="runtimeDefaults.refetch()">
          Retry
        </SButton>
      </div>
    </SAlert>
    <dl v-else class="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-3">
      <div v-for="row in rows" :key="row.label" class="flex items-baseline justify-between gap-4 border-b border-border pb-2">
        <dt class="text-sm text-text-secondary">
          {{ row.label }}
        </dt>
        <dd class="text-sm font-mono text-text-primary text-right">
          {{ row.value }}
        </dd>
      </div>
    </dl>
    <p v-if="noExecutors" class="mt-3 text-sm text-text-secondary">
      No executor has reported yet
    </p>
  </SCard>
</template>
