<script setup lang="ts">
import type { Column } from '@shared/ui'
import { useManagedConnectors } from '@entities/connector'
import { useConnectorTaskResult } from '@entities/connector-task'
import { useDeleteDestination, useDestinations, useTestDestinationConnection } from '@entities/destination'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SButton, SConfirmDialog, SEmptyState, SPagination, STable, useToast } from '@shared/ui'
import { Plus, Trash2, Zap } from 'lucide-vue-next'
import { ref, watch } from 'vue'

const currentPage = ref(1)
const PAGE_SIZE = 20
const { data, isLoading: loading, error } = useDestinations({ page: currentPage, pageSize: PAGE_SIZE })
const deleteDestinationMutation = useDeleteDestination()
const testConnectionMutation = useTestDestinationConnection()
const { data: connectors } = useManagedConnectors()
const toast = useToast()

const testTaskId = ref<string | null>(null)
const { data: testTaskResult } = useConnectorTaskResult(testTaskId)

watch(testTaskResult, (task) => {
  if (!task)
    return
  if (task.status === 'completed' && task.checkResult) {
    if (task.checkResult.success) {
      toast.success('Connection successful!')
    }
    else {
      toast.error(`Connection failed: ${task.checkResult.message}`)
    }
    testTaskId.value = null
  }
  else if (task.status === 'failed') {
    toast.error(`Connection failed: ${task.errorMessage}`)
    testTaskId.value = null
  }
})

function connectorLabel(managedConnectorId: string): string {
  const c = connectors.value?.find(mc => mc.id === managedConnectorId)
  return c ? `${c.dockerImage}:${c.dockerTag}` : managedConnectorId
}

const confirmDelete = ref<{ open: boolean, id: string, name: string }>({ open: false, id: '', name: '' })

async function testConnection(id: string) {
  try {
    const result = await testConnectionMutation.mutateAsync({ id })
    testTaskId.value = result.taskId
    toast.success('Testing connection...')
  }
  catch (e: unknown) {
    toast.error(`Error: ${getErrorMessage(e)}`)
  }
}

function requestDelete(id: string, name: string) {
  confirmDelete.value = { open: true, id, name }
}

async function doDelete() {
  const id = confirmDelete.value.id
  confirmDelete.value.open = false
  try {
    await deleteDestinationMutation.mutateAsync(id)
    toast.success('Destination deleted')
  }
  catch (e: unknown) {
    toast.error(`Error: ${getErrorMessage(e)}`)
  }
}

const columns: Column[] = [
  { key: 'name', label: 'Name' },
  { key: 'connector', label: 'Connector' },
  { key: 'createdAt', label: 'Created' },
  { key: 'actions', label: 'Actions', align: 'right' },
]
</script>

<template>
  <PageHeader title="Destinations" description="Manage your data destination connections">
    <template v-if="data?.items?.length" #actions>
      <SButton to="/destinations/new">
        <Plus class="w-4 h-4" /> Add Destination
      </SButton>
    </template>
  </PageHeader>

  <SAlert v-if="error" variant="danger" class="mb-4">
    {{ error.message }}
  </SAlert>

  <STable :columns="columns" :data="data?.items" :loading="loading" empty-text="No destinations configured">
    <template #empty>
      <SEmptyState title="No destinations configured" description="Add a destination to receive your synced data">
        <SButton to="/destinations/new" size="sm">
          Add Destination
        </SButton>
      </SEmptyState>
    </template>
    <template #cell-name="{ row }">
      <RouterLink :to="`/destinations/${row.id}`" class="text-sm font-medium text-primary hover:underline hover:text-primary-hover">
        {{ row.name }}
      </RouterLink>
    </template>
    <template #cell-connector="{ row }">
      <span class="text-sm text-text-secondary">{{ connectorLabel(row.managedConnectorId) }}</span>
    </template>
    <template #cell-createdAt="{ row }">
      <span class="text-sm text-text-secondary">{{ row.createdAt?.toLocaleDateString() ?? '-' }}</span>
    </template>
    <template #cell-actions="{ row }">
      <button class="p-1.5 text-text-muted hover:text-primary transition-colors" title="Test connection" @click="testConnection(row.id)">
        <Zap class="w-4 h-4" />
      </button>
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
    title="Delete destination"
    :message="`Delete &quot;${confirmDelete.name}&quot;? This cannot be undone.`"
    confirm-text="Delete"
    :loading="deleteDestinationMutation.isPending.value"
    @confirm="doDelete"
    @cancel="confirmDelete.open = false"
  />
</template>
