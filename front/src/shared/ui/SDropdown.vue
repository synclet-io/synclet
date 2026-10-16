<script setup lang="ts">
import type { DropdownItem } from './types'
import { Check } from 'lucide-vue-next'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId } from 'vue'

withDefaults(defineProps<{
  items: DropdownItem[]
  align?: 'left' | 'right'
}>(), {
  align: 'left',
})

const emit = defineEmits<{ select: [item: DropdownItem] }>()
const open = ref(false)
const dropdownRef = ref<HTMLElement>()
const triggerRef = ref<HTMLElement>()
const menuRef = ref<HTMLElement>()
const menuId = useId()

const triggerAttrs = computed(() => ({
  'aria-haspopup': 'menu' as const,
  'aria-expanded': open.value,
  'aria-controls': open.value ? menuId : undefined,
}))

function menuItems(): HTMLElement[] {
  return Array.from(menuRef.value?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? [])
}

function focusItem(index: number) {
  const items = menuItems()
  if (items.length === 0)
    return
  items[(index + items.length) % items.length].focus()
}

function focusTrigger() {
  triggerRef.value?.querySelector<HTMLElement>('button, a[href], [tabindex]')?.focus()
}

async function openMenu(focusIndex?: number) {
  open.value = true
  await nextTick()
  if (focusIndex !== undefined)
    focusItem(focusIndex)
}

function close(restoreFocus = false) {
  open.value = false
  if (restoreFocus)
    focusTrigger()
}

function toggle(e: MouseEvent) {
  if (open.value)
    close()
  else
    openMenu(e.detail === 0 ? 0 : undefined)
}

function selectItem(item: DropdownItem) {
  item.onClick?.()
  emit('select', item)
  close(true)
}

function onKeydown(e: KeyboardEvent) {
  const inMenu = menuRef.value?.contains(e.target as Node) ?? false
  const current = menuItems().indexOf(document.activeElement as HTMLElement)
  switch (e.key) {
    case 'Escape':
      if (open.value) {
        e.preventDefault()
        close(true)
      }
      break
    case 'ArrowDown':
      e.preventDefault()
      if (!open.value)
        openMenu(0)
      else
        focusItem(inMenu ? current + 1 : 0)
      break
    case 'ArrowUp':
      e.preventDefault()
      if (!open.value)
        openMenu(-1)
      else
        focusItem(inMenu ? current - 1 : -1)
      break
    case 'Home':
      if (inMenu) {
        e.preventDefault()
        focusItem(0)
      }
      break
    case 'End':
      if (inMenu) {
        e.preventDefault()
        focusItem(-1)
      }
      break
    case 'Tab':
      if (open.value)
        close()
      break
  }
}

function handleClickOutside(e: Event) {
  if (dropdownRef.value && !dropdownRef.value.contains(e.target as Node)) {
    open.value = false
  }
}

onMounted(() => document.addEventListener('click', handleClickOutside))
onBeforeUnmount(() => document.removeEventListener('click', handleClickOutside))
</script>

<template>
  <div ref="dropdownRef" class="relative" @keydown="onKeydown">
    <div ref="triggerRef" @click="toggle">
      <slot name="trigger" :open="open" :attrs="triggerAttrs" />
    </div>
    <Transition
      enter-active-class="transition-all duration-150 ease-out"
      leave-active-class="transition-all duration-100 ease-in"
      enter-from-class="opacity-0 scale-95 -translate-y-1"
      leave-to-class="opacity-0 scale-95 -translate-y-1"
    >
      <div
        v-if="open"
        :id="menuId"
        ref="menuRef"
        role="menu"
        class="absolute z-50 mt-1.5 min-w-[200px] max-w-[calc(100vw-2rem)] bg-surface border border-border rounded-xl shadow-overlay overflow-hidden p-1"
        :class="align === 'right' ? 'right-0' : 'left-0'"
      >
        <button
          v-for="item in items"
          :key="item.label"
          type="button"
          role="menuitem"
          tabindex="-1"
          :aria-current="item.active ? 'true' : undefined"
          class="w-full flex items-center justify-between px-3 py-2 text-sm rounded-lg hover:bg-surface-hover focus:outline-none focus-visible:bg-surface-hover focus-visible:ring-2 focus-visible:ring-primary/50 transition-colors"
          :class="item.active ? 'text-heading font-medium' : 'text-text-primary'"
          @click="selectItem(item)"
        >
          {{ item.label }}
          <Check v-if="item.active" class="w-4 h-4 text-primary" aria-hidden="true" />
        </button>
      </div>
    </Transition>
  </div>
</template>
