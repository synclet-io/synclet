<script setup lang="ts">
import type { Column } from '@shared/ui'
import { useAuth } from '@entities/auth'
import { useWorkflows } from '@entities/workflow'
import { describeCron } from '@shared/lib/cron'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SButton, STable } from '@shared/ui'
import { computed } from 'vue'

const auth = useAuth()
const canEdit = computed(() => {
  const role = auth.workspaces.value.find(w => w.workspaceId === auth.currentWorkspaceId.value)?.role
  return role === 'editor' || role === 'admin'
})

const { data: workflows, isLoading, error } = useWorkflows()

const columns: Column[] = [
  { key: 'name', label: 'Name' },
  { key: 'steps', label: 'Steps' },
  { key: 'schedule', label: 'Schedule' },
  { key: 'next', label: 'Next run' },
]

function scheduleLabel(schedule: string): string {
  if (!schedule)
    return 'Manual'
  return describeCron(schedule) ?? schedule
}
</script>

<template>
  <PageHeader title="Workflows" description="Run connections in order, each step after the steps it depends on">
    <template #actions>
      <SButton v-if="canEdit" to="/workflows/new" size="sm">
        New workflow
      </SButton>
    </template>
  </PageHeader>

  <SAlert v-if="error" variant="danger" class="mb-4">
    {{ getErrorMessage(error) }}
  </SAlert>

  <STable
    :columns="columns"
    :data="workflows"
    :loading="isLoading"
    empty-text="No workflows yet"
    empty-description="A workflow runs a set of connections as a graph of steps on one schedule."
  >
    <template #cell-name="{ row }">
      <RouterLink
        :to="{ name: 'workflow-detail', params: { id: row.id } }"
        class="text-sm font-medium text-primary hover:underline"
      >
        {{ row.name }}
      </RouterLink>
    </template>
    <template #cell-steps="{ row }">
      <span class="text-sm text-text-secondary">{{ row.steps.length }}</span>
    </template>
    <template #cell-schedule="{ row }">
      <span class="text-sm text-text-secondary">{{ scheduleLabel(row.schedule) }}</span>
    </template>
    <template #cell-next="{ row }">
      <span class="text-sm text-text-secondary">{{ row.nextScheduledAt?.toLocaleString() ?? '-' }}</span>
    </template>
  </STable>
</template>
