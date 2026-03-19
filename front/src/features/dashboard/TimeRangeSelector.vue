<script setup lang="ts">
import type { TimeRange } from '@entities/stats'

defineProps<{
  modelValue: TimeRange
}>()

const emit = defineEmits<{
  'update:modelValue': [value: TimeRange]
}>()

const options: TimeRange[] = ['24h', '7d', '30d']
</script>

<template>
  <div class="inline-flex gap-1 bg-surface-raised rounded-lg p-1">
    <button
      v-for="opt in options"
      :key="opt"
      class="rounded-md px-3 py-1 text-sm font-medium transition-colors"
      :class="modelValue === opt
        ? 'bg-primary text-on-primary'
        : 'text-text-secondary hover:text-text-primary'"
      @click="emit('update:modelValue', opt)"
    >
      {{ opt }}
    </button>
  </div>
</template>
