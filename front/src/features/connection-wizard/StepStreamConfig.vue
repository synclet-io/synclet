<script setup lang="ts">
import type { ConfiguredStream, DestinationSyncMode, SelectedField, SyncMode } from '@entities/connection'
import type { StreamValidationError } from '@features/stream-config/useStreamValidation'
import { useConnectorTaskResult } from '@entities/connector-task'
import { discoverSourceSchema, getSourceCatalog } from '@entities/source'
import { getLeafFields, parseJsonSchema } from '@features/stream-config/schemaParser'
import StreamExpandedRow from '@features/stream-config/StreamExpandedRow.vue'
import { useStreamValidation } from '@features/stream-config/useStreamValidation'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SBadge, SButton, SEmptyState, SSkeleton } from '@shared/ui'
import { AlertCircle, ChevronDown, ChevronRight, ChevronUp, RefreshCw } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'

interface StreamRow {
  name: string
  namespace: string
  syncMode: SyncMode
  destinationSyncMode: DestinationSyncMode
  enabled: boolean
  supportedSyncModes: string[]
  cursorField: string[]
  primaryKey: string[][]
  selectedFields: SelectedField[]
  jsonSchema: Record<string, any>
  sourceDefinedCursor: boolean
  defaultCursorField: string[]
  sourceDefinedPrimaryKey: string[][]
}

interface NamespaceGroup {
  namespace: string
  streams: { stream: StreamRow, index: number }[]
}

const props = defineProps<{
  sourceId: string
  streams: ConfiguredStream[]
  discoveredCatalog: Record<string, unknown> | null
}>()

const emit = defineEmits<{
  'update:streams': [streams: ConfiguredStream[]]
  'update:discoveredCatalog': [catalog: Record<string, unknown> | null]
  'skip': []
}>()

const localStreams = ref<StreamRow[]>([])
const discovering = ref(false)
const error = ref('')
const expandedStream = ref<string | null>(null)
const validationErrors = ref<StreamValidationError[]>([])
const expandedNamespaces = ref(new Set<string>())

const discoverTaskId = ref<string | null>(null)
const { data: discoverTaskResult } = useConnectorTaskResult(discoverTaskId)

// Watch discover task result and extract catalog when complete
watch(discoverTaskResult, (task) => {
  if (!task)
    return
  if (task.status === 'completed' && task.discoverResult) {
    const cat = task.discoverResult.catalog
    parseStreams(cat)
    emit('update:discoveredCatalog', cat ?? null)
    discovering.value = false
    discoverTaskId.value = null
  }
  else if (task.status === 'failed') {
    error.value = task.errorMessage || 'Schema discovery failed. Check that the source connector is running and try again.'
    discovering.value = false
    discoverTaskId.value = null
  }
})

const { validate } = useStreamValidation()

