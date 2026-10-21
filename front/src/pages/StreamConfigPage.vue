<script setup lang="ts">
import type { ConfiguredCatalogInput, DiscoveredCatalog, StreamRow } from '@features/stream-config/catalogStreams'
import { Code, ConnectError } from '@connectrpc/connect'
import { discoverSchema, getConfiguredCatalog, getDiscoveredCatalog, useConfigureStreams, useConnection, useResetConnectionState, useStreamStates } from '@entities/connection'
import { useConnectorTaskResult } from '@entities/connector-task'
import { getSourceCatalog } from '@entities/source'
import { mergeDiscoveredStreams, parseStreams, streamKey, toConfiguredStreams } from '@features/stream-config/catalogStreams'
import StreamConfigEditor from '@features/stream-config/StreamConfigEditor.vue'
import { streamErrorSummary } from '@features/stream-config/useStreamValidation'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SButton, SConfirmDialog, SEmptyState, SSkeleton, useToast } from '@shared/ui'
import { RefreshCw, RotateCcw, Save } from 'lucide-vue-next'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const id = route.params.id as string
const toast = useToast()

const { data: connection } = useConnection(id)

interface DiscoverySummary {
  added: string[]
  removed: string[]
}

const streams = ref<StreamRow[]>([])
const loading = ref(true)
const saving = ref(false)
const discovering = ref(false)
const error = ref('')
const configuredCatalogError = ref('')
const discoverySummary = ref<DiscoverySummary | null>(null)
const editor = ref<InstanceType<typeof StreamConfigEditor> | null>(null)
const confirmResetAll = ref(false)
const savedSnapshot = ref('')
const pendingLeave = ref<string | null>(null)

const discoverTaskId = ref<string | null>(null)
const { data: discoverTaskResult } = useConnectorTaskResult(discoverTaskId)

function configSnapshot(rows: StreamRow[]): string {
  return JSON.stringify(rows
    .filter(s => s.enabled)
    .map(s => [streamKey(s), s.syncMode, s.destinationSyncMode, s.cursorField, s.primaryKey, s.selectedFields, s.mappers]))
}

const isDirty = computed(() => !loading.value && configSnapshot(streams.value) !== savedSnapshot.value)

function setLoadedStreams(rows: StreamRow[]) {
  streams.value = rows
  savedSnapshot.value = configSnapshot(rows)
}

watch(discoverTaskResult, (task) => {
  if (!task)
    return
  if (task.status === 'completed' && task.discoverResult) {
    const cat = task.discoverResult.catalog as unknown as DiscoveredCatalog | undefined
    if (cat) {
      const { streams: merged, added, removed } = mergeDiscoveredStreams(streams.value, parseStreams(cat))
      streams.value = merged
      discoverySummary.value = added.length > 0 || removed.length > 0 ? { added, removed } : null
    }
    discovering.value = false
    discoverTaskId.value = null
    toast.success(discoverySummary.value
      ? `Schema discovered: ${discoverySummary.value.added.length} added, ${discoverySummary.value.removed.length} removed`
      : 'Schema discovered: no stream changes')
  }
  else if (task.status === 'failed') {
    error.value = task.errorMessage || 'Schema discovery failed. Check that the source connector is running and try again.'
    discovering.value = false
    discoverTaskId.value = null
  }
})

const configureStreamsMutation = useConfigureStreams()
const resetConnectionStateMutation = useResetConnectionState()
const { data: streamStatesResult } = useStreamStates(id)

async function loadDiscoveredCatalog(sourceId: string | undefined): Promise<DiscoveredCatalog | undefined> {
  if (sourceId) {
    try {
      const cached = await getSourceCatalog(sourceId)
      if (cached?.catalog && Object.keys(cached.catalog).length > 0)
        return cached.catalog as unknown as DiscoveredCatalog
    }
    catch (e) {
      console.warn('[stream-config] getSourceCatalog failed, falling back to connection discover', e)
    }
  }
  return await getDiscoveredCatalog(id) as DiscoveredCatalog | undefined
}

async function loadConfiguredCatalog(): Promise<ConfiguredCatalogInput | undefined> {
  try {
    return await getConfiguredCatalog(id) as ConfiguredCatalogInput | undefined
  }
  catch (e) {
    if (ConnectError.from(e).code === Code.NotFound)
      return undefined
    throw e
  }
}

async function loadCatalog(sourceId: string | undefined) {
  loading.value = true
  error.value = ''
  configuredCatalogError.value = ''
  try {
    const discovered = await loadDiscoveredCatalog(sourceId)
    let configured: ConfiguredCatalogInput | undefined
    try {
      configured = await loadConfiguredCatalog()
    }
    catch (e) {
      configuredCatalogError.value = getErrorMessage(e)
      return
    }
    if (discovered)
      setLoadedStreams(parseStreams(discovered, configured))
  }
  catch (e) {
    error.value = getErrorMessage(e) || 'Failed to load catalog. Try running schema discovery.'
  }
  finally {
    loading.value = false
  }
}

