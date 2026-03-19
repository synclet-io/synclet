<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'

const props = defineProps<{
  label: string
  value: string | number
  icon?: any
  to?: string
  color?: 'blue' | 'green' | 'amber' | 'purple'
}>()

const iconBg = computed(() => {
  const map: Record<string, string> = {
    blue: 'bg-blue-50 text-blue-600',
    green: 'bg-green-50 text-green-600',
    amber: 'bg-amber-50 text-amber-600',
    purple: 'bg-purple-50 text-purple-600',
  }
  return map[props.color || 'blue']
})
</script>

<template>
  <component
    :is="to ? RouterLink : 'div'"
    :to="to"
    class="bg-surface border border-border rounded-xl p-5 transition-all duration-200"
    :class="to ? 'hover:shadow-raised hover:border-border cursor-pointer group' : 'shadow-soft'"
  >
    <div class="flex items-start justify-between">
      <div>
        <p class="text-sm text-text-secondary">
          {{ label }}
        </p>
        <p class="text-2xl font-semibold text-heading mt-1 tracking-tight">
          {{ value }}
        </p>
        <slot />
      </div>
      <div v-if="icon" class="w-10 h-10 rounded-lg flex items-center justify-center" :class="iconBg">
        <component :is="icon" class="w-5 h-5" />
      </div>
    </div>
  </component>
</template>
