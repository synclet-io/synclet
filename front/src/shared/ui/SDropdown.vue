<script setup lang="ts">
import type { DropdownItem } from './types'
import { Check } from 'lucide-vue-next'
import { onBeforeUnmount, onMounted, ref } from 'vue'

withDefaults(defineProps<{
  items: DropdownItem[]
  align?: 'left' | 'right'
}>(), {
  align: 'left',
})

const emit = defineEmits<{ select: [item: DropdownItem] }>()
const open = ref(false)
const dropdownRef = ref<HTMLElement>()

function toggle() {
  open.value = !open.value
}

function selectItem(item: DropdownItem) {
  item.onClick?.()
  emit('select', item)
  open.value = false
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
  <div ref="dropdownRef" class="relative">
    <div @click="toggle">
      <slot name="trigger" :open="open" />
    </div>
    <Transition
      enter-active-class="transition-all duration-150 ease-out"
      leave-active-class="transition-all duration-100 ease-in"
      enter-from-class="opacity-0 scale-95 -translate-y-1"
      leave-to-class="opacity-0 scale-95 -translate-y-1"
    >
      <div
        v-if="open"
        class="absolute z-50 mt-1.5 min-w-[200px] max-w-[calc(100vw-2rem)] bg-surface border border-border rounded-xl shadow-overlay overflow-hidden p-1"
        :class="align === 'right' ? 'right-0' : 'left-0'"
      >
        <button
          v-for="item in items"
          :key="item.label"
          class="w-full flex items-center justify-between px-3 py-2 text-sm rounded-lg hover:bg-surface-hover transition-colors"
          :class="item.active ? 'text-heading font-medium' : 'text-text-primary'"
          @click="selectItem(item)"
        >
          {{ item.label }}
          <Check v-if="item.active" class="w-4 h-4 text-primary" />
        </button>
      </div>
    </Transition>
  </div>
</template>
