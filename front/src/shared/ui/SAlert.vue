<script setup lang="ts">
import { AlertCircle, AlertTriangle, CheckCircle, Info, X } from 'lucide-vue-next'
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  variant?: 'info' | 'success' | 'warning' | 'danger'
  dismissible?: boolean
}>(), {
  variant: 'info',
  dismissible: false,
})

const emit = defineEmits<{ dismiss: [] }>()

const styles = computed(() => {
  const map: Record<string, string> = {
    info: 'bg-info-bg border-info-border text-text-primary',
    success: 'bg-success-bg border-success-border text-text-primary',
    warning: 'bg-warning-bg border-warning-border text-text-primary',
    danger: 'bg-danger-bg border-danger-border text-text-primary',
  }
  return map[props.variant]
})

const iconStyles: Record<string, string> = {
  info: 'text-info',
  success: 'text-success',
  warning: 'text-warning',
  danger: 'text-danger',
}

const icons: Record<string, any> = {
  info: Info,
  success: CheckCircle,
  warning: AlertTriangle,
  danger: AlertCircle,
}
</script>

<template>
  <div class="flex items-start gap-3 px-4 py-3 border rounded-xl text-sm" :class="styles">
    <component :is="icons[variant]" class="w-4 h-4 mt-0.5 shrink-0" :class="iconStyles[variant]" />
    <div class="flex-1">
      <slot />
    </div>
    <button v-if="dismissible" type="button" aria-label="Dismiss" class="shrink-0 p-0.5 rounded text-text-muted hover:text-text-primary transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50" @click="emit('dismiss')">
      <X class="w-4 h-4" aria-hidden="true" />
    </button>
  </div>
</template>
