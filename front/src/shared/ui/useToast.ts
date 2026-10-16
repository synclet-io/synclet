import { ref } from 'vue'

export interface Toast {
  id: number
  message: string
  variant: 'success' | 'error' | 'info' | 'warning'
}

interface Timer {
  handle?: ReturnType<typeof setTimeout>
  remaining: number
  startedAt: number
}

export const TOAST_DURATIONS: Record<Toast['variant'], number> = {
  success: 5000,
  info: 5000,
  warning: 5000,
  error: 10000,
}

const toasts = ref<Toast[]>([])
const timers = new Map<number, Timer>()
let nextId = 0

function dismiss(id: number) {
  clearTimeout(timers.get(id)?.handle)
  timers.delete(id)
  toasts.value = toasts.value.filter(t => t.id !== id)
}

function start(id: number) {
  const timer = timers.get(id)
  if (!timer || timer.handle)
    return
  timer.startedAt = Date.now()
  timer.handle = setTimeout(dismiss, timer.remaining, id)
}

function pause(id: number) {
  const timer = timers.get(id)
  if (!timer?.handle)
    return
  clearTimeout(timer.handle)
  timer.handle = undefined
  timer.remaining = Math.max(0, timer.remaining - (Date.now() - timer.startedAt))
}

function addToast(message: string, variant: Toast['variant'], duration = TOAST_DURATIONS[variant]) {
  const id = nextId++
  toasts.value.push({ id, message, variant })
  timers.set(id, { remaining: duration, startedAt: 0 })
  start(id)
}

export function useToast() {
  return {
    toasts,
    success: (message: string) => addToast(message, 'success'),
    error: (message: string) => addToast(message, 'error'),
    info: (message: string) => addToast(message, 'info'),
    warning: (message: string) => addToast(message, 'warning'),
    dismiss,
    pause,
    resume: start,
  }
}
