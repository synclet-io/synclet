<script setup lang="ts">
import type { DestinationSyncMode, StreamState, SyncMode } from '@entities/connection'
import type { StreamRow } from './catalogStreams'
import type { StreamValidationError } from './useStreamValidation'
import { SBadge, SButton, SInput, SSelect } from '@shared/ui'
import { AlertCircle, ChevronDown, ChevronRight, ChevronUp } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { streamKey } from './catalogStreams'
import { getLeafFields, parseJsonSchema } from './schemaParser'
import { enableStream } from './streamDefaults'
import StreamExpandedRow from './StreamExpandedRow.vue'
import { useStreamValidation } from './useStreamValidation'

interface NamespaceGroup {
  namespace: string
  streams: StreamRow[]
}

const props = withDefaults(defineProps<{
  connectionId?: string
  streamStates?: StreamState[]
}>(), {
  connectionId: '',
  streamStates: () => [],
})

const streams = defineModel<StreamRow[]>('streams', { required: true })

const { validate: validateStreams } = useStreamValidation()
const searchQuery = ref('')
const expandedStream = ref<string | null>(null)
const validationErrors = ref<StreamValidationError[]>([])
const hasValidated = ref(false)
const expandedNamespaces = ref(new Set<string>())

watch(() => streams.value.map(s => s.namespace).join('\n'), () => {
  expandedNamespaces.value = new Set(streams.value.map(s => s.namespace))
}, { immediate: true })

watch(streams, (rows) => {
  if (hasValidated.value)
    validationErrors.value = validateStreams(rows)
}, { deep: true })

/** Validates every enabled stream, marks the invalid ones and opens the first of them. */
function validate(): StreamValidationError[] {
  hasValidated.value = true
  validationErrors.value = validateStreams(streams.value)
  if (validationErrors.value.length > 0)
    expandedStream.value = validationErrors.value[0].streamKey
  return validationErrors.value
}

defineExpose({ validate })

function withEnabled(s: StreamRow, enabled: boolean): StreamRow {
  if (s.enabled === enabled)
    return s
  if (enabled && !s.configured)
    return enableStream(s)
  return { ...s, enabled }
}

function setEnabled(s: StreamRow, enabled: boolean) {
  replaceStream(withEnabled(s, enabled))
}

function setStreamsEnabled(targets: StreamRow[], enabled: boolean) {
  const keys = new Set(targets.map(streamKey))
  streams.value = streams.value.map(s => keys.has(streamKey(s)) ? withEnabled(s, enabled) : s)
}

function allOn(rows: StreamRow[]): boolean {
  return rows.length > 0 && rows.every(s => s.enabled)
}

function partlyOn(rows: StreamRow[]): boolean {
  return rows.some(s => s.enabled) && !allOn(rows)
}

function replaceStream(updated: StreamRow) {
  const key = streamKey(updated)
  streams.value = streams.value.map(s => streamKey(s) === key ? updated : s)
}

function editStream(updated: StreamRow) {
  replaceStream({ ...updated, configured: true })
}

const groupedStreams = computed<NamespaceGroup[]>(() => {
  const groups = new Map<string, StreamRow[]>()
  for (const s of streams.value) {
    if (!groups.has(s.namespace))
      groups.set(s.namespace, [])
    groups.get(s.namespace)!.push(s)
  }
  const result: NamespaceGroup[] = []
  if (groups.has('')) {
    result.push({ namespace: '', streams: groups.get('')! })
    groups.delete('')
  }
  const sorted = Array.from(groups.entries()).sort(([a], [b]) => a.localeCompare(b))
  for (const [namespace, items] of sorted)
    result.push({ namespace, streams: items })
  return result
})

const filteredGroupedStreams = computed(() => {
  if (!searchQuery.value)
    return groupedStreams.value
  const q = searchQuery.value.toLowerCase()
  return groupedStreams.value
    .map(g => ({
      ...g,
      streams: g.streams.filter(s => s.name.toLowerCase().includes(q) || s.namespace.toLowerCase().includes(q)),
    }))
    .filter(g => g.streams.length > 0)
})

const visibleStreams = computed(() => filteredGroupedStreams.value.flatMap(g => g.streams))

