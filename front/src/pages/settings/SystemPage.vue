<script setup lang="ts">
import type { ExecutorInstance } from '@entities/system'
import type { Column } from '@shared/ui'
import { executorKindLabels, useExecutors } from '@entities/system'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { formatRelativeTime } from '@shared/lib/format'
import { SAlert, SCard, SEmptyState, STable } from '@shared/ui'

const executors = useExecutors()

const columns: Column[] = [
  { key: 'id', label: 'Executor' },
  { key: 'type', label: 'Type' },
  { key: 'version', label: 'Version' },
  { key: 'capacity', label: 'Capacity' },
  { key: 'capabilities', label: 'Capabilities' },
  { key: 'lastSeenAt', label: 'Last seen' },
]

function capabilities(executor: ExecutorInstance): string {
  return executor.nativeDestinations.length > 0 ? executor.nativeDestinations.join(', ') : '—'
}
</script>

<template>
  <SCard
    title="Executors"
    description="Executors that reported to the control plane within the silence timeout. The list refreshes every 15 seconds."
  >
    <SAlert v-if="executors.error.value" variant="danger">
      {{ getErrorMessage(executors.error.value) || 'Failed to load executors' }}
    </SAlert>
    <STable v-else :columns="columns" :data="executors.data.value ?? []" :loading="executors.isLoading.value">
      <template #empty>
        <SEmptyState title="No executors" description="No executor has reported to the control plane yet." />
      </template>
      <template #cell-id="{ row }">
        <span class="font-mono">{{ row.id }}</span>
      </template>
      <template #cell-type="{ row }">
        {{ executorKindLabels[(row as ExecutorInstance).type] }}
      </template>
      <template #cell-capacity="{ row }">
        {{ row.capacity > 0 ? row.capacity : 'Unlimited' }}
      </template>
      <template #cell-capabilities="{ row }">
        {{ capabilities(row) }}
      </template>
      <template #cell-lastSeenAt="{ row }">
        {{ row.lastSeenAt ? formatRelativeTime(row.lastSeenAt) : '—' }}
      </template>
    </STable>
  </SCard>
</template>
