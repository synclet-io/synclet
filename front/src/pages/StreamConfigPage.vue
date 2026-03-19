<script setup lang="ts">
import type { ConfiguredStream, DestinationSyncMode, SelectedField, SyncMode } from '@entities/connection'
import type { StreamRow } from '@features/stream-config/StreamExpandedRow.vue'
import type { StreamValidationError } from '@features/stream-config/useStreamValidation'
import { discoverSchema, getConfiguredCatalog, getDiscoveredCatalog, resetConnectionState, useConfigureStreams, useConnection, useStreamStates } from '@entities/connection'
import { useConnectorTaskResult } from '@entities/connector-task'
import { getSourceCatalog } from '@entities/source'
import { getLeafFields, parseJsonSchema } from '@features/stream-config/schemaParser'
import StreamExpandedRow from '@features/stream-config/StreamExpandedRow.vue'
import { useStreamValidation } from '@features/stream-config/useStreamValidation'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SBadge, SButton, SConfirmDialog, SEmptyState, SInput, SSelect, SSkeleton, useToast } from '@shared/ui'
import { AlertCircle, ChevronDown, ChevronRight, ChevronUp, RefreshCw, RotateCcw, Save } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

interface DiscoveredCatalog {
  streams: Array<{
    stream: {
      name: string
      namespace?: string
      json_schema?: Record<string, unknown>
      jsonSchema?: Record<string, unknown>
      supported_sync_modes?: string[]
      source_defined_cursor?: boolean
      default_cursor_field?: string[]
      source_defined_primary_key?: string[][]
    }
    sync_mode?: string
    destination_sync_mode?: string
  }>
}

interface ConfiguredCatalogInput {
  streams?: Array<{
    stream?: { name?: string, namespace?: string }
    name?: string
    namespace?: string
    sync_mode?: string
    syncMode?: string
    destination_sync_mode?: string
    destinationSyncMode?: string
    cursor_field?: string[]
    cursorField?: string[]
    primary_key?: Array<string[] | { field_path?: string[], fieldPath?: string[] }>
    primaryKey?: Array<string[] | { field_path?: string[], fieldPath?: string[] }>
    selected_fields?: Array<{ field_path?: string[], fieldPath?: string[] }>
    selectedFields?: Array<{ field_path?: string[], fieldPath?: string[] }>
  }>
}

const route = useRoute()
const router = useRouter()
const id = route.params.id as string
const toast = useToast()

const { data: connection } = useConnection(id)

interface NamespaceGroup {
  namespace: string
  streams: { stream: StreamRow, index: number }[]
}

const catalog = ref<DiscoveredCatalog | null>(null)
const streams = ref<StreamRow[]>([])
const loading = ref(true)
const saving = ref(false)
const discovering = ref(false)
const error = ref('')
const searchQuery = ref('')

const expandedStream = ref<string | null>(null)
const validationErrors = ref<StreamValidationError[]>([])
const hasAttemptedSave = ref(false)
const expandedNamespaces = ref(new Set<string>())
const confirmResetAll = ref(false)

const discoverTaskId = ref<string | null>(null)
const { data: discoverTaskResult } = useConnectorTaskResult(discoverTaskId)

// Watch discover task result and extract catalog when complete
watch(discoverTaskResult, (task) => {
  if (!task)
    return
  if (task.status === 'completed' && task.discoverResult) {
    const cat = task.discoverResult.catalog as unknown as DiscoveredCatalog | undefined
    catalog.value = cat ?? null
    if (cat)
      parseStreams(cat)
    discovering.value = false
    discoverTaskId.value = null
    toast.success('Schema discovered')
  }
  else if (task.status === 'failed') {
    error.value = task.errorMessage || 'Schema discovery failed. Check that the source connector is running and try again.'
    discovering.value = false
    discoverTaskId.value = null
  }
})

const configureStreamsMutation = useConfigureStreams()
const { validate } = useStreamValidation()
const { data: streamStatesResult } = useStreamStates(id)

function streamKey(s: StreamRow): string {
  return s.namespace ? `${s.namespace}.${s.name}` : s.name
}

// Load catalog: try cached source catalog first, fall back to connection-scoped discover
const hasLoadedCatalog = ref(false)
watch(connection, async (conn) => {
  if (!conn || hasLoadedCatalog.value || !loading.value)
    return

  // Try cached catalog from source first (avoids synchronous Docker)
  if (conn.sourceId) {
    try {
      const cached = await getSourceCatalog(conn.sourceId)
      if (cached && cached.catalog && Object.keys(cached.catalog).length > 0) {
        catalog.value = cached.catalog as unknown as DiscoveredCatalog
        let configuredCat: ConfiguredCatalogInput | undefined
        try {
          configuredCat = await getConfiguredCatalog(id) as ConfiguredCatalogInput | undefined
        }
        catch { /* No configured catalog yet */ }
        parseStreams(cached.catalog as unknown as DiscoveredCatalog, configuredCat)
        hasLoadedCatalog.value = true
        loading.value = false
        return
      }
    }
    catch { /* Fallback to connection-scoped discover below */ }
  }

  // Fallback: existing getDiscoveredCatalog (connection-scoped, synchronous Docker)
  try {
    const cat = await getDiscoveredCatalog(id) as DiscoveredCatalog | undefined
    catalog.value = cat ?? null
    let configuredCat: ConfiguredCatalogInput | undefined
    try {
      configuredCat = await getConfiguredCatalog(id) as ConfiguredCatalogInput | undefined
    }
    catch { /* No configured catalog yet */ }
    if (cat)
      parseStreams(cat, configuredCat)
  }
  catch { /* No catalog yet */ }
  hasLoadedCatalog.value = true
  loading.value = false
}, { immediate: true })

