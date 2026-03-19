<script setup lang="ts">
import type { Tab } from './types'
import { RouterLink } from 'vue-router'

defineProps<{
  tabs: Tab[]
  modelValue?: string
  variant?: 'underline' | 'pills'
}>()

defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <!-- Pills variant -->
  <nav v-if="variant === 'pills'" class="inline-flex gap-1 p-1 bg-surface-raised rounded-lg mb-6">
    <template v-for="tab in tabs" :key="tab.name">
      <RouterLink
        v-if="tab.to"
        :to="tab.to"
        class="px-3 py-1.5 text-sm font-medium rounded-md transition-colors whitespace-nowrap text-text-secondary hover:text-heading"
        active-class="bg-surface text-heading shadow-sm"
      >
        {{ tab.name }}
      </RouterLink>
      <button
        v-else
        class="px-3 py-1.5 text-sm font-medium rounded-md transition-colors whitespace-nowrap"
        :class="modelValue === (tab.value || tab.name)
          ? 'bg-surface text-heading shadow-sm'
          : 'text-text-secondary hover:text-heading'"
        @click="$emit('update:modelValue', tab.value || tab.name)"
      >
        {{ tab.name }}
      </button>
    </template>
  </nav>

  <!-- Underline variant (default) -->
  <nav v-else class="flex gap-1 border-b border-border mb-6">
    <template v-for="tab in tabs" :key="tab.name">
      <RouterLink
        v-if="tab.to"
        :to="tab.to"
        class="px-4 py-2.5 text-sm font-medium -mb-px border-b-2 transition-colors whitespace-nowrap border-transparent text-text-secondary hover:text-heading hover:border-border"
        active-class="border-primary text-primary"
      >
        {{ tab.name }}
      </RouterLink>
      <button
        v-else
        class="px-4 py-2.5 text-sm font-medium -mb-px border-b-2 transition-colors whitespace-nowrap"
        :class="modelValue === (tab.value || tab.name)
          ? 'border-primary text-primary'
          : 'border-transparent text-text-secondary hover:text-heading hover:border-border'"
        @click="$emit('update:modelValue', tab.value || tab.name)"
      >
        {{ tab.name }}
      </button>
    </template>
  </nav>
</template>