const namespaceCounts = computed(() => {
  const counts = new Map<string, { enabled: number, total: number }>()
  for (const g of groupedStreams.value)
    counts.set(g.namespace, { enabled: g.streams.filter(s => s.enabled).length, total: g.streams.length })
  return counts
})

function namespaceLabel(ns: string): string {
  return ns || 'Default'
}

function toggleNamespace(ns: string) {
  const updated = new Set(expandedNamespaces.value)
  if (updated.has(ns))
    updated.delete(ns)
  else
    updated.add(ns)
  expandedNamespaces.value = updated
}

function isNamespaceExpanded(ns: string): boolean {
  return expandedNamespaces.value.has(ns)
}

function toggleExpand(s: StreamRow) {
  const key = streamKey(s)
  expandedStream.value = expandedStream.value === key ? null : key
}

function totalFieldCount(s: StreamRow): number {
  if (!s.jsonSchema)
    return 0
  return getLeafFields(parseJsonSchema(s.jsonSchema)).length
}

function errorsForStream(s: StreamRow): StreamValidationError[] {
  const key = streamKey(s)
  return validationErrors.value.filter(e => e.streamKey === key)
}

const validSyncCombinations: { src: SyncMode, dst: DestinationSyncMode, label: string }[] = [
  { src: 'incremental', dst: 'append_dedup', label: 'Incremental | Append + Dedup' },
  { src: 'incremental', dst: 'append', label: 'Incremental | Append' },
  { src: 'full_refresh', dst: 'overwrite', label: 'Full Refresh | Overwrite' },
  { src: 'full_refresh', dst: 'append', label: 'Full Refresh | Append' },
]

function syncModeOptions(s: StreamRow) {
  return validSyncCombinations
    .filter(c => s.supportedSyncModes.includes(c.src))
    .map(c => ({ value: `${c.src}|${c.dst}`, label: c.label }))
}

function applySyncMode(s: StreamRow, combined: string) {
  const [src, dst] = combined.split('|')
  editStream({ ...s, syncMode: src as SyncMode, destinationSyncMode: dst as DestinationSyncMode })
}

const bulkSelectKey = ref(0)
const bulkResult = ref('')

/** Sets one sync mode on every enabled stream the search shows, skipping those whose source does not support it. */
function applyBulkSyncMode(combined: string) {
  const [src, dst] = combined.split('|')
  const selected = visibleStreams.value.filter(s => s.enabled)
  const supported = new Set(selected.filter(s => s.supportedSyncModes.includes(src)).map(streamKey))
  streams.value = streams.value.map(s => supported.has(streamKey(s))
    ? { ...s, syncMode: src as SyncMode, destinationSyncMode: dst as DestinationSyncMode, configured: true }
    : s)
  const skipped = selected.length - supported.size
  const plural = (n: number) => n === 1 ? 'stream' : 'streams'
  const skippedNote = skipped > 0 ? `; ${skipped} ${plural(skipped)} skipped because the source does not support it` : ''
  bulkResult.value = `Sync mode set on ${supported.size} ${plural(supported.size)}${skippedNote}`
  bulkSelectKey.value++
}

function streamState(s: StreamRow): StreamState | undefined {
  return props.streamStates.find(st => st.streamNamespace === s.namespace && st.streamName === s.name)
}
</script>

