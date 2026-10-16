<script setup lang="ts">
import { useAuth } from '@entities/auth'
import { useConnectorVersions, useManagedConnector } from '@entities/connector'
import { useConnectorTaskResult } from '@entities/connector-task'
import { discoverSourceSchema, useDeleteSource, useSource, useSourceCatalog, useUpdateSource } from '@entities/source'
import DeleteInstanceDialog from '@features/connector-instance/DeleteInstanceDialog.vue'
import InstanceConnectionsCard from '@features/connector-instance/InstanceConnectionsCard.vue'
import RuntimeConfigForm from '@features/runtime-config/RuntimeConfigForm.vue'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { sourceKeys } from '@shared/lib/queryKeys'
import { PageHeader, SAlert, SBadge, SButton, SCard, SSkeleton, useToast } from '@shared/ui'
import { useQueryClient } from '@tanstack/vue-query'
import { Pencil, RefreshCw, Trash2 } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const sourceId = computed(() => route.params.id as string)
const { data: source, isLoading, error } = useSource(sourceId)
const updateSource = useUpdateSource()
const deleteSource = useDeleteSource()
const connectorId = computed(() => source.value?.managedConnectorId ?? '')
const { data: connector } = useManagedConnector(connectorId)
const connectorImage = computed(() => connector.value?.dockerImage ?? '')
const connectorLabel = computed(() => connector.value ? `${connector.value.dockerImage}:${connector.value.dockerTag}` : '')
const { data: versionInfo } = useConnectorVersions(connectorImage)
const hasUpdate = computed(() => versionInfo.value?.latestVersion && connector.value?.dockerTag && versionInfo.value.latestVersion !== connector.value.dockerTag)
const toast = useToast()

const { data: catalog, isLoading: catalogLoading } = useSourceCatalog(sourceId)
const discoverTaskId = ref<string | null>(null)
const { data: discoverTaskResult } = useConnectorTaskResult(discoverTaskId)
const qc = useQueryClient()
const { currentWorkspaceId } = useAuth()

const catalogStatus = computed(() => {
  if (discoverTaskId.value)
    return 'discovering'
  if (catalogLoading.value)
    return 'loading'
  if (catalog.value && catalog.value.version > 0)
    return 'discovered'
  return 'none'
})

const catalogBadgeVariant = computed(() => {
  switch (catalogStatus.value) {
    case 'discovering': return 'info'
    case 'discovered': return 'success'
    case 'none': return 'gray'
    default: return 'gray'
  }
})

const catalogStatusLabel = computed(() => {
  switch (catalogStatus.value) {
    case 'discovering': return 'Discovering...'
    case 'discovered': return 'Catalog ready'
    case 'none': return 'Not discovered'
    default: return 'Not discovered'
  }
})

async function handleRefreshCatalog() {
  if (!source.value)
    return
  try {
    const result = await discoverSourceSchema(source.value.id)
    discoverTaskId.value = result.taskId
    toast.success('Discovering catalog...')
  }
  catch (e: unknown) {
    toast.error(`Failed to start discovery: ${getErrorMessage(e)}`)
  }
}

watch(discoverTaskResult, (task) => {
  if (!task)
    return
  if (task.status === 'completed') {
    toast.success('Catalog updated successfully')
    discoverTaskId.value = null
    qc.invalidateQueries({ queryKey: sourceKeys.all(currentWorkspaceId.value ?? '') })
  }
  else if (task.status === 'failed') {
    toast.error(`Catalog discovery failed: ${task.errorMessage || 'Unknown error'}`)
    discoverTaskId.value = null
  }
})

async function handleSaveRuntimeConfig(configJson: string | null) {
  if (!source.value)
    return
  try {
    await updateSource.mutateAsync({
      id: source.value.id,
      runtimeConfig: configJson,
    })
    toast.success('Runtime configuration saved')
  }
  catch (e: unknown) {
    toast.error(`Failed to save runtime configuration: ${getErrorMessage(e)}`)
  }
}

