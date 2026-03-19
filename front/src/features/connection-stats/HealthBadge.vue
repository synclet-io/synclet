<script setup lang="ts">
import type { HealthStatus } from '@entities/stats'

defineProps<{
  health: HealthStatus
}>()

const config: Record<HealthStatus, { bg: string, dot: string, label: string }> = {
  healthy: { bg: 'bg-green-100 text-green-700', dot: 'bg-green-500', label: 'Healthy' },
  warning: { bg: 'bg-amber-100 text-amber-700', dot: 'bg-amber-500', label: 'Warning' },
  failing: { bg: 'bg-red-100 text-red-700', dot: 'bg-red-500', label: 'Failing' },
  disabled: { bg: 'bg-slate-100 text-slate-500', dot: 'bg-slate-400', label: 'Disabled' },
}
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium"
    :class="config[health].bg"
  >
    <span class="w-1.5 h-1.5 rounded-full" :class="config[health].dot" />
    {{ config[health].label }}
  </span>
</template>
