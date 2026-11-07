<script setup lang="ts">
import type { JsonSchema, ValidationError } from '@shared/lib/jsonSchemaUtils'
import type { SchemaFormContext } from './schemaForm/context'
import { dateLayoutFor } from '@shared/lib/dateFormats'
import {
  coerceArrayItems,
  getFieldDefaults,
  getOrderedProperties,
  humanizeKey,
  normalizeProperties,
  validateField,
  validateObjectValue,
} from '@shared/lib/jsonSchemaUtils'
import { renderDescription } from '@shared/lib/schemaDescription'
import { computed, inject, provide, reactive, ref, watch } from 'vue'
import { joinFieldPath, schemaFormContextKey } from './schemaForm/context'
import DateField from './schemaForm/DateField.vue'
import EnumMultiSelect from './schemaForm/EnumMultiSelect.vue'
import JsonObjectField from './schemaForm/JsonObjectField.vue'
import OneOfField from './schemaForm/OneOfField.vue'

const props = defineProps<{
  schema: JsonSchema
  modelValue: Record<string, unknown>
  rootRequired?: string[]
  /** Dotted path of this form's value inside the root form's value; empty for the root form. */
  path?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, unknown>]
}>()

const touched = defineModel<Record<string, boolean>>('touched', { default: () => ({}) })

const resolvedSchema = computed(() => normalizeProperties(props.schema))

const orderedKeys = computed(() => getOrderedProperties(resolvedSchema.value)
  .filter(key => !resolvedSchema.value.properties?.[key]?.airbyte_hidden))

const requiredSet = computed(() => new Set(props.rootRequired ?? resolvedSchema.value.required ?? []))

const parentContext = inject(schemaFormContextKey, null)
const ctx: SchemaFormContext = parentContext ?? {
  showAll: ref(false),
  selections: reactive({}),
  invalidJson: reactive({}),
  errors: computed(() => {
    const errors = validateObjectValue(
      props.modelValue ?? {},
      { ...resolvedSchema.value, required: [...requiredSet.value] },
      props.path ?? '',
      { selectedBranch: path => ctx.selections[path] },
    )
    for (const [field, message] of Object.entries(ctx.invalidJson)) {
      if (!errors.some(e => e.field === field))
        errors.push({ field, message })
    }
    return errors
  }),
}
if (!parentContext)
  provide(schemaFormContextKey, ctx)

/** Validates the whole value, shows the error of every invalid field and returns the errors. */
function validate(): ValidationError[] {
  ctx.showAll.value = true
  return ctx.errors.value
}

defineExpose({ validate })

function fieldPath(key: string): string {
  return joinFieldPath(props.path, key)
}

// Apply defaults on mount
watch(
  () => props.schema,
  (schema) => {
    const defaults = getFieldDefaults(schema)
    if (Object.keys(defaults).length > 0) {
      const merged = { ...defaults }
      for (const [k, v] of Object.entries(props.modelValue)) {
        if (v !== undefined)
          merged[k] = v
      }
      emit('update:modelValue', merged)
    }
  },
  { immediate: true },
)

function updateField(key: string, value: unknown) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

function markTouched(key: string) {
  touched.value = { ...touched.value, [key]: true }
}

function getError(key: string, fieldSchema: JsonSchema): string | null {
  if (ctx.showAll.value)
    return ctx.errors.value.find(e => e.field === fieldPath(key))?.message ?? null
  if (!touched.value[key])
    return null
  const err = validateField(props.modelValue[key], fieldSchema, requiredSet.value.has(key))
  return err?.message ?? null
}

function getLabel(key: string, fieldSchema: JsonSchema): string {
  return fieldSchema.title || humanizeKey(key)
}

function getPlaceholder(fieldSchema: JsonSchema): string {
  if (fieldSchema.examples && fieldSchema.examples.length > 0) {
    return String(fieldSchema.examples[0])
  }
  return ''
}

function isConst(fieldSchema: JsonSchema): boolean {
  return fieldSchema.const !== undefined
}

function enumValue(fieldSchema: JsonSchema, selected: string): unknown {
  return (fieldSchema.enum ?? []).find(option => String(option) === selected) ?? selected
}

// Array of strings handling
const arrayDrafts = ref<Record<string, string>>({})

function parseArrayText(text: string): string[] {
  return text.split(',').map(s => s.trim()).filter(Boolean)
}

function joinArray(val: unknown): string {
  return Array.isArray(val) ? val.join(', ') : ''
}

watch(
  () => props.modelValue,
  (value) => {
    for (const [key, draft] of Object.entries(arrayDrafts.value)) {
      if (joinArray(parseArrayText(draft)) !== joinArray(value[key]))
        delete arrayDrafts.value[key]
    }
  },
)

