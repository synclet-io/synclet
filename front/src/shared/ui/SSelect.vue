<script setup lang="ts">
import { ChevronDown } from 'lucide-vue-next'
import { computed, useAttrs, useId } from 'vue'
import HelpTooltip from './HelpTooltip.vue'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  label?: string
  error?: string
  tooltip?: string
  options: Array<{ label: string, value: string | number }>
  modelValue?: string | number
  placeholder?: string
  required?: boolean
  disabled?: boolean
  size?: 'sm' | 'md'
}>()

defineEmits<{
  'update:modelValue': [value: string | number]
}>()

const attrs = useAttrs()
const generatedId = useId()

const controlAttrs = computed(() => {
  const { class: _class, style: _style, ...rest } = attrs
  return rest
})
const selectId = computed(() => (attrs.id as string | undefined) ?? generatedId)
const errorId = computed(() => props.error ? `${selectId.value}-error` : undefined)
</script>

<template>
  <div :class="$attrs.class" :style="$attrs.style">
    <div v-if="label" class="flex items-center gap-1.5 mb-1.5">
      <label :for="selectId" class="text-sm font-medium text-heading">{{ label }}</label>
      <HelpTooltip v-if="tooltip" :text="tooltip" />
    </div>
    <div class="relative">
      <select
        v-bind="controlAttrs"
        :id="selectId"
        :value="modelValue"
        :required="required"
        :disabled="disabled"
        :aria-invalid="error ? true : undefined"
        :aria-describedby="errorId"
        class="block w-full appearance-none border rounded-lg bg-surface text-heading transition-all duration-150 focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary disabled:opacity-50"
        :class="[
          error ? 'border-danger' : 'border-border',
          size === 'sm' ? 'h-7 pl-2 pr-7 text-xs' : 'h-10 pl-3.5 pr-10 text-sm',
        ]"
        @change="$emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
      >
        <option v-if="placeholder" value="" disabled>
          {{ placeholder }}
        </option>
        <option v-for="opt in options" :key="opt.value" :value="opt.value">
          {{ opt.label }}
        </option>
      </select>
      <ChevronDown
        class="pointer-events-none absolute top-1/2 -translate-y-1/2 text-text-muted"
        :class="size === 'sm' ? 'right-2 w-3.5 h-3.5' : 'right-3 w-4 h-4'"
        aria-hidden="true"
      />
    </div>
    <p v-if="error" :id="errorId" class="mt-1.5 text-xs text-danger">
      {{ error }}
    </p>
  </div>
</template>
