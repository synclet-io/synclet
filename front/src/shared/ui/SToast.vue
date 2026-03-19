<script setup lang="ts">
import { AlertCircle, AlertTriangle, CheckCircle, Info, X } from 'lucide-vue-next'
import { useToast } from './useToast'

const { toasts, dismiss } = useToast()

const icons: Record<string, any> = {
  success: CheckCircle,
  error: AlertCircle,
  info: Info,
  warning: AlertTriangle,
}

const iconStyles: Record<string, string> = {
  success: 'text-success',
  error: 'text-danger',
  info: 'text-info',
  warning: 'text-warning',
}
</script>

<template>
  <Teleport to="body">
    <div class="fixed bottom-5 right-5 z-[60] flex flex-col gap-2.5 max-w-sm">
      <TransitionGroup
        enter-active-class="transition-all duration-300 ease-out"
        leave-active-class="transition-all duration-200 ease-in"
        enter-from-class="opacity-0 translate-y-2 scale-95"
        leave-to-class="opacity-0 translate-x-4 scale-95"
      >
        <div
          v-for="toast in toasts"
          :key="toast.id"
          class="flex items-start gap-3 px-4 py-3 bg-surface border border-border rounded-xl shadow-overlay"
        >
          <component :is="icons[toast.variant]" class="w-5 h-5 mt-0.5 shrink-0" :class="iconStyles[toast.variant]" />
          <p class="flex-1 text-sm text-heading">
            {{ toast.message }}
          </p>
          <button class="shrink-0 p-0.5 text-text-muted hover:text-heading transition-colors" @click="dismiss(toast.id)">
            <X class="w-4 h-4" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
