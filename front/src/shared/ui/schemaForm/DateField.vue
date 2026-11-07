<script setup lang="ts">
import type { DateLayout } from '@shared/lib/dateFormats'
import type { JsonSchema } from '@shared/lib/jsonSchemaUtils'
import { fitsDateInput } from '@shared/lib/dateFormats'
import { renderDescription } from '@shared/lib/schemaDescription'
import { computed, ref, watch } from 'vue'
import { descriptionClass, inputClass, inputErrorClass, labelClass } from './context'

const props = defineProps<{
  fieldKey: string
  label: string
  required: boolean
  schema: JsonSchema
  layout: DateLayout
  modelValue: unknown
  error: string | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string | undefined]
  'blur': []
}>()

function needsText(value: unknown): boolean {
  if (value === undefined || value === null || value === '')
    return false
  return typeof value !== 'string' || !fitsDateInput(value, props.layout)
}

/** A stored value the native input cannot show exactly is edited as text, so it is never blanked or rewritten. */
const textMode = ref(needsText(props.modelValue))
let lastEmitted: unknown = props.modelValue

watch(() => props.modelValue, (value) => {
  if (value !== lastEmitted)
    textMode.value = needsText(value)
  lastEmitted = value
})

const inputValue = computed(() => {
  if (textMode.value)
    return props.modelValue == null ? '' : String(props.modelValue)
  return typeof props.modelValue === 'string' ? props.layout.toInput(props.modelValue) : ''
})

function onInput(input: string) {
  if (textMode.value)
    lastEmitted = input === '' ? undefined : input
  else
    lastEmitted = input ? props.layout.fromInput(input) : undefined
  emit('update:modelValue', lastEmitted as string | undefined)
}
</script>

<template>
  <div>
    <label :for="`field-${fieldKey}`" :class="labelClass">
      {{ label }}
      <span v-if="required" class="text-danger">*</span>
      <span v-if="layout.kind === 'date-time' && !textMode" class="text-xs font-normal text-text-muted">(UTC)</span>
    </label>
    <input
      :id="`field-${fieldKey}`"
      :type="textMode ? 'text' : layout.kind === 'date' ? 'date' : 'datetime-local'"
      :step="textMode || layout.kind === 'date' ? undefined : 1"
      :placeholder="textMode ? (schema.pattern_descriptor ?? String(schema.examples?.[0] ?? '')) : undefined"
      :value="inputValue"
      :class="error ? inputErrorClass : inputClass"
      @input="onInput(($event.target as HTMLInputElement).value)"
      @blur="emit('blur')"
    >
    <p v-if="schema.description" :class="descriptionClass" v-html="renderDescription(schema.description)" />
    <p v-if="error" class="text-xs text-danger mt-1">
      {{ error }}
    </p>
  </div>
</template>
