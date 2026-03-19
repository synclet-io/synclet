<script setup lang="ts">
import type { SchemaField } from './schemaParser'
import { SBadge } from '@shared/ui'
import { Lock } from 'lucide-vue-next'
import { computed } from 'vue'
import { getLeafFields, pathKey } from './schemaParser'

const props = withDefaults(defineProps<{
  field: SchemaField
  selectedPaths: Set<string>
  forcedPaths: Set<string>
  searchQuery: string
  depth?: number
  primaryKey: string[][]
  cursorField: string[]
  sourceDefinedPrimaryKey: string[][]
  sourceDefinedCursor: boolean
  defaultCursorField: string[]
  showCursorColumn: boolean
}>(), {
  depth: 0,
})

const emit = defineEmits<{
  toggle: [path: string[], selected: boolean]
  togglePk: [path: string[], selected: boolean]
  toggleCursor: [path: string[]]
}>()

const key = computed(() => pathKey(props.field.path))

const isLeaf = computed(() => !props.field.children || props.field.children.length === 0)

const isForced = computed(() => props.forcedPaths.has(key.value))

const isChecked = computed(() => {
  if (isForced.value)
    return true
  if (isLeaf.value)
    return props.selectedPaths.has(key.value)
  // Parent: checked if all leaf descendants are selected or forced
  const leaves = getLeafFields([props.field])
  return leaves.length > 0 && leaves.every(l => props.selectedPaths.has(pathKey(l.path)) || props.forcedPaths.has(pathKey(l.path)))
})

const isIndeterminate = computed(() => {
  if (isLeaf.value)
    return false
  const leaves = getLeafFields([props.field])
  if (leaves.length === 0)
    return false
  const checkedCount = leaves.filter(l => props.selectedPaths.has(pathKey(l.path)) || props.forcedPaths.has(pathKey(l.path))).length
  return checkedCount > 0 && checkedCount < leaves.length
})

// PK state for this field
const hasSourceDefinedPk = computed(() => props.sourceDefinedPrimaryKey.length > 0)

const isSourceDefinedPk = computed(() => {
  if (!isLeaf.value)
    return false
  return props.sourceDefinedPrimaryKey.some(pk => pathKey(pk) === key.value)
})

const isPkSelected = computed(() => {
  if (!isLeaf.value)
    return false
  if (isSourceDefinedPk.value)
    return true
  return props.primaryKey.some(pk => pathKey(pk) === key.value)
})

// Cursor state for this field
const isSourceDefinedCursorField = computed(() => {
  if (!isLeaf.value)
    return false
  return props.sourceDefinedCursor && pathKey(props.defaultCursorField) === key.value
})

const isCursorSelected = computed(() => {
  if (!isLeaf.value)
    return false
  if (isSourceDefinedCursorField.value)
    return true
  return pathKey(props.cursorField) === key.value
})

const matchesSearch = computed(() => {
  if (!props.searchQuery)
    return true
  const q = props.searchQuery.toLowerCase()
  const fullPath = props.field.path.join('.').toLowerCase()
  if (fullPath.includes(q))
    return true
  // Show parent if any child matches
  if (props.field.children) {
    return hasMatchingDescendant(props.field.children, q)
  }
  return false
})

function hasMatchingDescendant(fields: SchemaField[], query: string): boolean {
  for (const f of fields) {
    if (f.path.join('.').toLowerCase().includes(query))
      return true
    if (f.children && hasMatchingDescendant(f.children, query))
      return true
  }
  return false
}

function handleToggle() {
  if (isForced.value)
    return

  if (isLeaf.value) {
    emit('toggle', props.field.path, !isChecked.value)
  }
  else {
    // Parent: toggle all descendant leaves
    const leaves = getLeafFields([props.field])
    const selecting = !isChecked.value
    for (const leaf of leaves) {
      const lk = pathKey(leaf.path)
      if (props.forcedPaths.has(lk))
        continue // Don't toggle forced fields
      emit('toggle', leaf.path, selecting)
    }
  }
}

