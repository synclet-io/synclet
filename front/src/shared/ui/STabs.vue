<script setup lang="ts">
import type { Tab } from './types'
import { computed, nextTick, ref } from 'vue'
import { RouterLink } from 'vue-router'

const props = defineProps<{
  tabs: Tab[]
  modelValue?: string
  variant?: 'underline' | 'pills'
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const tablistRef = ref<HTMLElement>()

const isNav = computed(() => props.tabs.some(tab => tab.to))

function tabValue(tab: Tab) {
  return tab.value || tab.name
}

function isSelected(tab: Tab) {
  return props.modelValue === tabValue(tab)
}

function tabIndexOf(tab: Tab, index: number) {
  const hasSelection = props.tabs.some(isSelected)
  return isSelected(tab) || (!hasSelection && index === 0) ? 0 : -1
}

async function selectAt(index: number) {
  const count = props.tabs.length
  if (count === 0)
    return
  const next = (index + count) % count
  emit('update:modelValue', tabValue(props.tabs[next]))
  await nextTick()
  tablistRef.value?.querySelectorAll<HTMLElement>('[role="tab"]')[next]?.focus()
}

function onKeydown(e: KeyboardEvent, index: number) {
  const moves: Record<string, number> = {
    ArrowRight: index + 1,
    ArrowLeft: index - 1,
    Home: 0,
    End: props.tabs.length - 1,
  }
  if (!(e.key in moves))
    return
  e.preventDefault()
  selectAt(moves[e.key])
}

const containerClass = computed(() => props.variant === 'pills'
  ? 'inline-flex gap-1 p-1 bg-surface-raised rounded-lg mb-6'
  : 'flex gap-1 border-b border-border mb-6')

const tabClass = computed(() => props.variant === 'pills'
  ? 'px-3 py-1.5 text-sm font-medium rounded-md transition-colors whitespace-nowrap focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50'
  : 'px-4 py-2.5 text-sm font-medium -mb-px border-b-2 transition-colors whitespace-nowrap focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50')

function stateClass(active: boolean) {
  if (props.variant === 'pills')
    return active ? 'bg-surface text-heading shadow-sm' : 'text-text-secondary hover:text-heading'
  return active ? 'border-primary text-primary' : 'border-transparent text-text-secondary hover:text-heading hover:border-border'
}
</script>

<template>
  <nav v-if="isNav" :class="containerClass">
    <template v-for="tab in tabs" :key="tab.name">
      <RouterLink v-if="tab.to" v-slot="{ isExactActive, navigate, href }" :to="tab.to" custom>
        <a
          :href="href"
          :aria-current="isExactActive ? 'page' : undefined"
          class="cursor-pointer"
          :class="[tabClass, stateClass(isExactActive)]"
          @click="navigate"
        >
          {{ tab.name }}
        </a>
      </RouterLink>
    </template>
  </nav>

  <div v-else ref="tablistRef" role="tablist" :class="containerClass">
    <button
      v-for="(tab, index) in tabs"
      :key="tab.name"
      type="button"
      role="tab"
      :aria-selected="isSelected(tab)"
      :tabindex="tabIndexOf(tab, index)"
      :class="[tabClass, stateClass(isSelected(tab))]"
      @click="emit('update:modelValue', tabValue(tab))"
      @keydown="onKeydown($event, index)"
    >
      {{ tab.name }}
    </button>
  </div>
</template>