function getArrayValue(key: string): string {
  return arrayDrafts.value[key] ?? joinArray(props.modelValue[key])
}

function updateArrayField(key: string, text: string) {
  arrayDrafts.value[key] = text
  updateField(key, coerceArrayItems(parseArrayText(text), resolvedSchema.value.properties?.[key]?.items))
}

function commitArrayField(key: string) {
  delete arrayDrafts.value[key]
  markTouched(key)
}

// Array of objects handling
function isObjectArray(fieldSchema: JsonSchema): boolean {
  return fieldSchema.type === 'array' && fieldSchema.items?.type === 'object' && !!fieldSchema.items.properties
}

function getArrayItems(key: string): Record<string, unknown>[] {
  const value = props.modelValue[key]
  return Array.isArray(value) ? value as Record<string, unknown>[] : []
}

// Stable row keys, so a removed row takes its own input state with it.
const rowIds: Record<string, number[]> = {}
let nextRowId = 0

function getRowIds(key: string): number[] {
  const ids = rowIds[key] ?? []
  const count = getArrayItems(key).length
  while (ids.length < count)
    ids.push(nextRowId++)
  ids.length = count
  rowIds[key] = ids
  return ids
}

function addArrayItem(key: string) {
  updateField(key, [...getArrayItems(key), {}])
}

function removeArrayItem(key: string, index: number) {
  rowIds[key]?.splice(index, 1)
  updateField(key, getArrayItems(key).filter((_, i) => i !== index))
}

function updateArrayItem(key: string, index: number, value: Record<string, unknown>) {
  const items = [...getArrayItems(key)]
  items[index] = value
  updateField(key, items)
}

// Nested object handling
function updateObjectField(key: string, subValue: Record<string, unknown>) {
  updateField(key, subValue)
}

// Track secret field visibility separately from form data
const showSecret = ref<Record<string, boolean>>({})

const inputBase = 'w-full px-3 py-2 border rounded-lg bg-surface text-heading text-sm placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary'
const inputClass = `${inputBase} border-border-strong`
const inputErrorClass = `${inputBase} border-danger`
const labelClass = 'block text-sm font-medium text-heading mb-1'
</script>

