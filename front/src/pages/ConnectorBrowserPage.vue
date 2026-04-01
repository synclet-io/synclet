<script setup lang="ts">
import type { ConnectorType, ManagedConnector } from '@entities/connector'
import type { Repository } from '@entities/repository'
import type { Column, Tab } from '@shared/ui'
import { useAddConnector, useBatchUpdateConnectors, useDeleteManagedConnector, useManagedConnectors, useUpdateManagedConnector } from '@entities/connector'
import { useAddRepository, useDeleteRepository, useRepositories, useSyncRepository } from '@entities/repository'
import { BreakingChangeDialog } from '@features/connector-update'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SBadge, SButton, SConfirmDialog, SEmptyState, SInput, SModal, SPagination, SSelect, SSkeleton, STable, STabs, useToast } from '@shared/ui'
import { ArrowUpCircle, Container, Globe, RefreshCw, Trash2 } from 'lucide-vue-next'

import { refDebounced } from '@vueuse/core'
import { computed, ref, watch } from 'vue'

const toast = useToast()

// Filter state
const repositoryFilter = ref('')
const searchQuery = ref('')
const debouncedSearch = refDebounced(searchQuery, 300)

const mappedRepositoryId = computed(() => {
  if (repositoryFilter.value === '') return undefined
  if (repositoryFilter.value === 'custom') return null
  return repositoryFilter.value
})

// Installed connectors
const { data: managed, isLoading: managedLoading } = useManagedConnectors({
  repositoryId: mappedRepositoryId,
  search: debouncedSearch,
})
const deleteConnector = useDeleteManagedConnector()
const updateConnector = useUpdateManagedConnector()
const batchUpdate = useBatchUpdateConnectors()

const activeTab = ref<string>('source')
const tabs: Tab[] = [
  { name: 'Sources', value: 'source' },
  { name: 'Destinations', value: 'destination' },
  { name: 'Repositories', value: 'repositories' },
]

const connectorTab = computed(() => activeTab.value as ConnectorType)

const currentPage = ref(1)
const PAGE_SIZE = 20

// Reset page and search when tab changes
watch(connectorTab, () => {
  currentPage.value = 1
  searchQuery.value = ''
})

const filteredManaged = computed(() =>
  (managed.value ?? []).filter(c => c.connectorType === connectorTab.value),
)

const pagedManaged = computed(() => {
  const start = (currentPage.value - 1) * PAGE_SIZE
  return filteredManaged.value.slice(start, start + PAGE_SIZE)
})

const columns: Column[] = [
  { key: 'name', label: 'Name' },
  { key: 'dockerImage', label: 'Docker Image' },
  { key: 'dockerTag', label: 'Tag', width: '100px' },
  { key: 'status', label: 'Status', width: '100px' },
  { key: 'update', label: 'Update', width: '160px' },
  { key: 'actions', label: '', align: 'right', width: '100px' },
]

function statusVariant(status: string) {
  const map: Record<string, 'success' | 'warning' | 'info' | 'gray'> = {
    ready: 'success',
    pulling: 'info',
    error: 'warning',
  }
  return map[status?.toLowerCase()] || 'gray'
}

// Delete confirmation
const deleteTarget = ref<ManagedConnector | null>(null)
const deleteLoading = ref(false)

async function handleDelete() {
  if (!deleteTarget.value)
    return
  deleteLoading.value = true
  try {
    await deleteConnector.mutateAsync(deleteTarget.value.id)
    toast.success(`${deleteTarget.value.name} deleted`)
    deleteTarget.value = null
  }
  catch (e: unknown) {
    toast.error(getErrorMessage(e) || 'Failed to delete connector')
  }
  finally {
    deleteLoading.value = false
  }
}

// Breaking change dialog state
const breakingDialogTarget = ref<ManagedConnector | null>(null)
const breakingDialogLoading = ref(false)
const breakingQueue = ref<ManagedConnector[]>([])
const updateAllLoading = ref(false)

const connectorsWithUpdates = computed(() =>
  (managed.value ?? []).filter(c => c.updateInfo?.hasUpdate),
)

function handleUpdateClick(connector: ManagedConnector) {
  if (!connector.updateInfo?.hasUpdate)
    return

  // No breaking changes -- update directly
  if (!connector.updateInfo.breakingChanges?.length) {
    updateConnector.mutateAsync(connector.id)
      .then(() => toast.success(`${connector.name} updated`))
      .catch((e: unknown) => toast.error(getErrorMessage(e) || 'Failed to update connector'))
    return
  }

  // Has breaking changes -- show dialog
  breakingDialogTarget.value = connector
}