function parseStreams(cat: any) {
  if (!cat?.streams)
    return

  localStreams.value = cat.streams.map((s: any) => {
    const stream = s.stream || s
    const jsonSchema = stream.json_schema || stream.jsonSchema || {}
    const name = stream.name || ''
    const namespace = stream.namespace || ''

    return {
      name,
      namespace,
      syncMode: (s.sync_mode || 'full_refresh') as SyncMode,
      destinationSyncMode: (s.destination_sync_mode || 'overwrite') as DestinationSyncMode,
      enabled: false,
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

  // Initialize all namespaces as expanded
  const namespaces = new Set<string>()
  for (const s of localStreams.value) {
    namespaces.add(s.namespace)
  }
  expandedNamespaces.value = namespaces
}

function restoreSelectionsFromProps(rows: StreamRow[]) {
  // Restore enabled state, sync modes, cursors, PKs from props.streams
  const configured = new Map<string, ConfiguredStream>()
  for (const s of props.streams) {
    const key = s.namespace ? `${s.namespace}.${s.streamName}` : s.streamName
    configured.set(key, s)
  }
  for (const row of rows) {
    const key = row.namespace ? `${row.namespace}.${row.name}` : row.name
    const cfg = configured.get(key)
    if (cfg) {
      row.enabled = true
      row.syncMode = cfg.syncMode
      row.destinationSyncMode = cfg.destinationSyncMode
      row.cursorField = cfg.cursorField
      row.primaryKey = cfg.primaryKey
      row.selectedFields = cfg.selectedFields
    }
  }
}

// Restore from persisted catalog on mount, or try cached catalog from source
onMounted(async () => {
  if (props.discoveredCatalog && localStreams.value.length === 0) {
    parseStreams(props.discoveredCatalog)
    restoreSelectionsFromProps(localStreams.value)
    return
  }
  // Try loading cached catalog from source
  try {
    const cached = await getSourceCatalog(props.sourceId)
    if (cached && cached.catalog && Object.keys(cached.catalog).length > 0) {
      parseStreams(cached.catalog)
      emit('update:discoveredCatalog', cached.catalog as Record<string, unknown>)
    }
  }
  catch {
    // No cached catalog available, will show empty state / discover button
  }
})

function toConfiguredStreams(rows: StreamRow[]): ConfiguredStream[] {
  return rows.filter(r => r.enabled).map(r => ({
    streamName: r.name,
    namespace: r.namespace,
    syncMode: r.syncMode,
    destinationSyncMode: r.destinationSyncMode,
    cursorField: r.cursorField,
    primaryKey: r.primaryKey,
    selectedFields: r.selectedFields,
  }))
}

// Watch local streams and emit changes
watch(localStreams, (rows) => {
  emit('update:streams', toConfiguredStreams(rows))
}, { deep: true })

// Namespace grouping
const groupedStreams = computed<NamespaceGroup[]>(() => {
  const groups = new Map<string, { stream: StreamRow, index: number }[]>()
  localStreams.value.forEach((s, i) => {
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

function streamKey(s: StreamRow): string {
  return s.namespace ? `${s.namespace}.${s.name}` : s.name
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

function pkSummary(s: StreamRow): string {
  const pk = s.sourceDefinedPrimaryKey.length > 0 ? s.sourceDefinedPrimaryKey : s.primaryKey
  if (pk.length === 0)
    return 'No PK'
  return `PK: ${pk.map(p => p.at(-1)).join(', ')}`
}

function cursorSummary(s: StreamRow): string {
  if (s.sourceDefinedCursor && s.defaultCursorField.length > 0) {
    return `Cursor: ${s.defaultCursorField.join('.')}`
  }
  if (s.cursorField.length > 0) {
    return `Cursor: ${s.cursorField.join('.')}`
  }
  return 'No cursor'
}

function errorsForStream(s: StreamRow): StreamValidationError[] {
  const key = streamKey(s)
  return validationErrors.value.filter(e => e.streamKey === key)
}

function hasStreamErrors(s: StreamRow): boolean {
  return errorsForStream(s).length > 0
}

function updateStream(index: number, updated: StreamRow) {
  localStreams.value[index] = updated
  // Re-validate on changes
  if (validationErrors.value.length > 0) {
    validationErrors.value = validate(localStreams.value)
  }
}

function displayStreamName(s: StreamRow): string {
  if (hasMultipleNamespaces.value)
    return s.name
  return s.namespace ? `${s.namespace}.${s.name}` : s.name
}

async function handleDiscoverSchema() {
  discovering.value = true
  error.value = ''
  discoverTaskId.value = null
  try {
    const { taskId } = await discoverSourceSchema(props.sourceId)
    discoverTaskId.value = taskId
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Schema discovery failed. Check that the source connector is running and try again.'
    discovering.value = false
  }
}
</script>

<template>
  <div>
    <!-- Error alert -->
    <SAlert v-if="error" variant="danger" class="mb-4" dismissible @dismiss="error = ''">
      Schema discovery failed. Check that the source connector is running and try again.
    </SAlert>

    <!-- Loading state during discovery -->
    <div v-if="discovering" class="space-y-3">
      <SSkeleton v-for="i in 3" :key="i" height="40px" />
    </div>

    <!-- Empty state before discovery -->
    <div v-else-if="localStreams.length === 0" class="py-8">
      <SEmptyState title="No streams discovered" description="Click Discover Schema to load available streams from the source connector.">
        <SButton :loading="discovering" @click="handleDiscoverSchema">
          <RefreshCw class="w-4 h-4" /> Discover Schema
        </SButton>
      </SEmptyState>
    </div>

    <!-- Stream table -->
    <div v-else>
      <div class="flex items-center justify-between mb-4">
        <p class="text-sm text-text-secondary">
          {{ localStreams.filter(s => s.enabled).length }} of {{ localStreams.length }} streams enabled
        </p>
        <SButton variant="secondary" size="sm" :loading="discovering" @click="handleDiscoverSchema">
          <RefreshCw class="w-4 h-4" /> Discover Schema
        </SButton>
      </div>

      <div class="bg-surface border border-border rounded-xl overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full min-w-[700px]">
            <thead class="bg-surface-raised">
              <tr>
                <th class="px-5 py-3 text-left w-10">
                  <input type="checkbox" @change="localStreams.forEach(s => s.enabled = ($event.target as HTMLInputElement).checked)">
                </th>
                <th class="px-5 py-3 text-left text-xs font-medium text-text-secondary uppercase">
                  Stream Name
                </th>
                <th class="px-5 py-3 text-left text-xs font-medium text-text-secondary uppercase">
                  Fields
                </th>
                <th class="px-5 py-3 text-left text-xs font-medium text-text-secondary uppercase">
                  Primary Key
                </th>
                <th class="px-5 py-3 text-left text-xs font-medium text-text-secondary uppercase">
                  Cursor
                </th>
                <th class="px-5 py-3 w-10" />
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              <template v-for="group in groupedStreams" :key="group.namespace">
                <!-- Namespace header (only when multiple namespaces) -->
                <tr
                  v-if="hasMultipleNamespaces"
                  class="cursor-pointer hover:bg-surface-hover transition-colors bg-surface-raised/50"
                  @click="toggleNamespace(group.namespace)"
                >
                  <td colspan="6" class="px-5 py-2">
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
                  <template v-for="{ stream: s, index: i } in group.streams" :key="streamKey(s)">
                    <!-- Collapsed row -->
                    <tr
                      class="cursor-pointer hover:bg-surface-hover transition-colors"
                      :class="{ 'opacity-50': !s.enabled }"
                      @click="toggleExpand(s)"
                    >
                      <td class="px-5 py-3" @click.stop>
                        <input v-model="s.enabled" type="checkbox">
                      </td>
                      <td class="px-5 py-3 text-sm font-semibold text-heading">
                        <span class="inline-flex items-center gap-1.5">
                          {{ displayStreamName(s) }}
                          <AlertCircle v-if="hasStreamErrors(s)" class="w-4 h-4 text-danger" title="This stream has validation errors" />
                        </span>
                      </td>
                      <td class="px-5 py-3">
                        <SBadge variant="gray">
                          {{ selectedFieldCount(s) }}/{{ totalFieldCount(s) }} fields
                        </SBadge>
                      </td>
                      <td class="px-5 py-3">
                        <SBadge variant="gray">
                          {{ pkSummary(s) }}
                        </SBadge>
                      </td>
                      <td class="px-5 py-3">
                        <SBadge v-if="s.syncMode === 'incremental'" variant="gray">
                          {{ cursorSummary(s) }}
                        </SBadge>
                      </td>
                      <td class="px-5 py-3">
                        <ChevronUp v-if="expandedStream === streamKey(s)" class="w-4 h-4" />
                        <ChevronDown v-else class="w-4 h-4" />
                      </td>
                    </tr>
                    <!-- Expanded row -->
                    <tr v-if="expandedStream === streamKey(s)">
                      <td colspan="6">
                        <StreamExpandedRow
                          :stream="s"
                          :schema="s.jsonSchema"
                          :validation-errors="errorsForStream(s)"
                          connection-id=""
                          :state-data="undefined"
                          :state-updated-at="undefined"
                          @update:stream="updateStream(i, $event)"
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
    </div>
  </div>
</template>