<template>
  <div class="space-y-4">
    <template v-for="key in orderedKeys" :key="key">
      <div v-if="resolvedSchema.properties?.[key] && !isConst(resolvedSchema.properties[key])">
        <!-- oneOf field -->
        <OneOfField
          v-if="resolvedSchema.properties[key].oneOf"
          :field-key="key"
          :path="fieldPath(key)"
          :label="getLabel(key, resolvedSchema.properties[key])"
          :required="requiredSet.has(key)"
          :schema="resolvedSchema.properties[key]"
          :model-value="modelValue[key]"
          :error="getError(key, resolvedSchema.properties[key])"
          @update:model-value="updateField(key, $event)"
          @blur="markTouched(key)"
        />

        <!-- Nested object field -->
        <template v-else-if="resolvedSchema.properties[key].type === 'object' && resolvedSchema.properties[key].properties">
          <fieldset class="border border-border rounded-lg p-4">
            <legend class="text-sm font-medium text-heading px-2">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger">*</span>
            </legend>
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-text-muted mb-3 [&_a]:text-primary [&_a]:underline" v-html="renderDescription(resolvedSchema.properties[key].description)" />
            <JsonSchemaForm
              :schema="resolvedSchema.properties[key]"
              :model-value="(modelValue[key] as Record<string, unknown>) ?? {}"
              :root-required="resolvedSchema.properties[key].required"
              :path="fieldPath(key)"
              @update:model-value="updateObjectField(key, $event)"
            />
          </fieldset>
        </template>

        <!-- Object without properties: JSON editor -->
        <JsonObjectField
          v-else-if="resolvedSchema.properties[key].type === 'object'"
          :field-key="key"
          :path="fieldPath(key)"
          :label="getLabel(key, resolvedSchema.properties[key])"
          :required="requiredSet.has(key)"
          :schema="resolvedSchema.properties[key]"
          :model-value="modelValue[key]"
          :error="getError(key, resolvedSchema.properties[key])"
          @update:model-value="updateField(key, $event)"
          @blur="markTouched(key)"
        />

        <!-- Boolean -->
        <template v-else-if="resolvedSchema.properties[key].type === 'boolean'">
          <div class="flex items-center gap-2">
            <input
              :id="`field-${key}`"
              type="checkbox"
              :checked="!!modelValue[key]"
              class="h-4 w-4 rounded border-border-strong accent-primary-solid focus:ring-primary"
              @change="updateField(key, ($event.target as HTMLInputElement).checked)"
            >
            <label :for="`field-${key}`" class="text-sm font-medium text-heading">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger">*</span>
            </label>
            <span v-if="resolvedSchema.properties[key].description" class="text-xs text-text-muted [&_a]:text-primary [&_a]:underline" v-html="renderDescription(resolvedSchema.properties[key].description)" />
          </div>
        </template>

        <!-- Enum select -->
        <template v-else-if="resolvedSchema.properties[key].enum">
          <div>
            <label :for="`field-${key}`" :class="labelClass">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger">*</span>
            </label>
            <select
              :id="`field-${key}`"
              :value="modelValue[key] ?? ''"
              :class="getError(key, resolvedSchema.properties[key]) ? inputErrorClass : inputClass"
              @change="updateField(key, enumValue(resolvedSchema.properties[key], ($event.target as HTMLSelectElement).value))"
            >
              <option value="" disabled>
                Select...
              </option>
              <option v-for="opt in resolvedSchema.properties[key].enum" :key="String(opt)" :value="opt">
                {{ opt }}
              </option>
            </select>
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-text-muted mt-1 [&_a]:text-primary [&_a]:underline" v-html="renderDescription(resolvedSchema.properties[key].description)" />
            <p v-if="getError(key, resolvedSchema.properties[key])" class="text-xs text-danger mt-1">
              {{ getError(key, resolvedSchema.properties[key]) }}
            </p>
          </div>
        </template>

        <!-- Array of objects -->
        <template v-else-if="isObjectArray(resolvedSchema.properties[key])">
          <fieldset class="border border-border rounded-lg p-4">
            <legend class="text-sm font-medium text-heading px-2">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger">*</span>
            </legend>
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-text-muted mb-3 [&_a]:text-primary [&_a]:underline" v-html="renderDescription(resolvedSchema.properties[key].description)" />
            <div class="space-y-3">
              <div
                v-for="(rowId, index) in getRowIds(key)"
                :key="rowId"
                class="flex items-start gap-3 border border-border rounded-lg p-3"
                data-testid="array-item"
              >
                <div class="flex-1">
                  <JsonSchemaForm
                    :schema="resolvedSchema.properties[key].items!"
                    :model-value="getArrayItems(key)[index] ?? {}"
                    :root-required="resolvedSchema.properties[key].items!.required"
                    :path="fieldPath(`${key}.${index}`)"
                    @update:model-value="updateArrayItem(key, index, $event)"
                  />
                </div>
                <button
                  type="button"
                  class="text-xs text-text-muted hover:text-danger mt-1"
                  :aria-label="`Remove ${getLabel(key, resolvedSchema.properties[key])} item ${index + 1}`"
                  data-testid="array-item-remove"
                  @click="removeArrayItem(key, index)"
                >
                  Remove
                </button>
              </div>
            </div>
            <button
              type="button"
              class="mt-3 text-sm font-medium text-primary hover:underline"
              data-testid="array-item-add"
              @click="addArrayItem(key)"
            >
              Add item
            </button>
          </fieldset>
        </template>

        <!-- Array of enum values -->
        <EnumMultiSelect
          v-else-if="resolvedSchema.properties[key].type === 'array' && resolvedSchema.properties[key].items?.enum"
          :field-key="key"
          :label="getLabel(key, resolvedSchema.properties[key])"
          :required="requiredSet.has(key)"
          :schema="resolvedSchema.properties[key]"
          :model-value="modelValue[key]"
          :error="getError(key, resolvedSchema.properties[key])"
          @update:model-value="updateField(key, $event)"
          @blur="markTouched(key)"
        />

        <!-- Array of strings -->
        <template v-else-if="resolvedSchema.properties[key].type === 'array'">
          <div>
            <label :for="`field-${key}`" :class="labelClass">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger">*</span>
            </label>
            <input
              :id="`field-${key}`"
              type="text"
              :value="getArrayValue(key)"
              :placeholder="getPlaceholder(resolvedSchema.properties[key]) || 'Comma-separated values'"
              :class="getError(key, resolvedSchema.properties[key]) ? inputErrorClass : inputClass"
              @input="updateArrayField(key, ($event.target as HTMLInputElement).value)"
              @blur="commitArrayField(key)"
            >
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-text-muted mt-1 [&_a]:text-primary [&_a]:underline" v-html="renderDescription(resolvedSchema.properties[key].description)" />
            <p v-if="getError(key, resolvedSchema.properties[key])" class="text-xs text-danger mt-1">
              {{ getError(key, resolvedSchema.properties[key]) }}
            </p>
          </div>
        </template>

        <!-- Number / Integer -->
        <template v-else-if="resolvedSchema.properties[key].type === 'integer' || resolvedSchema.properties[key].type === 'number'">
          <div>
            <label :for="`field-${key}`" :class="labelClass">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger">*</span>
            </label>
            <input
              :id="`field-${key}`"
              type="number"
              :value="modelValue[key] ?? ''"
              :placeholder="getPlaceholder(resolvedSchema.properties[key])"
              :class="getError(key, resolvedSchema.properties[key]) ? inputErrorClass : inputClass"
              @input="updateField(key, ($event.target as HTMLInputElement).value === '' ? undefined : Number(($event.target as HTMLInputElement).value))"
              @blur="markTouched(key)"
            >
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-text-muted mt-1 [&_a]:text-primary [&_a]:underline" v-html="renderDescription(resolvedSchema.properties[key].description)" />
            <p v-if="getError(key, resolvedSchema.properties[key])" class="text-xs text-danger mt-1">
              {{ getError(key, resolvedSchema.properties[key]) }}
            </p>
          </div>
        </template>

        <!-- Date / date-time -->
        <DateField
          v-else-if="dateLayoutFor(resolvedSchema.properties[key].format, resolvedSchema.properties[key].pattern)"
          :field-key="key"
          :label="getLabel(key, resolvedSchema.properties[key])"
          :required="requiredSet.has(key)"
          :schema="resolvedSchema.properties[key]"
          :layout="dateLayoutFor(resolvedSchema.properties[key].format, resolvedSchema.properties[key].pattern)!"
          :model-value="modelValue[key]"
          :error="getError(key, resolvedSchema.properties[key])"
          @update:model-value="updateField(key, $event)"
          @blur="markTouched(key)"
        />

        <!-- Multiline string -->
        <template v-else-if="resolvedSchema.properties[key].multiline">
          <div>
            <label :for="`field-${key}`" :class="labelClass">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger">*</span>
            </label>
            <textarea
              :id="`field-${key}`"
              :value="(modelValue[key] as string) ?? ''"
              rows="4"
              :placeholder="getPlaceholder(resolvedSchema.properties[key])"
              :class="getError(key, resolvedSchema.properties[key]) ? inputErrorClass : inputClass"
              @input="updateField(key, ($event.target as HTMLTextAreaElement).value)"
              @blur="markTouched(key)"
            />
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-text-muted mt-1 [&_a]:text-primary [&_a]:underline" v-html="renderDescription(resolvedSchema.properties[key].description)" />
            <p v-if="getError(key, resolvedSchema.properties[key])" class="text-xs text-danger mt-1">
              {{ getError(key, resolvedSchema.properties[key]) }}
            </p>
          </div>
        </template>

        <!-- Secret string -->
        <template v-else-if="resolvedSchema.properties[key].airbyte_secret">
          <div>
            <label :for="`field-${key}`" :class="labelClass">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger">*</span>
            </label>
            <div class="relative">
              <input
                :id="`field-${key}`"
                :type="showSecret[key] ? 'text' : 'password'"
                :value="(modelValue[key] as string) ?? ''"
                :placeholder="getPlaceholder(resolvedSchema.properties[key])"
                :class="getError(key, resolvedSchema.properties[key]) ? inputErrorClass : inputClass"
                @input="updateField(key, ($event.target as HTMLInputElement).value)"
                @blur="markTouched(key)"
              >
              <button
                type="button"
                class="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-text-muted hover:text-text-primary"
                @click="showSecret[key] = !showSecret[key]"
              >
                {{ showSecret[key] ? 'Hide' : 'Show' }}
              </button>
            </div>
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-text-muted mt-1 [&_a]:text-primary [&_a]:underline" v-html="renderDescription(resolvedSchema.properties[key].description)" />
            <p v-if="getError(key, resolvedSchema.properties[key])" class="text-xs text-danger mt-1">
              {{ getError(key, resolvedSchema.properties[key]) }}
            </p>
          </div>
        </template>

        <!-- Default: text input -->
        <template v-else>
          <div>
            <label :for="`field-${key}`" :class="labelClass">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger">*</span>
            </label>
            <input
              :id="`field-${key}`"
              type="text"
              :value="(modelValue[key] as string) ?? ''"
              :placeholder="getPlaceholder(resolvedSchema.properties[key])"
              :class="getError(key, resolvedSchema.properties[key]) ? inputErrorClass : inputClass"
              @input="updateField(key, ($event.target as HTMLInputElement).value)"
              @blur="markTouched(key)"
            >
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-text-muted mt-1 [&_a]:text-primary [&_a]:underline" v-html="renderDescription(resolvedSchema.properties[key].description)" />
            <p v-if="getError(key, resolvedSchema.properties[key])" class="text-xs text-danger mt-1">
              {{ getError(key, resolvedSchema.properties[key]) }}
            </p>
          </div>
        </template>
      </div>
    </template>
  </div>
</template>
