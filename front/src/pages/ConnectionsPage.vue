<script setup lang="ts">
import type { Column } from '@shared/ui'
import { useAuth } from '@entities/auth'
import { useConnections } from '@entities/connection'
import * as connectionApi from '@entities/connection/api'
import { BulkActionToolbar, useBulkConnectionActions, useConnectionSelection } from '@features/connection-bulk-ops'
import SchemaDriftBadge from '@features/schema-changes/SchemaDriftBadge.vue'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { statusVariant } from '@shared/lib/format'
import { connectionKeys } from '@shared/lib/queryKeys'
import { PageHeader, SAlert, SBadge, SButton, SConfirmDialog, SEmptyState, SPagination, STable, useToast } from '@shared/ui'
import { useQueryClient } from '@tanstack/vue-query'
import { Pause, Play, Plus, Trash2 } from 'lucide-vue-next'
import { computed, ref } from 'vue'

const currentPage = ref(1)
const PAGE_SIZE = 20
const { data, isLoading: loading, error } = useConnections({ page: currentPage, pageSize: PAGE_SIZE })
const qc = useQueryClient()
const { currentWorkspaceId } = useAuth()
const toast = useToast()

const selection = useConnectionSelection()
const bulk = useBulkConnectionActions()

const pageIds = computed(() => data.value?.items.map(c => c.id) ?? [])
const allOnPageSelected = computed(() => selection.allSelected(pageIds.value))

type ConfirmAction = 'delete' | 'disableAll' | null
const confirmAction = ref<ConfirmAction>(null)
const confirmDeleteSingle = ref<{ open: boolean, id: string, name: string }>({ open: false, id: '', name: '' })

function invalidateAll() {
  qc.invalidateQueries({ queryKey: connectionKeys.all(currentWorkspaceId.value ?? '') })
}

async function requestDeleteSingle(id: string, name: string) {
  confirmDeleteSingle.value = { open: true, id, name }
}

async function doDeleteSingle() {
  const id = confirmDeleteSingle.value.id
  confirmDeleteSingle.value.open = false
  try {
    await connectionApi.deleteConnection(id)
    invalidateAll()
    toast.success('Connection deleted')
  }
  catch (e: unknown) {
    toast.error(`Error: ${getErrorMessage(e)}`)
  }
}

async function runBulk(label: { verb: string, past: string }, op: (id: string) => Promise<unknown>, ids: string[]) {
  if (ids.length === 0)
    return
  const result = await bulk.run(ids, op)
  invalidateAll()
  if (result.failed === 0)
    toast.success(`${label.past} ${result.succeeded} connection${result.succeeded === 1 ? '' : 's'}`)
  else
    toast.error(`${label.past} ${result.succeeded}/${result.total}; ${result.failed} failed`)
  selection.clear()
}

function bulkEnable() {
  runBulk({ verb: 'Enable', past: 'Enabled' }, connectionApi.enableConnection, selection.ids())
}

function bulkDisable() {
  runBulk({ verb: 'Disable', past: 'Disabled' }, connectionApi.disableConnection, selection.ids())
}

function bulkDeleteConfirm() {
  confirmAction.value = 'delete'
}

async function bulkDeleteRun() {
  confirmAction.value = null
  runBulk({ verb: 'Delete', past: 'Deleted' }, connectionApi.deleteConnection, selection.ids())
}

async function fetchAllConnectionIds(): Promise<string[]> {
  const res = await connectionApi.listConnections({})
  return res.items.map(c => c.id)
}

async function pauseAll() {
  confirmAction.value = 'disableAll'
}

async function pauseAllRun() {
  confirmAction.value = null
  const ids = await fetchAllConnectionIds()
  runBulk({ verb: 'Pause', past: 'Paused' }, connectionApi.disableConnection, ids)
}

async function resumeAll() {
  const ids = await fetchAllConnectionIds()
  runBulk({ verb: 'Resume', past: 'Resumed' }, connectionApi.enableConnection, ids)
}

const columns: Column[] = [
  { key: 'select', label: '', width: '40px' },
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
      <SButton variant="secondary" :disabled="bulk.running.value" title="Disable every connection — future runs won't start. In-flight runs continue." @click="pauseAll">
        <Pause class="w-4 h-4" /> Pause all
      </SButton>
      <SButton variant="secondary" :disabled="bulk.running.value" title="Enable every connection" @click="resumeAll">
        <Play class="w-4 h-4" /> Resume all
      </SButton>
      <SButton to="/connections/new">
        <Plus class="w-4 h-4" /> New Connection
      </SButton>
    </template>
  </PageHeader>

  <SAlert v-if="error" variant="danger" class="mb-4">
    {{ error.message }}
  </SAlert>

  <BulkActionToolbar
    v-if="!selection.isEmpty.value"
    :count="selection.count.value"
    :running="bulk.running.value"
    @enable="bulkEnable"
    @disable="bulkDisable"
    @delete="bulkDeleteConfirm"
    @clear="selection.clear()"
  />

  <SAlert v-if="bulk.running.value" variant="info" class="mb-3">
    Working through {{ bulk.progress.value.completed }} / {{ bulk.progress.value.total }}…
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
    <template #header-select>
      <input
        type="checkbox"
        class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        :checked="allOnPageSelected"
        :disabled="bulk.running.value"
        :aria-label="allOnPageSelected ? 'Deselect all on page' : 'Select all on page'"
        @change="selection.toggleAll(pageIds)"
      >
    </template>
    <template #cell-select="{ row }">
      <input
        type="checkbox"
        class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        :checked="selection.isSelected(row.id)"
        :disabled="bulk.running.value"
        :aria-label="`Select ${row.name}`"
        @change="selection.toggle(row.id)"
      >
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
      <button class="p-1.5 text-text-muted hover:text-danger transition-colors" title="Delete" @click="requestDeleteSingle(row.id, row.name)">
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
    :open="confirmDeleteSingle.open"
    title="Delete connection"
    :message="`Delete &quot;${confirmDeleteSingle.name}&quot;? This cannot be undone.`"
    confirm-text="Delete"
    @confirm="doDeleteSingle"
    @cancel="confirmDeleteSingle.open = false"
  />

  <SConfirmDialog
    :open="confirmAction === 'delete'"
    title="Delete selected connections"
    :message="`Delete ${selection.count.value} connection${selection.count.value === 1 ? '' : 's'}? This cannot be undone.`"
    confirm-text="Delete all"
    :loading="bulk.running.value"
    @confirm="bulkDeleteRun"
    @cancel="confirmAction = null"
  />

  <SConfirmDialog
    :open="confirmAction === 'disableAll'"
    title="Pause every connection"
    message="Future scheduled syncs will not start. In-flight syncs continue until they finish."
    confirm-text="Pause all"
    :loading="bulk.running.value"
    @confirm="pauseAllRun"
    @cancel="confirmAction = null"
  />
</template>
