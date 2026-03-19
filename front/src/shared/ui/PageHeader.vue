<script setup lang="ts">
import { ArrowLeft } from 'lucide-vue-next'
import { RouterLink } from 'vue-router'

defineProps<{
  title: string
  description?: string
  backTo?: string | object
  backLabel?: string
}>()
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
        <h1 class="text-xl font-semibold text-heading tracking-tight">
          {{ title }}
        </h1>
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
