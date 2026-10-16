<script setup lang="ts">
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
    <select
      v-bind="controlAttrs"
      :id="selectId"
      :value="modelValue"
      :required="required"
      :disabled="disabled"
      :aria-invalid="error ? true : undefined"
      :aria-describedby="errorId"
      class="block w-full border rounded-lg bg-surface text-heading transition-all duration-150 focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary disabled:opacity-50"
      :class="[
        error ? 'border-danger' : 'border-border',
        size === 'sm' ? 'h-7 px-2 text-xs' : 'h-10 px-3.5 text-sm',
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
    <p v-if="error" :id="errorId" class="mt-1.5 text-xs text-danger">
      {{ error }}
    </p>
  </div>
</template>