async function handleBreakingConfirm() {
  if (!breakingDialogTarget.value)
    return
  breakingDialogLoading.value = true
  try {
    await updateConnector.mutateAsync(breakingDialogTarget.value.id)
    toast.success(`${breakingDialogTarget.value.name} updated`)
  }
  catch (e: unknown) {
    toast.error(getErrorMessage(e) || 'Failed to update connector')
  }
  finally {
    breakingDialogLoading.value = false
    advanceBreakingQueue()
  }
}

function handleBreakingCancel() {
  advanceBreakingQueue()
}

function advanceBreakingQueue() {
  if (breakingQueue.value.length > 0) {
    breakingDialogTarget.value = breakingQueue.value.shift()!
  }
  else {
    breakingDialogTarget.value = null
  }
}

async function handleUpdateAll() {
  const withUpdates = connectorsWithUpdates.value
  if (withUpdates.length === 0)
    return

  const safeUpdates = withUpdates.filter(c => !c.updateInfo?.breakingChanges?.length)
  const breakingUpdates = withUpdates.filter(c => c.updateInfo?.breakingChanges?.length)

  updateAllLoading.value = true
  try {
    // Batch update safe connectors immediately
    if (safeUpdates.length > 0) {
      const result = await batchUpdate.mutateAsync(safeUpdates.map(c => c.id))
      toast.success(`${result.updatedCount} connector${result.updatedCount !== 1 ? 's' : ''} updated successfully`)
    }

    // Queue breaking-change connectors for sequential confirmation
    if (breakingUpdates.length > 0) {
      breakingQueue.value = [...breakingUpdates]
      breakingDialogTarget.value = breakingQueue.value.shift()!
    }
  }
  catch (e: unknown) {
    toast.error(getErrorMessage(e) || 'Failed to update connectors')
  }
  finally {
    updateAllLoading.value = false
  }
}

// Add connector modal
const showAddModal = ref(false)

// Custom connector form
const customForm = ref<{ dockerImage: string, dockerTag: string, name: string, connectorType: ConnectorType }>({ dockerImage: '', dockerTag: 'latest', name: '', connectorType: 'source' })
const customError = ref('')

function openAddModal() {
  customForm.value = { dockerImage: '', dockerTag: 'latest', name: '', connectorType: connectorTab.value }
  customError.value = ''
  showAddModal.value = true
}

function closeAddModal() {
  showAddModal.value = false
}

// Repository management
const { data: repositories, isLoading: reposLoading } = useRepositories()

const addConnector = useAddConnector()

const customLoading = ref(false)
const connectorTypeOptions = [
  { label: 'Source', value: 'source' },
  { label: 'Destination', value: 'destination' },
]

async function handleAddCustom() {
  customError.value = ''
  customLoading.value = true
  try {
    await addConnector.mutateAsync(customForm.value)
    toast.success(`${customForm.value.name || customForm.value.dockerImage} added`)
    closeAddModal()
  }
  catch (e: unknown) {
    customError.value = getErrorMessage(e) || 'Failed to add connector'
  }
  finally {
    customLoading.value = false
  }
}

// --- Repository management ---
const addRepoMutation = useAddRepository()
const deleteRepoMutation = useDeleteRepository()
const syncRepoMutation = useSyncRepository()

const repoColumns: Column[] = [
  { key: 'name', label: 'Name' },
  { key: 'url', label: 'URL' },
  { key: 'status', label: 'Status', width: '100px' },
  { key: 'connectorCount', label: 'Connectors', width: '100px' },
  { key: 'lastSyncedAt', label: 'Last Synced', width: '140px' },
  { key: 'actions', label: '', align: 'right', width: '160px' },
]

function repoStatusVariant(status: string) {
  const map: Record<string, 'success' | 'warning' | 'info' | 'gray'> = {
    synced: 'success',
    syncing: 'info',
    failed: 'warning',
  }
  return map[status] || 'gray'
}

