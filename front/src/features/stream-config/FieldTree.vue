<script setup lang="ts">
import type { SyncMode } from '@entities/connection'
import type { SchemaField } from './schemaParser'
import { Search } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import FieldTreeNode from './FieldTreeNode.vue'
import { getLeafFields, pathKey } from './schemaParser'

const props = defineProps<{
  fields: SchemaField[]
  selectedPaths: Set<string>
  forcedPaths: Set<string>
  primaryKey: string[][]
  cursorField: string[]
  sourceDefinedPrimaryKey: string[][]
  sourceDefinedCursor: boolean
  defaultCursorField: string[]
  syncMode: SyncMode
}>()

const emit = defineEmits<{
  toggle: [path: string[], selected: boolean]
  selectAll: []
  selectNone: []
  togglePk: [path: string[], selected: boolean]
  toggleCursor: [path: string[]]
}>()

const rawSearch = ref('')
const searchQuery = ref('')

let debounceTimer: ReturnType<typeof setTimeout> | undefined

watch(rawSearch, (val) => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    searchQuery.value = val
  }, 150)
})

const allSelected = computed(() => {
  const leafs = getLeafFields(props.fields)
  return leafs.length > 0 && leafs.every(f => props.selectedPaths.has(pathKey(f.path)))
})

const someSelected = computed(() => {
  const leafs = getLeafFields(props.fields)
  return leafs.some(f => props.selectedPaths.has(pathKey(f.path)))
})

function handleSelectAllChange(event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  if (checked) {
    emit('selectAll')
  }
  else {
    emit('selectNone')
  }
}
</script>

<template>
  <div>
    <!-- Search bar -->
    <div class="flex items-center gap-3 mb-3">
      <div class="relative flex-1">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
        <input
          v-model="rawSearch"
          type="text"
          placeholder="Search fields..."
          class="w-full h-9 pl-9 pr-3 border border-border rounded-lg text-sm bg-surface text-heading placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
        >
      </div>
    </div>

    <!-- Header row with select-all checkbox -->
    <div class="flex items-center gap-2 h-8 px-3 bg-surface-raised border-b border-border text-xs font-medium text-text-secondary uppercase">
      <div class="w-6 shrink-0">
        <input
          type="checkbox"
          :checked="allSelected"
          :indeterminate="someSelected && !allSelected"
          aria-label="Select all fields"
          class="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20 cursor-pointer"
          @change="handleSelectAllChange"
        >
      </div>
      <div class="flex-1">
        Field
      </div>
      <div class="w-14 text-center">
        Type
      </div>
      <div class="w-10 text-center">
        PK
      </div>
      <div v-if="syncMode === 'incremental'" class="w-10 text-center">
        Cursor
      </div>
    </div>

    <!-- Field tree -->
    <div class="max-h-80 overflow-y-auto">
      <FieldTreeNode
        v-for="field in fields"
        :key="pathKey(field.path)"
        :field="field"
        :selected-paths="selectedPaths"
        :forced-paths="forcedPaths"
        :search-query="searchQuery"
        :primary-key="primaryKey"
        :cursor-field="cursorField"
        :source-defined-primary-key="sourceDefinedPrimaryKey"
        :source-defined-cursor="sourceDefinedCursor"
        :default-cursor-field="defaultCursorField"
        :show-cursor-column="syncMode === 'incremental'"
        @toggle="(path, selected) => emit('toggle', path, selected)"
        @toggle-pk="(path, selected) => emit('togglePk', path, selected)"
        @toggle-cursor="(path) => emit('toggleCursor', path)"
      />
    </div>
  </div>
</template>
