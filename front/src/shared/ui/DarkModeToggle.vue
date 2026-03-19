<script setup lang="ts">
import type { Theme } from '@shared/lib/darkMode'
import { useDarkMode } from '@shared/lib/darkMode'
import { Monitor, Moon, Sun } from 'lucide-vue-next'

const { theme, setTheme } = useDarkMode()

const options: { value: Theme, icon: typeof Sun, label: string }[] = [
  { value: 'light', icon: Sun, label: 'Light' },
  { value: 'system', icon: Monitor, label: 'System' },
  { value: 'dark', icon: Moon, label: 'Dark' },
]
</script>

<template>
  <div class="inline-flex items-center gap-0.5 p-0.5 rounded-lg bg-surface-raised">
    <button
      v-for="opt in options"
      :key="opt.value"
      :aria-label="opt.label"
      :title="opt.label"
      class="flex items-center justify-center w-8 h-8 rounded-md transition-colors"
      :class="theme === opt.value
        ? 'bg-surface text-heading shadow-sm'
        : 'text-text-muted hover:text-text-secondary'"
      @click="setTheme(opt.value)"
    >
      <component :is="opt.icon" class="w-4 h-4" />
    </button>
  </div>
</template>
