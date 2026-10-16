<script setup lang="ts">
import type { ConfiguredStream, DestinationSyncMode, SyncMode } from '@entities/connection'
import type { ConfiguredCatalogInput, DiscoveredCatalog } from '@features/stream-config/catalogStreams'
import type { StreamRow } from '@features/stream-config/StreamExpandedRow.vue'
import type { StreamValidationError } from '@features/stream-config/useStreamValidation'
import { Code, ConnectError } from '@connectrpc/connect'
import { discoverSchema, getConfiguredCatalog, getDiscoveredCatalog, useConfigureStreams, useConnection, useResetConnectionState, useStreamStates } from '@entities/connection'
import { useConnectorTaskResult } from '@entities/connector-task'
import { getSourceCatalog } from '@entities/source'
import { mergeDiscoveredStreams, parseStreams, streamKey } from '@features/stream-config/catalogStreams'
import { getLeafFields, parseJsonSchema } from '@features/stream-config/schemaParser'
import StreamExpandedRow from '@features/stream-config/StreamExpandedRow.vue'
import { useStreamValidation } from '@features/stream-config/useStreamValidation'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SBadge, SButton, SConfirmDialog, SEmptyState, SInput, SSelect, SSkeleton, useToast } from '@shared/ui'
import { AlertCircle, ChevronDown, ChevronRight, ChevronUp, RefreshCw, RotateCcw, Save } from 'lucide-vue-next'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const id = route.params.id as string
const toast = useToast()

const { data: connection } = useConnection(id)

interface NamespaceGroup {
  namespace: string
  streams: { stream: StreamRow, index: number }[]
}

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
const searchQuery = ref('')

const expandedStream = ref<string | null>(null)
const validationErrors = ref<StreamValidationError[]>([])
const hasAttemptedSave = ref(false)
const expandedNamespaces = ref(new Set<string>())
const confirmResetAll = ref(false)
const savedSnapshot = ref('')
const pendingLeave = ref<string | null>(null)

const discoverTaskId = ref<string | null>(null)
const { data: discoverTaskResult } = useConnectorTaskResult(discoverTaskId)

function configSnapshot(rows: StreamRow[]): string {
  return JSON.stringify(rows
    .filter(s => s.enabled)
    .map(s => [streamKey(s), s.syncMode, s.destinationSyncMode, s.cursorField, s.primaryKey, s.selectedFields]))
}

const isDirty = computed(() => !loading.value && configSnapshot(streams.value) !== savedSnapshot.value)
const allEnabled = computed(() => streams.value.length > 0 && streams.value.every(s => s.enabled))
const someEnabled = computed(() => streams.value.some(s => s.enabled))

function setAllEnabled(enabled: boolean) {
  for (const s of streams.value)
    s.enabled = enabled
  revalidateIfNeeded()
}

function expandAllNamespaces() {
  expandedNamespaces.value = new Set(streams.value.map(s => s.namespace))
}

function setLoadedStreams(rows: StreamRow[]) {
  streams.value = rows
  savedSnapshot.value = configSnapshot(rows)
  expandAllNamespaces()
}

