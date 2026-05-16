<script setup lang="ts">
import type { Column } from '@shared/ui'
import { useConnections, useDeleteConnection } from '@entities/connection'
import SchemaDriftBadge from '@features/schema-changes/SchemaDriftBadge.vue'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { statusVariant } from '@shared/lib/format'
import { PageHeader, SAlert, SBadge, SButton, SConfirmDialog, SEmptyState, SPagination, STable, useToast } from '@shared/ui'
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
      <SEmptyState
        title="No connections yet"
        description="A connection pairs a source with a destination. Configure a source and a destination first, then create a connection to start syncing."
      >
        <div class="flex gap-2 flex-wrap justify-center">
          <SButton to="/sources" variant="secondary" size="sm">
            Manage sources
          </SButton>
          <SButton to="/destinations" variant="secondary" size="sm">
            Manage destinations
          </SButton>
          <SButton to="/connections/new" size="sm">
            <Plus class="w-4 h-4" /> Create connection
          </SButton>
        </div>
      </SEmptyState>
    </template>
    <template #cell-name="{ row }">
      <div class="flex items-center gap-2">
        <RouterLink :to="`/connections/${row.id}`" class="text-sm font-medium text-primary hover:underline hover:text-primary-hover">
          {{ row.name }}
        </RouterLink>
        <SchemaDriftBadge :connection-id="row.id" />
      </div>
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
