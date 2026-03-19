<script setup lang="ts">
withDefaults(defineProps<{
  title?: string
  description?: string
  padding?: boolean
}>(), {
  padding: true,
})
</script>

<template>
  <div class="bg-surface border border-border rounded-xl shadow-soft overflow-hidden">
    <div v-if="title || $slots.header" class="px-6 py-4 border-b border-border flex items-center justify-between">
      <div v-if="title">
        <h3 class="text-[15px] font-semibold text-heading">
          {{ title }}
        </h3>
        <p v-if="description" class="text-sm text-text-secondary mt-0.5">
          {{ description }}
        </p>
      </div>
      <slot name="header" />
    </div>
    <div :class="padding ? 'p-6' : ''">
      <slot />
    </div>
    <div v-if="$slots.footer" class="px-6 py-4 border-t border-border bg-surface-raised">
      <slot name="footer" />
    </div>
  </div>
</template>
