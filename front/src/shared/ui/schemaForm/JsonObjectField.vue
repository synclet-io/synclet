<script setup lang="ts">
import type { JsonSchema } from '@shared/lib/jsonSchemaUtils'
import { isPlainObject } from '@shared/lib/jsonSchemaUtils'
import { renderDescription } from '@shared/lib/schemaDescription'
import { inject, onBeforeUnmount, ref, watch } from 'vue'
import { descriptionClass, inputClass, inputErrorClass, labelClass, schemaFormContextKey } from './context'

const props = defineProps<{
  fieldKey: string
  path: string
  label: string
  required: boolean
  schema: JsonSchema
  modelValue: unknown
  error: string | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, unknown> | undefined]
  'blur': []
}>()

const ctx = inject(schemaFormContextKey, null)

function format(value: unknown): string {
  return value === undefined ? '' : JSON.stringify(value, null, 2)
}

const text = ref(format(props.modelValue))
const parseError = ref<string | null>(null)

function setParseError(message: string | null) {
  parseError.value = message
  if (!ctx)
    return
  if (message)
    ctx.invalidJson[props.path] = message
  else
    delete ctx.invalidJson[props.path]
}

let lastEmitted = JSON.stringify(props.modelValue)

watch(() => props.modelValue, (value) => {
  if (JSON.stringify(value) === lastEmitted)
    return
  lastEmitted = JSON.stringify(value)
  text.value = format(value)
  setParseError(null)
})

function commit(value: Record<string, unknown> | undefined) {
  lastEmitted = JSON.stringify(value)
  setParseError(null)
  emit('update:modelValue', value)
}

function onInput(next: string) {
  text.value = next
  if (!next.trim()) {
    commit(undefined)
    return
  }
  let parsed: unknown
  try {
    parsed = JSON.parse(next)
  }
  catch {
    setParseError('Invalid JSON')
    return
  }
  if (!isPlainObject(parsed)) {
    setParseError('Must be a JSON object')
    return
  }
  commit(parsed)
}

onBeforeUnmount(() => setParseError(null))
</script>

<template>
  <div>
    <label :for="`field-${fieldKey}`" :class="labelClass">
      {{ label }}
      <span v-if="required" class="text-danger">*</span>
    </label>
    <textarea
      :id="`field-${fieldKey}`"
      :value="text"
      rows="6"
      spellcheck="false"
      placeholder="{}"
      class="font-mono"
      :class="parseError || error ? inputErrorClass : inputClass"
      @input="onInput(($event.target as HTMLTextAreaElement).value)"
      @blur="emit('blur')"
    />
    <p v-if="schema.description" :class="descriptionClass" v-html="renderDescription(schema.description)" />
    <p v-if="parseError || error" class="text-xs text-danger mt-1">
      {{ parseError ?? error }}
    </p>
  </div>
</template>
