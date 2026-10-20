<script setup lang="ts">
import { X } from 'lucide-vue-next'
import { computed, nextTick, onBeforeUnmount, ref, useAttrs, useId, watch } from 'vue'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  modelValue: string[]
  suggestions?: string[]
  label?: string
  error?: string
  placeholder?: string
  readonly?: boolean
  allowCreate?: boolean
  emptyText?: string
}>(), {
  suggestions: () => [],
  allowCreate: true,
  emptyText: 'No tags',
})

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const attrs = useAttrs()
const generatedId = useId()

const inputId = computed(() => (attrs.id as string | undefined) ?? generatedId)
const labelId = computed(() => `${inputId.value}-label`)
const listboxId = computed(() => `${inputId.value}-listbox`)
const errorId = computed(() => props.error ? `${inputId.value}-error` : undefined)
const optionId = (index: number) => `${inputId.value}-option-${index}`

const rootRef = ref<HTMLElement>()
const inputRef = ref<HTMLInputElement>()
const listboxRef = ref<HTMLElement>()
const controlRef = ref<HTMLElement>()
const listboxStyle = ref<Record<string, string>>({})
const open = ref(false)
const query = ref('')
const activeIndex = ref(-1)

const normalizedQuery = computed(() => query.value.trim().toLowerCase())
const filtered = computed(() => {
  const chosen = new Set(props.modelValue)
  const q = normalizedQuery.value
  return props.suggestions.filter(s => !chosen.has(s) && (!q || s.includes(q)))
})
const activeDescendant = computed(() =>
  open.value && activeIndex.value >= 0 && activeIndex.value < filtered.value.length
    ? optionId(activeIndex.value)
    : undefined,
)

function add(raw: string) {
  const tag = raw.trim().toLowerCase()
  query.value = ''
  activeIndex.value = props.allowCreate ? -1 : 0
  if (!tag || props.modelValue.includes(tag))
    return
  emit('update:modelValue', [...props.modelValue, tag])
}

function remove(tag: string) {
  emit('update:modelValue', props.modelValue.filter(t => t !== tag))
}

function commitTyped() {
  const option = filtered.value[activeIndex.value]
  if (option) {
    add(option)
    return
  }
  const tag = normalizedQuery.value
  if (!tag)
    return
  if (props.allowCreate || props.suggestions.includes(tag))
    add(tag)
}

async function setActive(index: number) {
  const count = filtered.value.length
  activeIndex.value = count === 0 ? -1 : (index + count) % count
  await nextTick()
  if (activeDescendant.value)
    listboxRef.value?.querySelector(`#${CSS.escape(activeDescendant.value)}`)?.scrollIntoView?.({ block: 'nearest' })
}

function openList() {
  open.value = true
}

/** Places the list under the control with fixed positioning, so ancestors that clip overflow do not cut it off. */
function placeList() {
  const rect = controlRef.value?.getBoundingClientRect()
  if (!rect)
    return
  listboxStyle.value = { top: `${rect.bottom + 6}px`, left: `${rect.left}px`, width: `${rect.width}px` }
}

function stopTracking() {
  window.removeEventListener('scroll', placeList, true)
  window.removeEventListener('resize', placeList)
}

watch(open, (isOpen) => {
  stopTracking()
  if (!isOpen)
    return
  placeList()
  window.addEventListener('scroll', placeList, true)
  window.addEventListener('resize', placeList)
})

watch(() => props.modelValue.length, () => {
  if (open.value)
    void nextTick(placeList)
})

onBeforeUnmount(stopTracking)

function close() {
  open.value = false
  activeIndex.value = -1
}

function onInput(e: Event) {
  query.value = (e.target as HTMLInputElement).value
  openList()
  activeIndex.value = props.allowCreate || !normalizedQuery.value ? -1 : 0
}

function onKeydown(e: KeyboardEvent) {
  switch (e.key) {
    case 'Enter':
    case ',':
      e.preventDefault()
      commitTyped()
      break
    case 'Backspace':
      if (!query.value && props.modelValue.length > 0)
        remove(props.modelValue.at(-1)!)
      break
    case 'ArrowDown':
      e.preventDefault()
      openList()
      void setActive(activeIndex.value + 1)
      break
    case 'ArrowUp':
      e.preventDefault()
      openList()
      void setActive(activeIndex.value - 1)
      break
    case 'Escape':
      if (open.value) {
        e.preventDefault()
        close()
      }
      break
    case 'Tab':
      close()
      break
  }
}