<template>
  <div class="bg-surface border border-border rounded-xl overflow-hidden">
    <div class="p-3 border-b border-border flex flex-wrap items-center gap-3">
      <SInput v-model="searchQuery" placeholder="Search streams..." class="max-w-sm flex-1 min-w-48" />
      <SSelect
        :key="bulkSelectKey"
        size="sm"
        model-value=""
        placeholder="Set sync mode of selected streams"
        :options="validSyncCombinations.map(c => ({ value: `${c.src}|${c.dst}`, label: c.label }))"
        :disabled="!visibleStreams.some(s => s.enabled)"
        aria-label="Sync mode of selected streams"
        class="w-64"
        @update:model-value="applyBulkSyncMode($event as string)"
      />
      <span v-if="bulkResult" role="status" class="text-xs text-text-secondary">{{ bulkResult }}</span>
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
                :checked="allOn(visibleStreams)"
                :indeterminate="partlyOn(visibleStreams)"
                :disabled="visibleStreams.length === 0"
                @change="setStreamsEnabled(visibleStreams, ($event.target as HTMLInputElement).checked)"
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
            <tr
              :data-testid="`namespace-${group.namespace}`"
              class="cursor-pointer hover:bg-surface-hover transition-colors bg-surface-raised/50"
              @click="toggleNamespace(group.namespace)"
            >
              <td class="px-3 py-2 w-10" @click.stop>
                <input
                  type="checkbox"
                  class="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20 cursor-pointer"
                  :aria-label="`Enable all streams of namespace ${namespaceLabel(group.namespace)}`"
                  :checked="allOn(group.streams)"
                  :indeterminate="partlyOn(group.streams)"
                  @change="setStreamsEnabled(group.streams, ($event.target as HTMLInputElement).checked)"
                >
              </td>
              <td colspan="4" class="px-3 py-2">
                <button
                  type="button"
                  class="flex items-center gap-2 rounded focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
                  :aria-expanded="isNamespaceExpanded(group.namespace)"
                  @click.stop="toggleNamespace(group.namespace)"
                >
                  <ChevronDown v-if="isNamespaceExpanded(group.namespace)" class="w-4 h-4 text-text-secondary" />
                  <ChevronRight v-else class="w-4 h-4 text-text-secondary" />
                  <span class="font-semibold text-sm text-heading">
                    {{ namespaceLabel(group.namespace) }}
                  </span>
                  <SBadge variant="gray">
                    {{ namespaceCounts.get(group.namespace)?.enabled ?? 0 }}/{{ namespaceCounts.get(group.namespace)?.total ?? 0 }} enabled
                  </SBadge>
                </button>
              </td>
            </tr>

            <template v-if="isNamespaceExpanded(group.namespace)">
              <template v-for="s in group.streams" :key="streamKey(s)">
                <tr
                  class="cursor-pointer hover:bg-surface-hover transition-colors h-10"
                  :class="{ 'opacity-50': !s.enabled }"
                  @click="toggleExpand(s)"
                >
                  <td class="px-3 py-2 w-10" @click.stop>
                    <input
                      type="checkbox"
                      :checked="s.enabled"
                      :aria-label="`Enable stream ${s.name}`"
                      class="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20 cursor-pointer"
                      @change="setEnabled(s, ($event.target as HTMLInputElement).checked)"
                    >
                  </td>
                  <td class="px-3 py-2 text-sm font-medium text-heading">
                    {{ s.name }}
                    <AlertCircle v-if="errorsForStream(s).length > 0" class="inline w-4 h-4 text-danger ml-1" />
                  </td>
                  <td class="px-3 py-2" @click.stop>
                    <SSelect
                      size="sm"
                      :model-value="`${s.syncMode}|${s.destinationSyncMode}`"
                      :options="syncModeOptions(s)"
                      :aria-label="`Sync mode of stream ${s.name}`"
                      @update:model-value="applySyncMode(s, $event as string)"
                    />
                  </td>
                  <td class="px-3 py-2">
                    <span class="text-xs text-text-muted">{{ s.selectedFields.length }}/{{ totalFieldCount(s) }} fields</span>
                  </td>
                  <td class="px-3 py-2 w-10" @click.stop>
                    <SButton
                      variant="ghost"
                      size="icon"
                      :aria-label="`${expandedStream === streamKey(s) ? 'Collapse' : 'Expand'} stream ${s.name}`"
                      :aria-expanded="expandedStream === streamKey(s)"
                      @click="toggleExpand(s)"
                    >
                      <ChevronUp v-if="expandedStream === streamKey(s)" class="w-4 h-4" aria-hidden="true" />
                      <ChevronDown v-else class="w-4 h-4" aria-hidden="true" />
                    </SButton>
                  </td>
                </tr>
                <tr v-if="expandedStream === streamKey(s)">
                  <td colspan="5">
                    <StreamExpandedRow
                      :stream="s"
                      :schema="s.jsonSchema"
                      :validation-errors="errorsForStream(s)"
                      :connection-id="connectionId"
                      :state-data="streamState(s)?.stateData"
                      :state-updated-at="streamState(s)?.updatedAt"
                      @update:stream="editStream"
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
</template>
