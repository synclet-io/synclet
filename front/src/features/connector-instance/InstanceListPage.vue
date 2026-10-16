<script setup lang="ts">
import type { Column } from '@shared/ui'
import type { InstanceKind } from './instanceKind'
import { useManagedConnectors } from '@entities/connector'
import { useConnectorTaskResult } from '@entities/connector-task'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SButton, SEmptyState, SPagination, STable, useToast } from '@shared/ui'
import { Plus, Trash2, Zap } from 'lucide-vue-next'
import { ref, watch } from 'vue'
import DeleteInstanceDialog from './DeleteInstanceDialog.vue'
import { INSTANCE_KINDS, useDeleteInstance, useInstances, useTestInstanceConnection } from './instanceKind'

const props = defineProps<{
  kind: InstanceKind
}>()

const meta = INSTANCE_KINDS[props.kind]

const currentPage = ref(1)
const PAGE_SIZE = 20
const { data, isLoading: loading, error } = useInstances(props.kind, { page: currentPage, pageSize: PAGE_SIZE })
const deleteMutation = useDeleteInstance(props.kind)
const testConnectionMutation = useTestInstanceConnection(props.kind)
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
const deleteError = ref('')

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
  deleteError.value = ''
  confirmDelete.value = { open: true, id, name }
}

async function doDelete() {
  deleteError.value = ''
  try {
    await deleteMutation.mutateAsync(confirmDelete.value.id)
  }
  catch (e: unknown) {
    deleteError.value = getErrorMessage(e) || `Failed to delete ${meta.noun}`
    return
  }
  confirmDelete.value.open = false
  toast.success(`${meta.title} deleted`)
}

const columns: Column[] = [
  { key: 'name', label: 'Name' },
  { key: 'connector', label: 'Connector' },
  { key: 'createdAt', label: 'Created' },
  { key: 'actions', label: 'Actions', align: 'right' },
]
</script>

<template>
  <PageHeader :title="meta.pluralTitle" :description="meta.listDescription">
    <template v-if="data?.items?.length" #actions>
      <SButton :to="{ name: meta.routes.create }">
        <Plus class="w-4 h-4" /> Add {{ meta.title }}
      </SButton>
    </template>
  </PageHeader>

  <SAlert v-if="error" variant="danger" class="mb-4">
    {{ getErrorMessage(error) }}
  </SAlert>

  <STable :columns="columns" :data="data?.items" :loading="loading" :empty-text="`No ${meta.noun}s configured`">
    <template #empty>
      <SEmptyState :title="`No ${meta.noun}s yet`" :description="meta.emptyDescription">
        <div class="flex gap-2 flex-wrap justify-center">
          <SButton :to="{ name: 'settings-connectors' }" variant="secondary" size="sm">
            Manage connectors
          </SButton>
          <SButton :to="{ name: meta.routes.create }" size="sm">
            <Plus class="w-4 h-4" /> Add {{ meta.noun }}
          </SButton>
        </div>
      </SEmptyState>
    </template>
    <template #cell-name="{ row }">
      <RouterLink :to="{ name: meta.routes.detail, params: { id: row.id } }" class="text-sm font-medium text-primary hover:underline hover:text-primary-hover">
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

  <DeleteInstanceDialog
    :open="confirmDelete.open"
    :kind="kind"
    :name="confirmDelete.name"
    :loading="deleteMutation.isPending.value"
    :error="deleteError"
    @confirm="doDelete"
    @cancel="confirmDelete.open = false"
  />
</template>
