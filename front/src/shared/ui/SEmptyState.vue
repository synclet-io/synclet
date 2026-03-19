<script setup lang="ts">
import { Inbox } from 'lucide-vue-next'

defineProps<{
  title?: string
  description?: string
  icon?: any
}>()
</script>

<template>
  <div class="flex flex-col items-center justify-center text-center py-10">
    <div class="w-12 h-12 rounded-xl bg-surface-raised flex items-center justify-center mb-4">
      <component :is="icon || Inbox" class="w-6 h-6 text-text-muted" />
    </div>
    <h3 v-if="title" class="text-sm font-medium text-heading mb-1">
      {{ title }}
    </h3>
    <p v-if="description" class="text-sm text-text-muted max-w-xs mb-4">
      {{ description }}
    </p>
    <slot />
  </div>
</template>