function parseStreams(cat: DiscoveredCatalog, configuredCat?: ConfiguredCatalogInput) {
  if (!cat?.streams)
    return

  type ConfiguredEntry = NonNullable<ConfiguredCatalogInput['streams']>[number]
  const configuredMap = new Map<string, ConfiguredEntry>()
  if (configuredCat?.streams) {
    for (const cs of configuredCat.streams) {
      const stream = cs.stream || cs
      const name = stream.name || ''
      const namespace = stream.namespace || ''
      const key = namespace ? `${namespace}.${name}` : name
      configuredMap.set(key, cs)
    }
  }

  const hasConfigured = configuredMap.size > 0

  streams.value = cat.streams.map((s) => {
    const stream = s.stream || s
    const jsonSchema = stream.json_schema || stream.jsonSchema || {}
    const name = stream.name || ''
    const namespace = stream.namespace || ''
    const key = namespace ? `${namespace}.${name}` : name

    const configured = configuredMap.get(key)

    if (configured) {
      const cursorField = configured.cursor_field || configured.cursorField || []
      const primaryKey = (configured.primary_key || configured.primaryKey || []).map((pk: string[] | { field_path?: string[], fieldPath?: string[] }) => {
        if (Array.isArray(pk))
          return pk
        if (pk?.field_path || pk?.fieldPath)
          return pk.field_path || pk.fieldPath || []
        return []
      })
      const selectedFields = (configured.selected_fields || configured.selectedFields || []).map((sf: { field_path?: string[], fieldPath?: string[] }) => {
        if (sf?.field_path || sf?.fieldPath)
          return { fieldPath: sf.field_path || sf.fieldPath || [] }
        return { fieldPath: [] as string[] }
      })

      return {
        name,
        namespace,
        syncMode: (configured.sync_mode || configured.syncMode || 'full_refresh') as SyncMode,
        destinationSyncMode: (configured.destination_sync_mode || configured.destinationSyncMode || 'overwrite') as DestinationSyncMode,
        enabled: true,
        supportedSyncModes: stream.supported_sync_modes || ['full_refresh', 'incremental'],
        cursorField,
        primaryKey,
        selectedFields,
        jsonSchema,
        sourceDefinedCursor: !!stream.source_defined_cursor,
        defaultCursorField: stream.default_cursor_field || [],
        sourceDefinedPrimaryKey: stream.source_defined_primary_key || [],
      } satisfies StreamRow
    }

    return {
      name,
      namespace,
      syncMode: (s.sync_mode || 'full_refresh') as SyncMode,
      destinationSyncMode: (s.destination_sync_mode || 'overwrite') as DestinationSyncMode,
      enabled: !hasConfigured,
      supportedSyncModes: stream.supported_sync_modes || ['full_refresh', 'incremental'],
      cursorField: [] as string[],
      primaryKey: [] as string[][],
      selectedFields: [] as SelectedField[],
      jsonSchema,
      sourceDefinedCursor: !!stream.source_defined_cursor,
      defaultCursorField: stream.default_cursor_field || [],
      sourceDefinedPrimaryKey: stream.source_defined_primary_key || [],
    } satisfies StreamRow
  })

  const namespaces = new Set<string>()
  for (const s of streams.value) {
    namespaces.add(s.namespace)
  }
  expandedNamespaces.value = namespaces
}

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
    await resetConnectionState(id)
    toast.success('All stream state reset')
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e)
  }
}

async function saveConfig() {
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
      <SButton variant="danger" @click="confirmResetAll = true">
        <RotateCcw class="w-4 h-4" /> Reset All State
      </SButton>
      <SButton variant="secondary" :loading="discovering" @click="handleDiscoverSchema">
        <RefreshCw class="w-4 h-4" /> Refresh Catalog
      </SButton>
      <SButton :loading="saving" @click="saveConfig">
        <Save class="w-4 h-4" /> Save Configuration
      </SButton>
    </template>
  </PageHeader>

  <SAlert v-if="error" variant="danger" class="mb-4" dismissible @dismiss="error = ''">
    {{ error }}
  </SAlert>

  <!-- Loading state -->
  <div v-if="loading" class="bg-surface border border-border rounded-xl overflow-hidden p-5 space-y-3">
    <SSkeleton v-for="i in 5" :key="i" height="40px" />
  </div>

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
                @change="streams.forEach(s => s.enabled = ($event.target as HTMLInputElement).checked)"
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
</template>
