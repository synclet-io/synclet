<script setup lang="ts">
import { formatTrend } from '@shared/lib/format'
import { ArrowDown, ArrowUp, Minus } from 'lucide-vue-next'
import { computed } from 'vue'

const props = defineProps<{
  delta: number
  lowerIsBetter?: boolean
}>()

const trend = computed(() => formatTrend(props.delta))

const icon = computed(() => ({ up: ArrowUp, down: ArrowDown, flat: Minus })[trend.value.direction])

const colorClass = computed(() => {
  const { direction } = trend.value
  if (direction === 'flat')
    return 'text-text-muted'
  return (direction === 'up') !== props.lowerIsBetter ? 'text-success' : 'text-danger'
})
</script>

<template>
  <p class="inline-flex items-center gap-1 text-xs mt-1" :class="colorClass">
    <component :is="icon" class="w-3 h-3" aria-hidden="true" />
    <span v-if="trend.direction !== 'flat'" class="sr-only">{{ trend.direction === 'up' ? 'Up' : 'Down' }}</span>
    {{ trend.text }}
  </p>
</template>