async function handleResetRuntimeConfig() {
  if (!source.value)
    return
  try {
    await updateSource.mutateAsync({
      id: source.value.id,
      runtimeConfig: null,
    })
    toast.success('Runtime configuration reset to defaults')
  }
  catch (e: unknown) {
    toast.error(`Failed to reset runtime configuration: ${getErrorMessage(e)}`)
  }
}

const deleteOpen = ref(false)
const deleteError = ref('')

function requestDelete() {
  deleteError.value = ''
  deleteOpen.value = true
}

async function handleDelete() {
  if (!source.value)
    return
  deleteError.value = ''
  try {
    await deleteSource.mutateAsync(source.value.id)
  }
  catch (e: unknown) {
    deleteError.value = getErrorMessage(e) || 'Failed to delete source'
    return
  }
  deleteOpen.value = false
  toast.success('Source deleted')
  router.push({ name: 'sources' })
}
</script>

<template>
  <div v-if="isLoading">
    <SSkeleton class="h-8 w-48 mb-4" />
    <SSkeleton class="h-64" />
  </div>
  <div v-else-if="error">
    <SAlert variant="danger">
      Failed to load source: {{ getErrorMessage(error) }}
    </SAlert>
  </div>
  <div v-else-if="source">
    <PageHeader
      :title="source.name"
      :description="connectorLabel"
      back-label="Sources"
      :back-to="{ name: 'sources' }"
    >
      <template #actions>
        <SButton variant="secondary" :loading="catalogStatus === 'discovering'" @click="handleRefreshCatalog">
          <RefreshCw class="w-4 h-4" /> Refresh Catalog
        </SButton>
        <SButton variant="secondary" :to="`/sources/${source.id}/edit`">
          <Pencil class="w-4 h-4" /> Edit
        </SButton>
        <SButton variant="danger" @click="requestDelete">
          <Trash2 class="w-4 h-4" /> Delete
        </SButton>
      </template>
    </PageHeader>

    <SCard class="mb-6">
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm p-6">
        <div>
          <span class="text-text-muted">Connector</span>
          <p class="text-text-primary font-medium">
            {{ connectorLabel }}
          </p>
          <RouterLink
            v-if="hasUpdate"
            :to="{ name: 'settings-connectors' }"
            class="mt-1 inline-flex items-center gap-2 hover:underline"
            title="Manage connector versions"
          >
            <SBadge variant="info">
              Update available
            </SBadge>
            <span class="text-xs text-primary">Latest: v{{ versionInfo!.latestVersion }}</span>
          </RouterLink>
        </div>
        <div>
          <span class="text-text-muted">Created</span>
          <p class="text-text-primary">
            {{ source.createdAt?.toLocaleDateString() ?? '-' }}
          </p>
        </div>
        <div>
          <span class="text-text-muted">Catalog</span>
          <div class="flex items-center gap-2 mt-1">
            <SBadge :variant="catalogBadgeVariant" dot>
              {{ catalogStatusLabel }}
            </SBadge>
            <span v-if="catalog?.discoveredAt" class="text-xs text-text-secondary">
              Last discovered: {{ catalog.discoveredAt.toLocaleDateString() }}
            </span>
            <span v-else class="text-xs text-text-secondary">--</span>
          </div>
        </div>
      </div>
    </SCard>

    <InstanceConnectionsCard kind="source" :instance-id="source.id" />

    <RuntimeConfigForm
      :runtime-config="source.runtimeConfig"
      entity-type="source"
      :saving="updateSource.isPending.value"
      @save="handleSaveRuntimeConfig"
      @reset="handleResetRuntimeConfig"
    />

    <DeleteInstanceDialog
      :open="deleteOpen"
      kind="source"
      :name="source.name"
      :loading="deleteSource.isPending.value"
      :error="deleteError"
      @confirm="handleDelete"
      @cancel="deleteOpen = false"
    />
  </div>
</template>
