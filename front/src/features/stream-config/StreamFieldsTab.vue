<script setup lang="ts">
import type { DestinationSyncMode, SelectedField, SyncMode } from '@entities/connection'
import type { SchemaField } from './schemaParser'
import type { StreamValidationError } from './useStreamValidation'
import { computed } from 'vue'
import FieldTree from './FieldTree.vue'
import { getLeafFields, parseJsonSchema, pathKey } from './schemaParser'

export interface StreamRow {
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

const props = defineProps<{
  stream: StreamRow
  schema: Record<string, any>
  validationErrors: StreamValidationError[]
}>()

const emit = defineEmits<{
  'update:stream': [stream: StreamRow]
}>()

const parsedFields = computed<SchemaField[]>(() => {
  if (!props.schema)
    return []
  return parseJsonSchema(props.schema)
})

const leafFields = computed(() => getLeafFields(parsedFields.value))

function pathMap(paths: string[][]): Map<string, string[]> {
  return new Map(paths.map(p => [pathKey(p), p]))
}

const selectedPathMap = computed(() => pathMap(props.stream.selectedFields.map(sf => sf.fieldPath)))

const forcedPathMap = computed(() => {
  const pkFields = props.stream.sourceDefinedPrimaryKey.length > 0
    ? props.stream.sourceDefinedPrimaryKey
    : props.stream.primaryKey
  const cursor = props.stream.sourceDefinedCursor
    ? props.stream.defaultCursorField
    : props.stream.cursorField
  return pathMap(cursor.length > 0 ? [...pkFields, cursor] : pkFields)
})

const selectedPaths = computed(() => new Set(selectedPathMap.value.keys()))
const forcedPaths = computed(() => new Set(forcedPathMap.value.keys()))

const hasFieldsError = computed(() => props.validationErrors.some(e => e.type === 'missing_fields'))

function emitUpdate(partial: Partial<StreamRow>) {
  emit('update:stream', { ...props.stream, ...partial })
}

function emitSelected(paths: Map<string, string[]>) {
  for (const [key, path] of forcedPathMap.value)
    paths.set(key, path)
  const selectedFields: SelectedField[] = Array.from(paths.values(), fieldPath => ({ fieldPath }))
  emitUpdate({ selectedFields })
}

function handleFieldToggle(path: string[], selected: boolean) {
  const current = new Map(selectedPathMap.value)
  if (selected)
    current.set(pathKey(path), path)
  else
    current.delete(pathKey(path))
  emitSelected(current)
}

function handleSelectAll() {
  emitSelected(pathMap(leafFields.value.map(f => f.path)))
}

function handleSelectNone() {
  emitSelected(new Map())
}

function handleTogglePk(path: string[], selected: boolean) {
  const key = pathKey(path)
  let updated: string[][]
  if (selected) {
    updated = [...props.stream.primaryKey, path]
  }
  else {
    updated = props.stream.primaryKey.filter(pk => pathKey(pk) !== key)
  }
  emitUpdate({ primaryKey: updated })
}

function handleToggleCursor(path: string[]) {
  emitUpdate({ cursorField: path })
}
</script>

<template>
  <div class="p-4">
    <!-- Inline validation errors summary -->
    <div v-if="validationErrors.length > 0" class="mb-3 space-y-1">
      <p v-for="err in validationErrors" :key="err.type" class="text-xs text-danger flex items-center gap-1.5">
        <span class="w-1 h-1 rounded-full bg-danger flex-shrink-0" />
        {{ err.message }}
      </p>
    </div>

    <!-- Field tree with PK/Cursor columns -->
    <div>
      <p v-if="hasFieldsError" class="text-xs text-danger mb-2">
        Select at least one field for replication
      </p>
      <FieldTree
        :fields="parsedFields"
        :selected-paths="selectedPaths"
        :forced-paths="forcedPaths"
        :primary-key="stream.primaryKey"
        :cursor-field="stream.cursorField"
        :source-defined-primary-key="stream.sourceDefinedPrimaryKey"
        :source-defined-cursor="stream.sourceDefinedCursor"
        :default-cursor-field="stream.defaultCursorField"
        :sync-mode="stream.syncMode"
        @toggle="handleFieldToggle"
        @select-all="handleSelectAll"
        @select-none="handleSelectNone"
        @toggle-pk="handleTogglePk"
        @toggle-cursor="handleToggleCursor"
      />
    </div>
  </div>
</template>
