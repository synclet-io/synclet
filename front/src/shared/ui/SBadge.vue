<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  variant?: 'success' | 'danger' | 'warning' | 'gray' | 'info'
  dot?: boolean
  label?: string
}>(), {
  variant: 'gray',
  dot: false,
})

const colorClasses = computed(() => {
  const map: Record<string, string> = {
    success: 'bg-success-bg text-success',
    danger: 'bg-danger-bg text-danger',
    warning: 'bg-warning-bg text-warning',
    info: 'bg-info-bg text-info',
    gray: 'bg-surface-raised text-text-secondary',
  }
  return map[props.variant]
})

const dotColor = computed(() => {
  const map: Record<string, string> = {
    success: 'bg-success',
    danger: 'bg-danger',
    warning: 'bg-warning',
    info: 'bg-info',
    gray: 'bg-text-muted',
  }
  return map[props.variant]
})
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-xs font-medium"
    :class="colorClasses"
  >
    <span v-if="dot" class="w-1.5 h-1.5 rounded-full" :class="dotColor" />
    <slot>{{ label }}</slot>
  </span>
</template>