function formatRelativeTime(dateStr: string | null): string {
  if (!dateStr)
    return 'Never'
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMin = Math.floor(diffMs / 60000)
  if (diffMin < 1)
    return 'Just now'
  if (diffMin < 60)
    return `${diffMin} minute${diffMin > 1 ? 's' : ''} ago`
  const diffHr = Math.floor(diffMin / 60)
  if (diffHr < 24)
    return `${diffHr} hour${diffHr > 1 ? 's' : ''} ago`
  const diffDays = Math.floor(diffHr / 24)
  return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`
}

const showAddRepoModal = ref(false)
const addRepoForm = ref({ name: '', url: '', authHeader: '' })
const addRepoError = ref('')
const addRepoLoading = ref(false)

const WELL_KNOWN_REPOS = [
  {
    label: 'Airbyte OSS',
    value: 'airbyte-oss',
    name: 'Airbyte OSS',
    url: 'https://connectors.airbyte.com/files/registries/v0/oss_registry.json',
  },
]

const PRESET_OPTIONS = [
  ...WELL_KNOWN_REPOS.map(r => ({ label: r.label, value: r.value })),
  { label: 'Custom', value: 'custom' },
]

const selectedPreset = ref<string>(WELL_KNOWN_REPOS[0].value)
const isCustom = computed(() => selectedPreset.value === 'custom')

watch(selectedPreset, (val) => {
  const preset = WELL_KNOWN_REPOS.find(r => r.value === val)
  if (preset) {
    addRepoForm.value.name = preset.name
    addRepoForm.value.url = preset.url
  }
  else {
    addRepoForm.value.name = ''
    addRepoForm.value.url = ''
  }
}, { immediate: true })

function openAddRepoModal() {
  selectedPreset.value = WELL_KNOWN_REPOS[0].value
  addRepoForm.value = { name: '', url: '', authHeader: '' }
  addRepoError.value = ''
  showAddRepoModal.value = true
}

async function handleAddRepo() {
  addRepoError.value = ''
  addRepoLoading.value = true
  try {
    const preset = WELL_KNOWN_REPOS.find(r => r.value === selectedPreset.value)
    const name = isCustom.value ? addRepoForm.value.name : (preset?.name ?? addRepoForm.value.name)
    const url = isCustom.value ? addRepoForm.value.url : (preset?.url ?? addRepoForm.value.url)
    const repo = await addRepoMutation.mutateAsync({
      name,
      url,
      authHeader: addRepoForm.value.authHeader || undefined,
    })
    toast.success(`${repo.name} added`)
    showAddRepoModal.value = false
  }
  catch (e: unknown) {
    addRepoError.value = getErrorMessage(e) || 'Failed to add repository'
  }
  finally {
    addRepoLoading.value = false
  }
}

const deleteRepoTarget = ref<Repository | null>(null)
const deleteRepoLoading = ref(false)

async function handleDeleteRepo() {
  if (!deleteRepoTarget.value)
    return
  deleteRepoLoading.value = true
  try {
    await deleteRepoMutation.mutateAsync(deleteRepoTarget.value.id)
    toast.success(`${deleteRepoTarget.value.name} deleted`)
    deleteRepoTarget.value = null
  }
  catch (e: unknown) {
    toast.error(getErrorMessage(e) || 'Failed to delete repository')
  }
  finally {
    deleteRepoLoading.value = false
  }
}

const deleteRepoMessage = computed(() => {
  if (!deleteRepoTarget.value)
    return ''
  const r = deleteRepoTarget.value
  if (r.connectorCount > 0) {
    return `Are you sure you want to delete ${r.name}? ${r.connectorCount} installed connectors will lose their repository link.`
  }
  return `Are you sure you want to delete ${r.name}? This cannot be undone.`
})

const syncingRepoId = ref<string | null>(null)

async function handleSyncRepo(repo: Repository) {
  syncingRepoId.value = repo.id
  try {
    await syncRepoMutation.mutateAsync(repo.id)
    toast.success(`${repo.name} synced`)
  }
  catch (e: unknown) {
    toast.error(getErrorMessage(e) || 'Failed to sync repository')
  }
  finally {
    syncingRepoId.value = null
  }
}
</script>

<template>
  <STabs v-model="activeTab" :tabs="tabs" variant="pills" class="mb-6" />

  <!-- Repositories tab -->
  <template v-if="activeTab === 'repositories'">
    <div class="flex justify-end mb-4">
      <SButton @click="openAddRepoModal">
        <Globe class="w-4 h-4" /> Add Repository
      </SButton>
    </div>

    <div v-if="reposLoading" class="space-y-3">
      <SSkeleton v-for="i in 4" :key="i" variant="rect" height="48px" />
    </div>

    <STable v-else-if="repositories && repositories.length > 0" :columns="repoColumns" :data="repositories">
      <template #cell-name="{ row }">
        <span class="font-medium text-heading">{{ row.name }}</span>
      </template>
      <template #cell-url="{ row }">
        <span class="text-text-muted truncate max-w-[300px] block">{{ row.url }}</span>
        <span v-if="row.status === 'failed' && row.lastError" class="text-xs text-text-muted block mt-0.5">{{ row.lastError }}</span>
      </template>
      <template #cell-status="{ row }">
        <SBadge :variant="repoStatusVariant(row.status)">
          {{ row.status }}
        </SBadge>
      </template>
      <template #cell-connectorCount="{ row }">
        <span class="text-sm">{{ row.connectorCount }}</span>
      </template>
      <template #cell-lastSyncedAt="{ row }">
        <span class="text-sm text-text-muted">{{ formatRelativeTime(row.lastSyncedAt) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="flex items-center gap-2 justify-end">
          <SButton size="sm" variant="ghost" :loading="syncingRepoId === row.id" @click="handleSyncRepo(row)">
            <RefreshCw class="w-3.5 h-3.5" /> Sync Now
          </SButton>
          <SButton size="sm" variant="ghost" class="text-danger hover:text-danger" @click="deleteRepoTarget = row">
            <Trash2 class="w-3.5 h-3.5" /> Delete
          </SButton>
        </div>
      </template>
    </STable>

    <SEmptyState v-else title="No repositories configured" description="Add a connector repository to browse and install connectors">
      <SButton size="sm" @click="openAddRepoModal">
        <Globe class="w-4 h-4" /> Add Repository
      </SButton>
    </SEmptyState>

    <!-- Add Repository Modal -->
    <SModal :open="showAddRepoModal" title="Add Repository" size="sm" @close="showAddRepoModal = false">
      <form class="space-y-4" @submit.prevent="handleAddRepo">
        <SAlert v-if="addRepoError" variant="danger" dismissible @dismiss="addRepoError = ''">
          {{ addRepoError }}
        </SAlert>
        <SSelect
          v-model="selectedPreset"
          label="Repository Preset"
          :options="PRESET_OPTIONS"
          placeholder="Select a preset"
          required
        />
        <template v-if="isCustom">
          <SInput v-model="addRepoForm.name" label="Name" placeholder="My Registry" required />
          <SInput v-model="addRepoForm.url" label="Registry URL" placeholder="https://example.com/registry.json" required />
        </template>
        <div>
          <SInput v-model="addRepoForm.authHeader" label="Auth Header" placeholder="Bearer your-token" />
          <p class="text-xs text-text-muted mt-1">
            Optional. For private registries, provide the Authorization header value.
          </p>
        </div>
        <div class="flex justify-end gap-3 pt-2">
          <SButton variant="secondary" type="button" @click="showAddRepoModal = false">
            Discard
          </SButton>
          <SButton type="submit" :loading="addRepoLoading">
            Add Repository
          </SButton>
        </div>
      </form>
    </SModal>

    <!-- Delete Repository Confirmation -->
    <SConfirmDialog
      :open="!!deleteRepoTarget"
      title="Delete Repository"
      :message="deleteRepoMessage"
      confirm-text="Delete Repository"
      variant="danger"
      :loading="deleteRepoLoading"
      @close="deleteRepoTarget = null"
      @confirm="handleDeleteRepo"
    />
  </template>

  <!-- Connectors tab (Sources / Destinations) -->
  <template v-else>
    <div class="flex items-end gap-3 mb-4">
      <SSelect
        v-model="repositoryFilter"
        size="sm"
        class="w-48"
        :options="[
          { label: 'All repositories', value: '' },
          ...(repositories ?? []).map(r => ({ label: r.name, value: r.id })),
          { label: 'Custom', value: 'custom' },
        ]"
      />
      <SInput
        v-model="searchQuery"
        class="w-64"
        placeholder="Search connectors..."
      />
      <div class="ml-auto flex gap-2">
        <SButton
          v-if="connectorsWithUpdates.length > 0"
          variant="secondary"
          :loading="updateAllLoading"
          @click="handleUpdateAll"
        >
          <ArrowUpCircle class="w-4 h-4" /> Update All ({{ connectorsWithUpdates.length }})
        </SButton>
        <SButton @click="openAddModal">
          <Container class="w-4 h-4" /> Add Custom
        </SButton>
      </div>
    </div>

    <div v-if="managedLoading" class="space-y-3">
      <SSkeleton v-for="i in 4" :key="i" variant="rect" height="48px" />
    </div>

    <SEmptyState v-else-if="!pagedManaged.length" title="No connectors found" description="Try adjusting your filters or search query." />

    <STable v-else :columns="columns" :data="pagedManaged">
      <template #cell-name="{ row }">
        <span class="font-medium text-heading">{{ row.name }}</span>
      </template>
      <template #cell-dockerImage="{ row }">
        <code class="text-xs text-text-muted">{{ row.dockerImage }}</code>
      </template>
      <template #cell-dockerTag="{ row }">
        <code class="text-xs">{{ row.dockerTag }}</code>
      </template>
      <template #cell-status="{ row }">
        <SBadge :variant="statusVariant(row.status)">
          {{ row.status }}
        </SBadge>
      </template>
      <template #cell-update="{ row }">
        <div v-if="row.updateInfo?.hasUpdate" class="flex items-center gap-2">
          <SBadge variant="info">
            {{ row.updateInfo.availableVersion }}
          </SBadge>
          <SButton
            size="sm" variant="secondary"
            :loading="updateConnector.isPending.value && breakingDialogTarget?.id === row.id"
            @click="handleUpdateClick(row)"
          >
            Update
          </SButton>
        </div>
      </template>
      <template #cell-actions="{ row }">
        <SButton size="sm" variant="ghost" class="text-danger hover:text-danger" @click="deleteTarget = row">
          Delete
        </SButton>
      </template>
      <template #empty>
        <SEmptyState
          :title="`No ${connectorTab}s installed`"
          :description="`Add a ${connectorTab} connector to get started`"
        >
          <SButton size="sm" @click="openAddModal">
            <Container class="w-4 h-4" /> Add Custom
          </SButton>
        </SEmptyState>
      </template>
    </STable>

    <SPagination
      :total="filteredManaged.length"
      :page-size="PAGE_SIZE"
      :current-page="currentPage"
      class="mt-4"
      @page-change="currentPage = $event"
    />
  </template>

  <!-- Delete confirmation -->
  <SConfirmDialog
    :open="!!deleteTarget"
    title="Delete Connector"
    :message="`Are you sure you want to delete ${deleteTarget?.name}? This cannot be undone.`"
    confirm-text="Delete"
    variant="danger"
    :loading="deleteLoading"
    @close="deleteTarget = null"
    @confirm="handleDelete"
  />

  <!-- Add Custom Connector Modal -->
  <SModal :open="showAddModal" title="Add Custom Connector" size="sm" @close="closeAddModal">
    <form class="space-y-4" @submit.prevent="handleAddCustom">
      <SAlert v-if="customError" variant="danger" dismissible @dismiss="customError = ''">
        {{ customError }}
      </SAlert>
      <SInput v-model="customForm.dockerImage" label="Docker Image" placeholder="airbyte/source-postgres" required />
      <SInput v-model="customForm.dockerTag" label="Tag" placeholder="latest" required />
      <SInput v-model="customForm.name" label="Name" placeholder="Postgres Source" required />
      <SSelect v-model="customForm.connectorType" label="Type" :options="connectorTypeOptions" />
      <div class="flex justify-end gap-3 pt-2">
        <SButton variant="secondary" type="button" @click="closeAddModal">
          Cancel
        </SButton>
        <SButton type="submit" :loading="customLoading">
          Add Connector
        </SButton>
      </div>
    </form>
  </SModal>

  <!-- Breaking Change Dialog -->
  <BreakingChangeDialog
    :open="!!breakingDialogTarget"
    :connector-name="breakingDialogTarget?.name ?? ''"
    :current-version="breakingDialogTarget?.dockerTag ?? ''"
    :target-version="breakingDialogTarget?.updateInfo?.availableVersion ?? ''"
    :breaking-changes="breakingDialogTarget?.updateInfo?.breakingChanges ?? []"
    :loading="breakingDialogLoading"
    @confirm="handleBreakingConfirm"
    @cancel="handleBreakingCancel"
  />
</template>
