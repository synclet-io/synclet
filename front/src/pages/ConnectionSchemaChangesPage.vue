<script setup lang="ts">
import type { SchemaChange, SchemaChangeType } from '@entities/connection'
import type { Column } from '@shared/ui'
import { useConnection, useDiscoverSchema, useSchemaChanges } from '@entities/connection'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SBadge, SButton, SEmptyState, STable, useToast } from '@shared/ui'
import { CheckCircle2, RefreshCw } from 'lucide-vue-next'
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const toast = useToast()
const connectionId = computed(() => route.params.id as string)

const { data: connection } = useConnection(connectionId)
const { data: changes, isLoading, error, refetch } = useSchemaChanges(connectionId)
const discoverMutation = useDiscoverSchema()

const changeCount = computed(() => changes.value?.length ?? 0)

const policyMessage = computed(() => {
  if (!connection.value)
    return ''
  switch (connection.value.schemaChangePolicy) {
    case 'propagate':
      return 'Policy: changes are propagated automatically on the next sync.'
    case 'ignore':
      return 'Policy: changes are ignored — review and update streams manually.'
    case 'pause':
      return 'Policy: the connection is paused when schema changes are detected.'
    default:
      return ''
  }
})

async function refreshFromSource() {
  try {
    await discoverMutation.mutateAsync(connectionId.value)
    await refetch()
    toast.success('Source schema refreshed')
  }
  catch (e: unknown) {
    toast.error(`Refresh failed: ${getErrorMessage(e)}`)
  }
}

function changeVariant(type: SchemaChangeType): 'success' | 'danger' | 'warning' | 'gray' {
  switch (type) {
    case 'added': return 'success'
    case 'removed': return 'danger'
    case 'modified': return 'warning'
    default: return 'gray'
  }
}

function changeLabel(change: SchemaChange): string {
  switch (change.type) {
    case 'added': return 'Added'
    case 'removed': return 'Removed'
    case 'modified': return 'Modified'
    default: return 'Change'
  }
}

const columns: Column[] = [
  { key: 'namespace', label: 'Namespace' },
  { key: 'stream', label: 'Stream' },
  { key: 'type', label: 'Change' },
  { key: 'column', label: 'Column' },
  { key: 'oldType', label: 'Old type' },
  { key: 'newType', label: 'New type' },
]
</script>

<template>
  <div class="space-y-4 mt-4">
    <SAlert v-if="error" variant="danger">
      Failed to load schema changes: {{ error.message }}
    </SAlert>

    <div v-if="changeCount > 0" class="flex items-center justify-between gap-4">
      <SAlert variant="warning" class="flex-1">
        <p class="font-medium">
          {{ changeCount }} pending schema {{ changeCount === 1 ? 'change' : 'changes' }} detected.
        </p>
        <p v-if="policyMessage" class="text-sm mt-1">
          {{ policyMessage }}
        </p>
      </SAlert>
      <SButton variant="secondary" :loading="discoverMutation.isPending.value" @click="refreshFromSource">
        <RefreshCw class="w-4 h-4" :class="{ 'animate-spin': discoverMutation.isPending.value }" />
        Refresh from source
      </SButton>
    </div>

    <STable
      v-if="changeCount > 0"
      :columns="columns"
      :data="changes ?? []"
      :loading="isLoading"
    >
      <template #cell-namespace="{ row }">
        <span class="font-mono text-xs text-text-secondary">{{ row.namespace || '-' }}</span>
      </template>
      <template #cell-stream="{ row }">
        <span class="font-medium text-text-primary">{{ row.streamName }}</span>
      </template>
      <template #cell-type="{ row }">
        <SBadge :variant="changeVariant(row.type)">
          {{ changeLabel(row) }}
        </SBadge>
      </template>
      <template #cell-column="{ row }">
        <span class="font-mono text-xs text-text-primary">{{ row.columnName || '-' }}</span>
      </template>
      <template #cell-oldType="{ row }">
        <span class="font-mono text-xs text-text-secondary">{{ row.oldType ?? '-' }}</span>
      </template>
      <template #cell-newType="{ row }">
        <span class="font-mono text-xs text-text-secondary">{{ row.newType ?? '-' }}</span>
      </template>
    </STable>

    <template v-else-if="!isLoading">
      <SEmptyState
        title="No pending schema changes"
        description="Schema drift is detected on every sync. Use Refresh from source to re-run discovery now."
        :icon="CheckCircle2"
      >
        <template #actions>
          <SButton :loading="discoverMutation.isPending.value" @click="refreshFromSource">
            <RefreshCw class="w-4 h-4" :class="{ 'animate-spin': discoverMutation.isPending.value }" />
            Refresh from source
          </SButton>
        </template>
      </SEmptyState>
    </template>
  </div>
</template>
