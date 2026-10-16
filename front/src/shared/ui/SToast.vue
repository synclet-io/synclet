<script setup lang="ts">
import { AlertCircle, AlertTriangle, CheckCircle, Info, X } from 'lucide-vue-next'
import { useToast } from './useToast'

const { toasts, dismiss, pause, resume } = useToast()

const icons: Record<string, any> = {
  success: CheckCircle,
  error: AlertCircle,
  info: Info,
  warning: AlertTriangle,
}

function onLeave(id: number, e: MouseEvent) {
  if (!(e.currentTarget as HTMLElement).contains(document.activeElement))
    resume(id)
}

function onFocusOut(id: number, e: FocusEvent) {
  const el = e.currentTarget as HTMLElement
  if (!el.contains(e.relatedTarget as Node | null) && !el.matches(':hover'))
    resume(id)
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
    <div
      role="region"
      aria-label="Notifications"
      aria-live="polite"
      class="fixed bottom-5 right-5 z-[60] flex flex-col gap-2.5 max-w-sm"
    >
      <TransitionGroup
        enter-active-class="transition-all duration-300 ease-out"
        leave-active-class="transition-all duration-200 ease-in"
        enter-from-class="opacity-0 translate-y-2 scale-95"
        leave-to-class="opacity-0 translate-x-4 scale-95"
      >
        <div
          v-for="toast in toasts"
          :key="toast.id"
          :role="toast.variant === 'error' ? 'alert' : 'status'"
          class="flex items-start gap-3 px-4 py-3 bg-surface border border-border rounded-xl shadow-overlay"
          @mouseenter="pause(toast.id)"
          @mouseleave="onLeave(toast.id, $event)"
          @focusin="pause(toast.id)"
          @focusout="onFocusOut(toast.id, $event)"
        >
          <component :is="icons[toast.variant]" class="w-5 h-5 mt-0.5 shrink-0" :class="iconStyles[toast.variant]" aria-hidden="true" />
          <p class="flex-1 text-sm text-heading">
            {{ toast.message }}
          </p>
          <button
            type="button"
            aria-label="Dismiss notification"
            class="shrink-0 p-0.5 rounded text-text-muted hover:text-heading transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
            @click="dismiss(toast.id)"
          >
            <X class="w-4 h-4" aria-hidden="true" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
