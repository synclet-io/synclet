<script lang="ts">
import { X } from 'lucide-vue-next'
import { nextTick, onUnmounted, ref, useId, watch } from 'vue'

const openStack: symbol[] = []

const FOCUSABLE = 'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
</script>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  open: boolean
  title?: string
  size?: 'sm' | 'md' | 'lg'
}>(), {
  size: 'md',
})

const emit = defineEmits<{ close: [] }>()

const sizeClasses: Record<string, string> = {
  sm: 'max-w-sm',
  md: 'max-w-lg',
  lg: 'max-w-2xl',
}

const titleId = useId()
const panelRef = ref<HTMLElement>()
const bodyRef = ref<HTMLElement>()
const token = Symbol('modal')
let previouslyFocused: HTMLElement | null = null

function focusables(root = panelRef.value): HTMLElement[] {
  return Array.from(root?.querySelectorAll<HTMLElement>(FOCUSABLE) ?? [])
}

function onKeydown(e: KeyboardEvent) {
  if (openStack.at(-1) !== token)
    return
  if (e.key === 'Escape') {
    e.preventDefault()
    emit('close')
    return
  }
  if (e.key !== 'Tab' || !panelRef.value)
    return
  const items = focusables()
  if (items.length === 0) {
    e.preventDefault()
    panelRef.value.focus()
    return
  }
  const first = items[0]
  const last = items.at(-1)!
  const active = document.activeElement
  const inside = panelRef.value.contains(active)
  if (e.shiftKey && (active === first || !inside)) {
    e.preventDefault()
    last.focus()
  }
  else if (!e.shiftKey && (active === last || !inside)) {
    e.preventDefault()
    first.focus()
  }
}

function activate() {
  previouslyFocused = document.activeElement instanceof HTMLElement ? document.activeElement : null
  openStack.push(token)
  document.addEventListener('keydown', onKeydown)
  document.body.style.overflow = 'hidden'
  nextTick(() => {
    const target = panelRef.value?.querySelector<HTMLElement>('[autofocus]')
      ?? focusables(bodyRef.value)[0]
      ?? focusables()[0]
      ?? panelRef.value
    target?.focus()
  })
}

function deactivate() {
  const index = openStack.indexOf(token)
  if (index === -1)
    return
  openStack.splice(index, 1)
  document.removeEventListener('keydown', onKeydown)
  if (openStack.length === 0)
    document.body.style.overflow = ''
  previouslyFocused?.focus()
  previouslyFocused = null
}

watch(() => props.open, val => val ? activate() : deactivate(), { immediate: true })

onUnmounted(deactivate)
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-200"
      leave-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div class="fixed inset-0 bg-slate-900/60 backdrop-blur-sm" aria-hidden="true" @click="emit('close')" />
        <Transition
          appear
          enter-active-class="transition-all duration-200"
          enter-from-class="opacity-0 scale-95 translate-y-2"
        >
          <div
            ref="panelRef"
            role="dialog"
            aria-modal="true"
            :aria-labelledby="title ? titleId : undefined"
            tabindex="-1"
            class="relative bg-surface border border-border rounded-2xl shadow-overlay w-full focus:outline-none"
            :class="sizeClasses[size]"
          >
            <div v-if="title" class="flex items-center justify-between px-6 py-4 border-b border-border">
              <h2 :id="titleId" class="text-base font-semibold text-heading">
                {{ title }}
              </h2>
              <button
                type="button"
                aria-label="Close"
                class="p-1.5 -mr-1 rounded-lg text-text-muted hover:text-heading hover:bg-surface-hover transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
                @click="emit('close')"
              >
                <X class="w-4 h-4" aria-hidden="true" />
              </button>
            </div>
            <div ref="bodyRef">
              <div class="p-6">
                <slot />
              </div>
              <div v-if="$slots.footer" class="px-6 py-4 border-t border-border flex justify-end gap-3">
                <slot name="footer" />
              </div>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
