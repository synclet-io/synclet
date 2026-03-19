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

const selectedPaths = computed(() => {
  const set = new Set<string>()
  for (const sf of props.stream.selectedFields) {
    set.add(pathKey(sf.fieldPath))
  }
  return set
})

const forcedPaths = computed(() => {
  const set = new Set<string>()
  // Force PK fields
  const pkFields = props.stream.sourceDefinedPrimaryKey.length > 0
    ? props.stream.sourceDefinedPrimaryKey
    : props.stream.primaryKey
  for (const pk of pkFields) {
    set.add(pathKey(pk))
  }
  // Force cursor field
  const cursor = props.stream.sourceDefinedCursor
    ? props.stream.defaultCursorField
    : props.stream.cursorField
  if (cursor.length > 0) {
    set.add(pathKey(cursor))
  }
  return set
})

const hasFieldsError = computed(() => props.validationErrors.some(e => e.type === 'missing_fields'))

function emitUpdate(partial: Partial<StreamRow>) {
  emit('update:stream', { ...props.stream, ...partial })
}

function handleFieldToggle(path: string[], selected: boolean) {
  const key = pathKey(path)
  const current = new Set(selectedPaths.value)
  if (selected) {
    current.add(key)
  }
  else {
    current.delete(key)
  }
  // Also ensure forced paths are always included
  for (const fp of forcedPaths.value) {
    current.add(fp)
  }
  const newFields: SelectedField[] = Array.from(current).map(k => ({ fieldPath: k.split('.') }))
  emitUpdate({ selectedFields: newFields })
}

function handleSelectAll() {
  const allPaths = new Set(leafFields.value.map(f => pathKey(f.path)))
  for (const fp of forcedPaths.value) {
    allPaths.add(fp)
  }
  const newFields: SelectedField[] = Array.from(allPaths).map(k => ({ fieldPath: k.split('.') }))
  emitUpdate({ selectedFields: newFields })
}

function handleSelectNone() {
  // Keep only forced paths
  const newFields: SelectedField[] = Array.from(forcedPaths.value).map(k => ({ fieldPath: k.split('.') }))
  emitUpdate({ selectedFields: newFields })
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
