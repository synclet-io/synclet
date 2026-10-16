<script setup lang="ts">
import { computed, useAttrs, useId } from 'vue'
import HelpTooltip from './HelpTooltip.vue'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  label?: string
  error?: string
  hint?: string
  tooltip?: string
  type?: string
  modelValue?: string | number
  placeholder?: string
  required?: boolean
  disabled?: boolean
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
const inputId = computed(() => (attrs.id as string | undefined) ?? generatedId)
const messageId = computed(() => (props.error || props.hint) ? `${inputId.value}-message` : undefined)
</script>

<template>
  <div :class="$attrs.class" :style="$attrs.style">
    <div v-if="label" class="flex items-center gap-1.5 mb-1.5">
      <label :for="inputId" class="text-sm font-medium text-heading">{{ label }}</label>
      <HelpTooltip v-if="tooltip" :text="tooltip" />
    </div>
    <input
      v-bind="controlAttrs"
      :id="inputId"
      :type="type || 'text'"
      :value="modelValue"
      :placeholder="placeholder"
      :required="required"
      :disabled="disabled"
      :aria-invalid="error ? true : undefined"
      :aria-describedby="messageId"
      class="block w-full h-10 px-3.5 border rounded-lg text-sm bg-surface text-heading placeholder:text-text-muted transition-all duration-150 focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary disabled:opacity-50 disabled:bg-surface-raised"
      :class="error ? 'border-danger' : 'border-border'"
      @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    >
    <p v-if="error" :id="messageId" class="mt-1.5 text-xs text-danger">
      {{ error }}
    </p>
    <p v-else-if="hint" :id="messageId" class="mt-1.5 text-xs text-text-muted">
      {{ hint }}
    </p>
  </div>
</template>
