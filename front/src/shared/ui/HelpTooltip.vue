<script setup lang="ts">
import { HelpCircle } from 'lucide-vue-next'
import { ref } from 'vue'

defineProps<{
  text: string
}>()

const open = ref(false)

function toggle() {
  open.value = !open.value
}
</script>

<template>
  <span class="relative inline-flex">
    <button
      type="button"
      class="inline-flex items-center text-text-muted hover:text-text-primary transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary rounded-full"
      :aria-label="`Help: ${text}`"
      :aria-expanded="open"
      @mouseenter="open = true"
      @mouseleave="open = false"
      @focus="open = true"
      @blur="open = false"
      @click.prevent="toggle"
    >
      <HelpCircle class="w-3.5 h-3.5" />
    </button>
    <span
      v-if="open"
      role="tooltip"
      class="absolute left-1/2 -translate-x-1/2 bottom-full mb-2 z-30 w-64 px-3 py-2 rounded-lg bg-slate-900 text-white text-xs leading-snug shadow-overlay"
    >
      {{ text }}
    </span>
  </span>
</template>
