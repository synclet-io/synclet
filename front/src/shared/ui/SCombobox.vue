<script setup lang="ts">
import { ChevronDown, X } from 'lucide-vue-next'
import { computed, nextTick, ref, useAttrs, useId } from 'vue'
import HelpTooltip from './HelpTooltip.vue'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  label?: string
  error?: string
  tooltip?: string
  options: Array<{ label: string, value: string | number }>
  modelValue?: string | number
  placeholder?: string
  emptyText?: string
  disabled?: boolean
  size?: 'sm' | 'md'
}>(), {
  emptyText: 'No results',
  size: 'md',
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
}>()

const attrs = useAttrs()
const generatedId = useId()

const controlAttrs = computed(() => {
  const { class: _class, style: _style, id: _id, ...rest } = attrs
  return rest
})
const inputId = computed(() => (attrs.id as string | undefined) ?? generatedId)
const labelId = computed(() => `${inputId.value}-label`)
const listboxId = computed(() => `${inputId.value}-listbox`)
const errorId = computed(() => props.error ? `${inputId.value}-error` : undefined)
const optionId = (index: number) => `${inputId.value}-option-${index}`

const rootRef = ref<HTMLElement>()
const inputRef = ref<HTMLInputElement>()
const listboxRef = ref<HTMLElement>()
const open = ref(false)
const query = ref('')
const activeIndex = ref(-1)

const hasValue = computed(() => props.modelValue !== undefined && props.modelValue !== '')
const selected = computed(() => props.options.find(o => o.value === props.modelValue))
const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  return q ? props.options.filter(o => o.label.toLowerCase().includes(q)) : props.options
})
const inputValue = computed(() => open.value ? query.value : (selected.value?.label ?? ''))
const inputPlaceholder = computed(() => open.value && selected.value ? selected.value.label : props.placeholder)
const activeDescendant = computed(() =>
  open.value && activeIndex.value >= 0 && activeIndex.value < filtered.value.length
    ? optionId(activeIndex.value)
    : undefined,
)

async function setActive(index: number) {
  const count = filtered.value.length
  activeIndex.value = count === 0 ? -1 : (index + count) % count
  await nextTick()
  if (activeDescendant.value)
    listboxRef.value?.querySelector(`#${CSS.escape(activeDescendant.value)}`)?.scrollIntoView?.({ block: 'nearest' })
}

function openList(index?: number) {
  if (props.disabled)
    return
  if (!open.value) {
    open.value = true
    query.value = ''
  }
  const selectedIndex = filtered.value.findIndex(o => o.value === props.modelValue)
  void setActive(index ?? (selectedIndex >= 0 ? selectedIndex : 0))
}

function close() {
  open.value = false
  query.value = ''
  activeIndex.value = -1
}

function pick(value: string | number) {
  emit('update:modelValue', value)
  close()
}

function clear() {
  emit('update:modelValue', '')
  close()
  inputRef.value?.focus()
}

function onInput(e: Event) {
  if (!open.value)
    openList()
  query.value = (e.target as HTMLInputElement).value
  void setActive(0)
}

function onClick() {
  if (open.value)
    close()
  else
    openList()
}

function onKeydown(e: KeyboardEvent) {
  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      if (!open.value)
        openList()
      else
        void setActive(activeIndex.value + 1)
      break
    case 'ArrowUp':
      e.preventDefault()
      if (!open.value)
        openList(-1)
      else
        void setActive(activeIndex.value - 1)
      break
    case 'Home':
      if (open.value) {
        e.preventDefault()
        void setActive(0)
      }
      break
    case 'End':
      if (open.value) {
        e.preventDefault()
        void setActive(-1)
      }
      break
    case 'Enter':
      if (open.value) {
        e.preventDefault()
        const option = filtered.value[activeIndex.value]
        if (option)
          pick(option.value)
      }
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
  <div ref="rootRef" :class="$attrs.class" :style="$attrs.style" @focusout="onFocusout">
    <div v-if="label" class="flex items-center gap-1.5 mb-1.5">
      <label :id="labelId" :for="inputId" class="text-sm font-medium text-heading">{{ label }}</label>
      <HelpTooltip v-if="tooltip" :text="tooltip" />
    </div>
    <div class="relative">
      <input
        v-bind="controlAttrs"
        :id="inputId"
        ref="inputRef"
        type="text"
        role="combobox"
        autocomplete="off"
        aria-autocomplete="list"
        :aria-expanded="open"
        :aria-controls="listboxId"
        :aria-activedescendant="activeDescendant"
        :aria-invalid="error ? true : undefined"
        :aria-describedby="errorId"
        :value="inputValue"
        :placeholder="inputPlaceholder"
        :disabled="disabled"
        class="block w-full border rounded-lg bg-surface text-heading placeholder:text-text-muted transition-all duration-150 focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary disabled:opacity-50"
        :class="[
          error ? 'border-danger' : 'border-border',
          size === 'sm' ? 'h-7 pl-2 text-xs' : 'h-10 pl-3.5 text-sm',
          size === 'sm' ? (hasValue ? 'pr-12' : 'pr-7') : (hasValue ? 'pr-16' : 'pr-10'),
        ]"
        @input="onInput"
        @click="onClick"
        @keydown="onKeydown"
      >
      <button
        v-if="hasValue && !disabled"
        type="button"
        :aria-label="`Clear ${label ?? 'selection'}`"
        class="absolute top-1/2 -translate-y-1/2 flex items-center justify-center rounded text-text-muted hover:text-heading focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
        :class="size === 'sm' ? 'right-6 w-4 h-4' : 'right-8 w-5 h-5'"
        @mousedown.prevent
        @click="clear"
      >
        <X :class="size === 'sm' ? 'w-3 h-3' : 'w-3.5 h-3.5'" aria-hidden="true" />
      </button>
      <ChevronDown
        class="pointer-events-none absolute top-1/2 -translate-y-1/2 text-text-muted"
        :class="size === 'sm' ? 'right-2 w-3.5 h-3.5' : 'right-3 w-4 h-4'"
        aria-hidden="true"
      />
      <ul
        v-show="open && filtered.length > 0"
        :id="listboxId"
        ref="listboxRef"
        role="listbox"
        :aria-labelledby="label ? labelId : undefined"
        :aria-label="label ? undefined : placeholder"
        class="absolute z-50 mt-1.5 w-full max-h-60 overflow-auto bg-surface border border-border rounded-xl shadow-overlay p-1"
      >
        <li
          v-for="(opt, index) in filtered"
          :id="optionId(index)"
          :key="opt.value"
          role="option"
          :aria-selected="opt.value === modelValue"
          class="px-3 py-2 rounded-lg cursor-pointer truncate"
          :class="[
            size === 'sm' ? 'text-xs' : 'text-sm',
            index === activeIndex ? 'bg-surface-hover' : '',
            opt.value === modelValue ? 'text-heading font-medium' : 'text-text-primary',
          ]"
          @mousedown.prevent
          @mousemove="activeIndex = index"
          @click="pick(opt.value)"
        >
          {{ opt.label }}
        </li>
      </ul>
      <p
        v-if="open && filtered.length === 0"
        role="status"
        class="absolute z-50 mt-1.5 w-full bg-surface border border-border rounded-xl shadow-overlay px-3 py-2 text-text-muted"
        :class="size === 'sm' ? 'text-xs' : 'text-sm'"
      >
        {{ emptyText }}
      </p>
    </div>
    <p v-if="error" :id="errorId" class="mt-1.5 text-xs text-danger">
      {{ error }}
    </p>
  </div>
</template>
