import { ref } from 'vue'

export interface Toast {
  id: number
  message: string
  variant: 'success' | 'error' | 'info' | 'warning'
}

const toasts = ref<Toast[]>([])
let nextId = 0

function addToast(message: string, variant: Toast['variant'], duration = 5000) {
  const id = nextId++
  toasts.value.push({ id, message, variant })
  setTimeout(() => {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }, duration)
}

export function useToast() {
  return {
    toasts,
    success: (message: string) => addToast(message, 'success'),
    error: (message: string) => addToast(message, 'error'),
    info: (message: string) => addToast(message, 'info'),
    warning: (message: string) => addToast(message, 'warning'),
    dismiss: (id: number) => {
      toasts.value = toasts.value.filter(t => t.id !== id)
    },
  }
}
