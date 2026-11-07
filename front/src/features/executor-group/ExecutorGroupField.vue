<script setup lang="ts">
import { SAlert, SSelect } from '@shared/ui'
import { computed } from 'vue'
import { useExecutorGroups } from './useExecutorGroups'

const props = defineProps<{
  modelValue: string
  disabled?: boolean
  exclude?: string
}>()

defineEmits<{
  'update:modelValue': [value: string]
}>()

const { groups, onlyDefault } = useExecutorGroups()

const options = computed(() => groups.value
  .filter(g => g.name !== props.exclude)
  .map(g => ({ value: g.name, label: g.liveExecutors > 0 ? g.name : `${g.name} (no live executors)` })))

const selectedWithoutExecutors = computed(() => groups.value.find(g => g.name === props.modelValue)?.liveExecutors === 0)
</script>

<template>
  <div v-if="!onlyDefault" class="space-y-2">
    <SSelect
      label="Executor group"
      tooltip="Executors of this group run its checks, discovery and syncs."
      :options="options"
      :model-value="modelValue"
      :disabled="disabled"
      @update:model-value="$emit('update:modelValue', String($event))"
    />
    <SAlert v-if="selectedWithoutExecutors" variant="warning">
      No live executor in group "{{ modelValue }}": its syncs and checks will not run until an executor of this group connects.
    </SAlert>
    <slot />
  </div>
</template>