const hasLoadedCatalog = ref(false)
watch(connection, (conn) => {
  if (!conn || hasLoadedCatalog.value)
    return
  hasLoadedCatalog.value = true
  loadCatalog(conn.sourceId)
}, { immediate: true })

function retryLoad() {
  loadCatalog(connection.value?.sourceId)
}

onBeforeRouteLeave((to) => {
  if (!isDirty.value || pendingLeave.value === to.fullPath)
    return true
  pendingLeave.value = to.fullPath
  return false
})

function confirmLeave() {
  if (pendingLeave.value)
    router.push(pendingLeave.value)
}

function onBeforeUnload(e: BeforeUnloadEvent) {
  if (isDirty.value)
    e.preventDefault()
}

onMounted(() => window.addEventListener('beforeunload', onBeforeUnload))
onBeforeUnmount(() => window.removeEventListener('beforeunload', onBeforeUnload))

async function handleDiscoverSchema() {
  discovering.value = true
  error.value = ''
  discoverTaskId.value = null
  try {
    const { taskId } = await discoverSchema(id)
    discoverTaskId.value = taskId
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e)
    discovering.value = false
  }
}

async function resetAllState() {
  confirmResetAll.value = false
  error.value = ''
  try {
    await resetConnectionStateMutation.mutateAsync(id)
    toast.success('All stream state reset')
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e)
  }
}

async function saveConfig() {
  if (configuredCatalogError.value)
    return
  const errors = editor.value?.validate() ?? []
  if (errors.length > 0) {
    error.value = `Fix ${errors.length} issue(s) before saving: ${streamErrorSummary(errors)}`
    return
  }

  saving.value = true
  error.value = ''
  try {
    await configureStreamsMutation.mutateAsync({ connectionId: id, streams: toConfiguredStreams(streams.value) })
    savedSnapshot.value = configSnapshot(streams.value)
    router.push(`/connections/${id}`)
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e)
  }
  finally {
    saving.value = false
  }
}
</script>

<template>
  <PageHeader
    title="Configure Streams"
    description="Select and configure streams for this connection"
    :back-label="connection?.name || 'Connection'"
    :back-to="`/connections/${id}`"
  >
    <template #actions>
      <SButton variant="danger" :loading="resetConnectionStateMutation.isPending.value" @click="confirmResetAll = true">
        <RotateCcw class="w-4 h-4" /> Reset All State
      </SButton>
      <SButton variant="secondary" :loading="discovering" @click="handleDiscoverSchema">
        <RefreshCw class="w-4 h-4" /> Refresh Catalog
      </SButton>
      <SButton :loading="saving" :disabled="loading || !!configuredCatalogError" @click="saveConfig">
        <Save class="w-4 h-4" /> Save Configuration
      </SButton>
    </template>
  </PageHeader>

  <SAlert v-if="error" variant="danger" class="mb-4" dismissible @dismiss="error = ''">
    {{ error }}
  </SAlert>

  <SAlert v-if="discoverySummary" variant="info" class="mb-4" dismissible @dismiss="discoverySummary = null">
    <p v-if="discoverySummary.added.length > 0">
      Added (disabled until you enable them): {{ discoverySummary.added.join(', ') }}
    </p>
    <p v-if="discoverySummary.removed.length > 0">
      Removed from the source and dropped on save: {{ discoverySummary.removed.join(', ') }}
    </p>
  </SAlert>

  <!-- Loading state -->
  <div v-if="loading" class="bg-surface border border-border rounded-xl overflow-hidden p-5 space-y-3">
    <SSkeleton v-for="i in 5" :key="i" height="40px" />
  </div>

  <SAlert v-else-if="configuredCatalogError" variant="danger">
    <p>Could not load the saved stream configuration: {{ configuredCatalogError }}</p>
    <p class="mt-1">
      Editing is blocked so the saved configuration is not overwritten with defaults.
    </p>
    <SButton size="sm" variant="secondary" class="mt-3" @click="retryLoad">
      <RefreshCw class="w-4 h-4" /> Retry
    </SButton>
  </SAlert>

  <!-- Empty state -->
  <div v-else-if="streams.length === 0" class="py-12">
    <SEmptyState title="No streams discovered yet" description="Discover the schema to see available streams">
      <SButton :loading="discovering" @click="handleDiscoverSchema">
        <RefreshCw class="w-4 h-4" /> Refresh Catalog
      </SButton>
    </SEmptyState>
  </div>

  <StreamConfigEditor
    v-else
    ref="editor"
    v-model:streams="streams"
    :connection-id="id"
    :stream-states="streamStatesResult?.states"
  />

  <SConfirmDialog
    :open="confirmResetAll"
    title="Reset all state"
    message="Reset state for all streams? This will cause a full resync of the entire connection."
    confirm-text="Reset"
    @confirm="resetAllState"
    @cancel="confirmResetAll = false"
  />

  <SConfirmDialog
    :open="!!pendingLeave && isDirty"
    title="Discard unsaved changes"
    message="You have unsaved stream configuration changes. Leave this page and discard them?"
    confirm-text="Discard changes"
    @confirm="confirmLeave"
    @cancel="pendingLeave = null"
  />
</template>
