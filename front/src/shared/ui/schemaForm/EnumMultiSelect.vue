<script setup lang="ts">
import type { JsonSchema } from '@shared/lib/jsonSchemaUtils'
import { renderDescription } from '@shared/lib/schemaDescription'
import { computed } from 'vue'
import { descriptionClass } from './context'

const props = defineProps<{
  fieldKey: string
  label: string
  required: boolean
  schema: JsonSchema
  modelValue: unknown
  error: string | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: unknown[]]
  'blur': []
}>()

const options = computed(() => (props.schema.items?.enum ?? []) as unknown[])
const selected = computed(() => Array.isArray(props.modelValue) ? props.modelValue : [])

function toggle(option: unknown, checked: boolean) {
  const next = checked
    ? options.value.filter(o => o === option || selected.value.includes(o))
    : selected.value.filter(o => o !== option)
  emit('update:modelValue', next)
  emit('blur')
}
</script>

<template>
  <fieldset>
    <legend class="block text-sm font-medium text-heading mb-1">
      {{ label }}
      <span v-if="required" class="text-danger">*</span>
    </legend>
    <div class="flex flex-wrap gap-x-4 gap-y-2" :class="error ? 'rounded-lg ring-1 ring-danger p-2' : ''">
      <label v-for="(option, idx) in options" :key="String(option)" class="flex items-center gap-2 text-sm text-heading">
        <input
          :id="`field-${fieldKey}-${idx}`"
          type="checkbox"
          :value="String(option)"
          :checked="selected.includes(option)"
          class="h-4 w-4 rounded border-border-strong accent-primary-solid focus:ring-primary"
          @change="toggle(option, ($event.target as HTMLInputElement).checked)"
        >
        {{ option }}
      </label>
    </div>
    <p v-if="schema.description" :class="descriptionClass" v-html="renderDescription(schema.description)" />
    <p v-if="error" class="text-xs text-danger mt-1">
      {{ error }}
    </p>
  </fieldset>
</template>