function onFocusout(e: FocusEvent) {
  if (!rootRef.value?.contains(e.relatedTarget as Node | null))
    close()
}
</script>

<template>
  <div ref="rootRef" data-tag-input :class="$attrs.class" :style="$attrs.style" @focusout="onFocusout">
    <div v-if="label" class="mb-1.5">
      <label v-if="!readonly" :id="labelId" :for="inputId" class="text-sm font-medium text-heading">{{ label }}</label>
      <span v-else :id="labelId" class="text-sm font-medium text-heading">{{ label }}</span>
    </div>

    <div v-if="readonly" class="flex flex-wrap gap-1.5" :aria-labelledby="label ? labelId : undefined" role="list">
      <span
        v-for="tag in modelValue"
        :key="tag"
        role="listitem"
        :data-tag="tag"
        class="inline-flex items-center rounded-md bg-surface-hover px-2 py-0.5 text-xs font-medium text-heading"
      >{{ tag }}</span>
      <span v-if="modelValue.length === 0" class="text-sm text-text-muted">{{ emptyText }}</span>
    </div>

    <div v-else class="relative">
      <div
        ref="controlRef"
        class="flex flex-wrap items-center gap-1.5 min-h-10 w-full border rounded-lg bg-surface px-2 py-1.5 transition-all duration-150 focus-within:ring-2 focus-within:ring-primary/20 focus-within:border-primary"
        :class="error ? 'border-danger' : 'border-border'"
        @click="inputRef?.focus()"
      >
        <span
          v-for="tag in modelValue"
          :key="tag"
          :data-tag="tag"
          class="inline-flex items-center gap-1 rounded-md bg-surface-hover pl-2 pr-1 py-0.5 text-xs font-medium text-heading"
        >
          {{ tag }}
          <button
            type="button"
            :aria-label="`Remove tag ${tag}`"
            class="flex items-center justify-center w-4 h-4 rounded text-text-muted hover:text-heading focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
            @mousedown.prevent
            @click.stop="remove(tag)"
          >
            <X class="w-3 h-3" aria-hidden="true" />
          </button>
        </span>
        <input
          :id="inputId"
          ref="inputRef"
          type="text"
          role="combobox"
          autocomplete="off"
          aria-autocomplete="list"
          :aria-expanded="open && filtered.length > 0"
          :aria-controls="listboxId"
          :aria-activedescendant="activeDescendant"
          :aria-invalid="error ? true : undefined"
          :aria-describedby="errorId"
          :value="query"
          :placeholder="modelValue.length === 0 ? placeholder : undefined"
          class="flex-1 min-w-24 h-6 bg-transparent text-sm text-heading placeholder:text-text-muted focus:outline-none"
          @input="onInput"
          @focus="openList"
          @keydown="onKeydown"
        >
      </div>
      <ul
        v-show="open && filtered.length > 0"
        :id="listboxId"
        ref="listboxRef"
        role="listbox"
        :aria-labelledby="label ? labelId : undefined"
        :aria-label="label ? undefined : placeholder"
        :style="listboxStyle"
        class="fixed z-50 max-h-60 overflow-auto bg-surface border border-border rounded-xl shadow-overlay p-1"
      >
        <li
          v-for="(tag, index) in filtered"
          :id="optionId(index)"
          :key="tag"
          role="option"
          :aria-selected="index === activeIndex"
          class="px-3 py-2 rounded-lg cursor-pointer truncate text-sm text-text-primary"
          :class="index === activeIndex ? 'bg-surface-hover' : ''"
          @mousedown.prevent
          @mousemove="activeIndex = index"
          @click="add(tag)"
        >
          {{ tag }}
        </li>
      </ul>
    </div>

    <p v-if="error" :id="errorId" role="alert" class="mt-1.5 text-xs text-danger">
      {{ error }}
    </p>
  </div>
</template>
