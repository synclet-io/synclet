<script setup lang="ts">
import type { Column } from '@shared/ui'
import { useConnections, useDeleteConnection } from '@entities/connection'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { statusVariant } from '@shared/lib/format'
import { PageHeader, SBadge, SButton, SConfirmDialog, SEmptyState, SPagination, STable, useToast } from '@shared/ui'
import { Plus, Trash2 } from 'lucide-vue-next'
import { ref } from 'vue'

const currentPage = ref(1)
const PAGE_SIZE = 20
const { data, isLoading: loading, error } = useConnections({ page: currentPage, pageSize: PAGE_SIZE })
const deleteConnectionMutation = useDeleteConnection()
const toast = useToast()

const confirmDelete = ref<{ open: boolean, id: string, name: string }>({ open: false, id: '', name: '' })

function requestDelete(id: string, name: string) {
  confirmDelete.value = { open: true, id, name }
}

async function doDelete() {
  const id = confirmDelete.value.id
  confirmDelete.value.open = false
  try {
    await deleteConnectionMutation.mutateAsync(id)
    toast.success('Connection deleted')
  }
  catch (e: unknown) {
    toast.error(`Error: ${getErrorMessage(e)}`)
  }
}

const columns: Column[] = [
  { key: 'name', label: 'Name' },
  { key: 'status', label: 'Status' },
  { key: 'schedule', label: 'Schedule' },
  { key: 'createdAt', label: 'Created' },
  { key: 'actions', label: 'Actions', align: 'right' },
]
</script>

<template>
  <PageHeader title="Connections" description="Source-to-destination data pipelines">
    <template v-if="data?.items?.length" #actions>
      <SButton to="/connections/new">
        <Plus class="w-4 h-4" /> New Connection
      </SButton>
    </template>
  </PageHeader>

  <SAlert v-if="error" variant="danger" class="mb-4">
    {{ error.message }}
  </SAlert>

  <STable :columns="columns" :data="data?.items" :loading="loading" empty-text="No connections configured">
    <template #empty>
      <SEmptyState title="No connections configured" description="Create a connection to sync data between a source and destination">
        <SButton to="/connections/new" size="sm">
          Create Connection
        </SButton>
      </SEmptyState>
    </template>
    <template #cell-name="{ row }">
      <RouterLink :to="`/connections/${row.id}`" class="text-sm font-medium text-primary hover:underline hover:text-primary-hover">
        {{ row.name }}
      </RouterLink>
    </template>
    <template #cell-status="{ row }">
      <SBadge :variant="statusVariant(row.status)" dot>
        {{ row.status }}
      </SBadge>
    </template>
    <template #cell-schedule="{ row }">
      <span class="text-sm text-text-secondary">{{ row.schedule || 'Manual' }}</span>
    </template>
    <template #cell-createdAt="{ row }">
      <span class="text-sm text-text-secondary">{{ row.createdAt?.toLocaleDateString() ?? '-' }}</span>
    </template>
    <template #cell-actions="{ row }">
      <button class="p-1.5 text-text-muted hover:text-danger transition-colors" title="Delete" @click="requestDelete(row.id, row.name)">
        <Trash2 class="w-4 h-4" />
      </button>
    </template>
  </STable>

  <SPagination
    :total="data?.total ?? 0"
    :page-size="PAGE_SIZE"
    :current-page="currentPage"
    class="mt-4"
    @page-change="currentPage = $event"
  />

  <SConfirmDialog
    :open="confirmDelete.open"
    title="Delete connection"
    :message="`Delete &quot;${confirmDelete.name}&quot;? This cannot be undone.`"
    confirm-text="Delete"
    :loading="deleteConnectionMutation.isPending.value"
    @confirm="doDelete"
    @cancel="confirmDelete.open = false"
  />
</template>
