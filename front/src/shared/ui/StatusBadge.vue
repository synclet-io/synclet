<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status: string
}>()

const colorClass = computed(() => {
  switch (props.status.toLowerCase()) {
    case 'active':
    case 'completed':
    case 'running':
    case 'starting':
      return 'bg-success-50 text-success-600 dark:bg-success-500/10 dark:text-success-500'
    case 'failed':
    case 'error':
      return 'bg-danger-50 text-danger-600 dark:bg-danger-500/10 dark:text-danger-500'
    case 'scheduled':
    case 'paused':
      return 'bg-warning-50 text-warning-600 dark:bg-warning-500/10 dark:text-warning-500'
    case 'inactive':
    case 'cancelled':
      return 'bg-gray-100 text-gray-600 dark:bg-gray-500/10 dark:text-gray-400'
    default:
      return 'bg-gray-100 text-gray-600'
  }
})
</script>

<template>
  <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium" :class="colorClass">
    {{ status }}
  </span>
</template>
