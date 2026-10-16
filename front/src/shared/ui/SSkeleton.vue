<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  variant?: 'text' | 'circle' | 'rect'
  width?: string
  height?: string
  count?: number
}>(), {
  variant: 'text',
  count: 1,
})

const itemClass = computed(() => [
  'animate-pulse bg-surface-raised',
  `s-skeleton-${props.variant}`,
  props.variant === 'circle' ? 'rounded-full' : 'rounded-lg',
])

const itemStyle = computed(() => ({
  width: props.width,
  height: props.variant === 'circle' ? (props.height ?? props.width) : props.height,
}))
</script>

<template>
  <div v-if="count === 1" :class="itemClass" :style="itemStyle" />
  <div v-else class="space-y-2.5">
    <div v-for="i in count" :key="i" :class="itemClass" :style="itemStyle" />
  </div>
</template>

<style>
:where(.s-skeleton-text) {
  height: 1rem;
}

:where(.s-skeleton-rect) {
  height: 5rem;
}

:where(.s-skeleton-circle) {
  width: 2.5rem;
  height: 2.5rem;
}
</style>