watch(discoverTaskResult, (task) => {
  if (!task)
    return
  if (task.status === 'completed' && task.discoverResult) {
    const cat = task.discoverResult.catalog as unknown as DiscoveredCatalog | undefined
    if (cat) {
      const { streams: merged, added, removed } = mergeDiscoveredStreams(streams.value, parseStreams(cat))
      streams.value = merged
      expandAllNamespaces()
      revalidateIfNeeded()
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
const { validate } = useStreamValidation()
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

// Namespace grouping
const groupedStreams = computed<NamespaceGroup[]>(() => {
  const groups = new Map<string, { stream: StreamRow, index: number }[]>()
  streams.value.forEach((s, i) => {
    const ns = s.namespace
    if (!groups.has(ns))
      groups.set(ns, [])
    groups.get(ns)!.push({ stream: s, index: i })
  })

  const result: NamespaceGroup[] = []
  if (groups.has('')) {
    result.push({ namespace: '', streams: groups.get('')! })
    groups.delete('')
  }
  const sorted = Array.from(groups.entries()).sort(([a], [b]) => a.localeCompare(b))
  for (const [ns, items] of sorted) {
    result.push({ namespace: ns, streams: items })
  }
  return result
})

const filteredGroupedStreams = computed(() => {
  if (!searchQuery.value)
    return groupedStreams.value
  const q = searchQuery.value.toLowerCase()
  return groupedStreams.value
    .map(g => ({
      ...g,
      streams: g.streams.filter(s => s.stream.name.toLowerCase().includes(q)
        || s.stream.namespace.toLowerCase().includes(q)),
    }))
    .filter(g => g.streams.length > 0)
})

const hasMultipleNamespaces = computed(() => groupedStreams.value.length > 1)

function toggleNamespace(ns: string) {
  const updated = new Set(expandedNamespaces.value)
  if (updated.has(ns)) {
    updated.delete(ns)
  }
  else {
    updated.add(ns)
  }
  expandedNamespaces.value = updated
}

function isNamespaceExpanded(ns: string): boolean {
  return expandedNamespaces.value.has(ns)
}

function toggleExpand(s: StreamRow) {
  const key = streamKey(s)
  expandedStream.value = expandedStream.value === key ? null : key
}

function selectedFieldCount(s: StreamRow): number {
  return s.selectedFields.length
}

function totalFieldCount(s: StreamRow): number {
  if (!s.jsonSchema)
    return 0
  const fields = parseJsonSchema(s.jsonSchema)
  return getLeafFields(fields).length
}

function errorsForStream(s: StreamRow): StreamValidationError[] {
  const key = streamKey(s)
  return validationErrors.value.filter(e => e.streamKey === key)
}

function hasStreamErrors(s: StreamRow): boolean {
  return errorsForStream(s).length > 0
}

function streamErrorSummary(): string {
  const byStream = new Map<string, string[]>()
  for (const err of validationErrors.value) {
    if (!byStream.has(err.streamKey))
      byStream.set(err.streamKey, [])
    byStream.get(err.streamKey)!.push(err.message)
  }
  const parts: string[] = []
  for (const [stream, msgs] of byStream) {
    parts.push(`${stream}: ${msgs.join(', ')}`)
  }
  return parts.join('; ')
}

function updateStreamByKey(updatedStream: StreamRow) {
  const key = streamKey(updatedStream)
  const idx = streams.value.findIndex(s => streamKey(s) === key)
  if (idx !== -1) {
    streams.value[idx] = updatedStream
  }
  revalidateIfNeeded()
}

function revalidateIfNeeded() {
  if (hasAttemptedSave.value) {
    validationErrors.value = validate(streams.value)
  }
}

function displayStreamName(s: StreamRow): string {
  if (hasMultipleNamespaces.value)
    return s.name
  return s.namespace ? `${s.namespace}.${s.name}` : s.name
}

const validSyncCombinations: { src: SyncMode, dst: DestinationSyncMode, label: string }[] = [
  { src: 'incremental', dst: 'append_dedup', label: 'Incremental | Append + Dedup' },
  { src: 'incremental', dst: 'append', label: 'Incremental | Append' },
  { src: 'full_refresh', dst: 'overwrite', label: 'Full Refresh | Overwrite' },
  { src: 'full_refresh', dst: 'append', label: 'Full Refresh | Append' },
]

function combinedSyncModeOptions(s: StreamRow) {
  return validSyncCombinations
    .filter(c => s.supportedSyncModes.includes(c.src))
    .map(c => ({
      value: `${c.src}|${c.dst}`,
      label: c.label,
    }))
}

function combinedSyncModeValue(s: StreamRow): string {
  return `${s.syncMode}|${s.destinationSyncMode}`
}

function applyCombinedSyncMode(s: StreamRow, combined: string) {
  const [src, dst] = combined.split('|')
  s.syncMode = src as SyncMode
  s.destinationSyncMode = dst as DestinationSyncMode
  revalidateIfNeeded()
}

function getStreamStateData(namespace: string, name: string): string | undefined {
  const state = streamStatesResult.value?.states.find(s => s.streamNamespace === namespace && s.streamName === name)
  return state?.stateData
}

function getStreamStateUpdatedAt(namespace: string, name: string): Date | undefined {
  const state = streamStatesResult.value?.states.find(s => s.streamNamespace === namespace && s.streamName === name)
  return state?.updatedAt
}

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
  hasAttemptedSave.value = true
  const errors = validate(streams.value)
  validationErrors.value = errors

  if (errors.length > 0) {
    error.value = `Fix ${errors.length} issue(s) before saving: ${streamErrorSummary()}`
    expandedStream.value = errors[0].streamKey
    return
  }

  saving.value = true
  error.value = ''
  try {
    const configured: ConfiguredStream[] = streams.value
      .filter(s => s.enabled)
      .map(s => ({
        streamName: s.name,
        namespace: s.namespace,
        syncMode: s.syncMode,
        destinationSyncMode: s.destinationSyncMode,
        cursorField: s.sourceDefinedCursor ? [] : s.cursorField,
        primaryKey: s.sourceDefinedPrimaryKey.length > 0 ? [] : s.primaryKey,
        selectedFields: s.selectedFields,
      }))
    await configureStreamsMutation.mutateAsync({ connectionId: id, streams: configured })
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

  <!-- Stream table -->
  <div v-else class="bg-surface border border-border rounded-xl overflow-hidden">
    <!-- Search -->
    <div class="p-3 border-b border-border">
      <SInput v-model="searchQuery" placeholder="Search streams..." class="max-w-sm" />
    </div>

    <div class="overflow-x-auto">
      <table class="w-full min-w-[700px]">
        <thead class="bg-surface-raised">
          <tr>
            <th class="px-3 py-2 text-left w-10">
              <input
                type="checkbox"
                class="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20 cursor-pointer"
                aria-label="Enable all streams"
                :checked="allEnabled"
                :indeterminate="someEnabled && !allEnabled"
                @change="setAllEnabled(($event.target as HTMLInputElement).checked)"
              >
            </th>
            <th class="px-3 py-2 text-left text-xs font-medium text-text-secondary uppercase">
              Stream Name
            </th>
            <th class="px-3 py-2 text-left text-xs font-medium text-text-secondary uppercase w-52">
              Sync Mode
            </th>
            <th class="px-3 py-2 text-left text-xs font-medium text-text-secondary uppercase">
              Fields
            </th>
            <th class="px-3 py-2 w-10" />
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <template v-for="group in filteredGroupedStreams" :key="group.namespace">
            <!-- Namespace header -->
            <tr
              v-if="hasMultipleNamespaces"
              class="cursor-pointer hover:bg-surface-hover transition-colors bg-surface-raised/50"
              @click="toggleNamespace(group.namespace)"
            >
              <td colspan="5" class="px-3 py-2">
                <div class="flex items-center gap-2">
                  <ChevronDown v-if="isNamespaceExpanded(group.namespace)" class="w-4 h-4 text-text-secondary" />
                  <ChevronRight v-else class="w-4 h-4 text-text-secondary" />
                  <span class="font-semibold text-sm text-heading">
                    {{ group.namespace || 'Default' }}
                  </span>
                  <SBadge variant="gray">
                    {{ group.streams.length }} streams
                  </SBadge>
                </div>
              </td>
            </tr>

            <!-- Stream rows -->
            <template v-if="!hasMultipleNamespaces || isNamespaceExpanded(group.namespace)">
              <template v-for="{ stream: s } in group.streams" :key="streamKey(s)">
                <!-- Collapsed row -->
                <tr
                  class="cursor-pointer hover:bg-surface-hover transition-colors h-10"
                  :class="{ 'opacity-50': !s.enabled }"
                  @click="toggleExpand(s)"
                >
                  <td class="px-3 py-2 w-10" @click.stop>
                    <input
                      v-model="s.enabled" type="checkbox" :aria-label="`Enable stream ${s.name}`"
                      class="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20 cursor-pointer"
                    >
                  </td>
                  <td class="px-3 py-2 text-sm font-medium text-heading">
                    {{ displayStreamName(s) }}
                    <AlertCircle v-if="hasAttemptedSave && hasStreamErrors(s)" class="inline w-4 h-4 text-danger ml-1" />
                  </td>
                  <td class="px-3 py-2" @click.stop>
                    <SSelect
                      size="sm"
                      :model-value="combinedSyncModeValue(s)"
                      :options="combinedSyncModeOptions(s)"
                      @update:model-value="applyCombinedSyncMode(s, $event as string)"
                    />
                  </td>
                  <td class="px-3 py-2">
                    <span class="text-xs text-text-muted">{{ selectedFieldCount(s) }}/{{ totalFieldCount(s) }} fields</span>
                  </td>
                  <td class="px-3 py-2 w-10">
                    <ChevronUp
                      v-if="expandedStream === streamKey(s)" class="w-4 h-4"
                      :aria-label="`Collapse stream ${s.name}`"
                    />
                    <ChevronDown
                      v-else class="w-4 h-4"
                      :aria-label="`Expand stream ${s.name}`"
                    />
                  </td>
                </tr>
                <!-- Expanded row -->
                <tr v-if="expandedStream === streamKey(s)">
                  <td colspan="5">
                    <StreamExpandedRow
                      :stream="s"
                      :schema="s.jsonSchema"
                      :validation-errors="errorsForStream(s)"
                      :connection-id="id"
                      :state-data="getStreamStateData(s.namespace, s.name)"
                      :state-updated-at="getStreamStateUpdatedAt(s.namespace, s.name)"
                      @update:stream="updateStreamByKey($event)"
                    />
                  </td>
                </tr>
              </template>
            </template>
          </template>
        </tbody>
      </table>
    </div>
  </div>

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
