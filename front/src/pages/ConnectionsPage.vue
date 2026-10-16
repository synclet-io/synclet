<script setup lang="ts">
import type { Column } from '@shared/ui'
import { useAuth } from '@entities/auth'
import { useConnections } from '@entities/connection'
import * as connectionApi from '@entities/connection/api'
import { useDestinations } from '@entities/destination'
import { useSources } from '@entities/source'
import { BulkActionToolbar, useBulkConnectionActions, useConnectionSelection } from '@features/connection-bulk-ops'
import { buildConnectionListQuery, parseConnectionListQuery } from '@features/connection-filters/listQuery'
import SchemaDriftBadge from '@features/schema-changes/SchemaDriftBadge.vue'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { statusVariant } from '@shared/lib/format'
import { connectionKeys } from '@shared/lib/queryKeys'
import { PageHeader, SAlert, SBadge, SButton, SCheckbox, SConfirmDialog, SEmptyState, SPagination, SSelect, STable, useToast } from '@shared/ui'
import { useQueryClient } from '@tanstack/vue-query'
import { Copy, FilterX, Pause, Play, Plus, Trash2 } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'

const PAGE_SIZE = 20
const route = useRoute()
const router = useRouter()

const listQuery = computed(() => parseConnectionListQuery(route.query))
const currentPage = computed(() => listQuery.value.page)
const sourceId = computed(() => listQuery.value.sourceId)
const destinationId = computed(() => listQuery.value.destinationId)
const hasFilters = computed(() => !!sourceId.value || !!destinationId.value)

function updateQuery(patch: Parameters<typeof buildConnectionListQuery>[1]) {
  router.replace({ query: buildConnectionListQuery(route.query, patch) })
}

function clearFilters() {
  updateQuery({ sourceId: '', destinationId: '' })
}

const { data, isLoading: loading, error } = useConnections({
  page: currentPage,
  pageSize: PAGE_SIZE,
  includePendingSchemaChanges: true,
  sourceId,
  destinationId,
})
const { data: sources } = useSources()
const { data: destinations } = useDestinations()

const sourceOptions = computed(() => [
  { label: 'All sources', value: '' },
  ...(sources.value?.items ?? []).map(s => ({ label: s.name, value: s.id })),
])
const destinationOptions = computed(() => [
  { label: 'All destinations', value: '' },
  ...(destinations.value?.items ?? []).map(d => ({ label: d.name, value: d.id })),
])

const showFilters = computed(() => hasFilters.value || (data.value?.total ?? 0) > 0)
const qc = useQueryClient()
const { currentWorkspaceId } = useAuth()
const toast = useToast()

const selection = useConnectionSelection()
const bulk = useBulkConnectionActions()

watch([sourceId, destinationId], () => selection.clear())

const pageIds = computed(() => data.value?.items.map(c => c.id) ?? [])
const allOnPageSelected = computed(() => selection.allSelected(pageIds.value))
const someOnPageSelected = computed(() => !allOnPageSelected.value && pageIds.value.some(id => selection.isSelected(id)))

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
  { key: 'source', label: 'Source' },
  { key: 'destination', label: 'Destination' },
  { key: 'status', label: 'Status' },
  { key: 'schedule', label: 'Schedule' },
  { key: 'createdAt', label: 'Created' },
  { key: 'actions', label: 'Actions', align: 'right' },
]
</script>

<template>
  <PageHeader title="Connections" description="Source-to-destination data pipelines">
    <template v-if="showFilters" #actions>
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
    {{ getErrorMessage(error) }}
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

  <div v-if="showFilters" class="flex flex-wrap items-end gap-3 mb-4">
    <SSelect
      label="Source"
      :options="sourceOptions"
      :model-value="sourceId"
      class="w-56"
      @update:model-value="updateQuery({ sourceId: String($event) })"
    />
    <SSelect
      label="Destination"
      :options="destinationOptions"
      :model-value="destinationId"
      class="w-56"
      @update:model-value="updateQuery({ destinationId: String($event) })"
    />
    <SButton v-if="hasFilters" variant="ghost" @click="clearFilters">
      <FilterX class="w-4 h-4" /> Clear filters
    </SButton>
  </div>

  <STable :columns="columns" :data="data?.items" :loading="loading" empty-text="No connections configured">
    <template #empty>
      <SEmptyState
        v-if="hasFilters"
        :icon="FilterX"
        title="No connections match these filters"
        description="Filters are hiding all connections. Pick another source or destination, or clear the filters."
      >
        <SButton variant="secondary" size="sm" @click="clearFilters">
          <FilterX class="w-4 h-4" /> Clear filters
        </SButton>
      </SEmptyState>
      <SEmptyState
        v-else
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
      <SCheckbox
        :model-value="allOnPageSelected"
        :indeterminate="someOnPageSelected"
        :disabled="bulk.running.value"
        :aria-label="allOnPageSelected ? 'Deselect all on page' : 'Select all on page'"
        @update:model-value="selection.toggleAll(pageIds)"
      />
    </template>
    <template #cell-select="{ row }">
      <SCheckbox
        :model-value="selection.isSelected(row.id)"
        :disabled="bulk.running.value"
        :aria-label="`Select ${row.name}`"
        @update:model-value="selection.toggle(row.id)"
      />
    </template>
    <template #cell-name="{ row }">
      <div class="flex items-center gap-2">
        <RouterLink :to="`/connections/${row.id}`" class="text-sm font-medium text-primary hover:underline hover:text-primary-hover">
          {{ row.name }}
        </RouterLink>
        <SchemaDriftBadge :connection-id="row.id" :count="row.pendingSchemaChanges" />
      </div>
    </template>
    <template #cell-source="{ row }">
      <div class="min-w-0">
        <RouterLink :to="{ name: 'source-detail', params: { id: row.sourceId } }" class="block text-sm text-primary hover:underline hover:text-primary-hover truncate">
          {{ row.sourceName || row.sourceId }}
        </RouterLink>
        <span v-if="row.sourceConnectorName" class="block text-xs text-text-muted truncate">{{ row.sourceConnectorName }}</span>
      </div>
    </template>
    <template #cell-destination="{ row }">
      <div class="min-w-0">
        <RouterLink :to="{ name: 'destination-detail', params: { id: row.destinationId } }" class="block text-sm text-primary hover:underline hover:text-primary-hover truncate">
          {{ row.destinationName || row.destinationId }}
        </RouterLink>
        <span v-if="row.destinationConnectorName" class="block text-xs text-text-muted truncate">{{ row.destinationConnectorName }}</span>
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
      <div class="flex items-center justify-end gap-1">
        <SButton
          :to="`/connections/new?cloneFrom=${row.id}`"
          variant="ghost"
          size="icon"
          title="Duplicate"
          :aria-label="`Duplicate ${row.name}`"
        >
          <Copy class="w-4 h-4" />
        </SButton>
        <SButton variant="ghost" size="icon" title="Delete" :aria-label="`Delete ${row.name}`" @click="requestDeleteSingle(row.id, row.name)">
          <Trash2 class="w-4 h-4" />
        </SButton>
      </div>
    </template>
  </STable>

  <SPagination
    :total="data?.total ?? 0"
    :page-size="PAGE_SIZE"
    :current-page="currentPage"
    class="mt-4"
    @page-change="updateQuery({ page: $event })"
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
