<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'text' | 'circle' | 'rect'
  width?: string
  height?: string
  count?: number
}>(), {
  variant: 'text',
  count: 1,
})
</script>

<template>
  <div class="space-y-2.5">
    <div
      v-for="i in count"
      :key="i"
      class="animate-pulse bg-surface-raised rounded-lg"
      :class="{
        'h-4': variant === 'text',
        'rounded-full': variant === 'circle',
      }"
      :style="{
        width: variant === 'circle' ? (width || '40px') : (width || '100%'),
        height: variant === 'text' ? (height || '16px') : variant === 'circle' ? (width || '40px') : (height || '80px'),
      }"
    />
  </div>
</template>