function handlePkToggle() {
  if (isSourceDefinedPk.value)
    return
  emit('togglePk', props.field.path, !isPkSelected.value)
}

function handleCursorToggle() {
  if (isSourceDefinedCursorField.value)
    return
  emit('toggleCursor', props.field.path)
}

function setIndeterminate(el: HTMLInputElement | null) {
  if (el)
    el.indeterminate = isIndeterminate.value
}
</script>

<template>
  <div v-if="matchesSearch">
    <div
      class="flex items-center gap-2 h-8"
      :style="{ paddingLeft: `${12 + depth * 16}px`, paddingRight: '12px' }"
    >
      <!-- Field selection checkbox -->
      <div class="w-6 shrink-0 flex items-center">
        <input
          :ref="(el: any) => setIndeterminate(el)"
          type="checkbox"
          :checked="isChecked"
          :disabled="isForced"
          class="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20 cursor-pointer"
          @change="handleToggle"
        >
      </div>
      <span class="flex-1 text-sm text-heading truncate">{{ field.name }}</span>

      <!-- Type badge -->
      <span class="w-14 flex justify-center">
        <SBadge variant="gray">{{ field.type }}</SBadge>
      </span>

      <!-- PK column -->
      <span class="w-10 flex justify-center">
        <template v-if="isLeaf">
          <span v-if="hasSourceDefinedPk && isSourceDefinedPk" class="inline-flex items-center gap-0.5" title="Source-defined primary key">
            <input type="checkbox" checked disabled class="w-3.5 h-3.5 rounded border-border text-primary">
            <Lock class="w-3 h-3 text-text-muted" />
          </span>
          <input
            v-else-if="hasSourceDefinedPk"
            type="checkbox"
            :checked="false"
            disabled
            class="w-3.5 h-3.5 rounded border-border text-text-muted opacity-40"
            title="Primary key is source-defined"
          >
          <input
            v-else
            type="checkbox"
            :checked="isPkSelected"
            class="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20 cursor-pointer"
            title="Toggle primary key"
            @change="handlePkToggle"
          >
        </template>
      </span>

      <!-- Cursor column -->
      <span v-if="showCursorColumn" class="w-10 flex justify-center">
        <template v-if="isLeaf">
          <span v-if="isSourceDefinedCursorField" class="inline-flex items-center gap-0.5" title="Source-defined cursor">
            <input type="radio" checked disabled class="w-3.5 h-3.5 border-border text-primary">
            <Lock class="w-3 h-3 text-text-muted" />
          </span>
          <input
            v-else
            type="radio"
            :checked="isCursorSelected"
            name="cursor-field"
            class="w-3.5 h-3.5 border-border text-primary focus:ring-primary/20 cursor-pointer"
            title="Select as cursor field"
            @change="handleCursorToggle"
          >
        </template>
      </span>
    </div>

    <div v-if="depth >= 9 && field.children && field.children.length > 0">
      <p class="text-xs text-text-muted" :style="{ paddingLeft: `${(depth + 1) * 16}px` }">
        Nested fields beyond depth 10 are not shown
      </p>
    </div>

    <template v-else-if="field.children">
      <FieldTreeNode
        v-for="child in field.children"
        :key="pathKey(child.path)"
        :field="child"
        :selected-paths="selectedPaths"
        :forced-paths="forcedPaths"
        :search-query="searchQuery"
        :depth="depth + 1"
        :primary-key="primaryKey"
        :cursor-field="cursorField"
        :source-defined-primary-key="sourceDefinedPrimaryKey"
        :source-defined-cursor="sourceDefinedCursor"
        :default-cursor-field="defaultCursorField"
        :show-cursor-column="showCursorColumn"
        @toggle="(path, selected) => emit('toggle', path, selected)"
        @toggle-pk="(path, selected) => emit('togglePk', path, selected)"
        @toggle-cursor="(path) => emit('toggleCursor', path)"
      />
    </template>
  </div>
</template>
