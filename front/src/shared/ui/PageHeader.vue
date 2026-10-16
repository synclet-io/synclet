<script setup lang="ts">
import { ArrowLeft } from 'lucide-vue-next'
import { RouterLink } from 'vue-router'

withDefaults(defineProps<{
  title: string
  description?: string
  backTo?: string | object
  backLabel?: string
  level?: 1 | 2
}>(), { level: 1 })
</script>

<template>
  <div class="mb-6 sm:mb-8">
    <div v-if="backTo" class="mb-2">
      <RouterLink :to="backTo" class="inline-flex items-center gap-1.5 text-sm text-text-secondary hover:text-primary transition-colors">
        <ArrowLeft class="w-3.5 h-3.5" />
        {{ backLabel }}
      </RouterLink>
    </div>
    <slot name="breadcrumb" />
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <component
          :is="`h${level}`"
          class="font-semibold text-heading tracking-tight"
          :class="level === 1 ? 'text-xl' : 'text-lg'"
        >
          {{ title }}
        </component>
        <p v-if="description" class="mt-1 text-sm text-text-secondary">
          {{ description }}
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <slot name="actions" />
      </div>
    </div>
  </div>
</template>
