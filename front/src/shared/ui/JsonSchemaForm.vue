<script setup lang="ts">
import type { JsonSchema } from '@shared/lib/jsonSchemaUtils'
import {
  getFieldDefaults,
  getOrderedProperties,
  humanizeKey,

  mergeAllOf,
  resolveOneOf,
  validateField,
} from '@shared/lib/jsonSchemaUtils'
import DOMPurify from 'dompurify'
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  schema: JsonSchema
  modelValue: Record<string, unknown>
  rootRequired?: string[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, unknown>]
}>()

const touched = defineModel<Record<string, boolean>>('touched', { default: () => ({}) })

const resolvedSchema = computed(() => mergeAllOf(props.schema))

const orderedKeys = computed(() => getOrderedProperties(resolvedSchema.value))

const requiredSet = computed(() => new Set(props.rootRequired ?? resolvedSchema.value.required ?? []))

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

// oneOf handling
function getOneOfBranchIndex(key: string): number {
  const fieldSchema = resolvedSchema.value.properties?.[key]
  if (!fieldSchema?.oneOf)
    return -1
  return resolveOneOf(fieldSchema, props.modelValue[key] as Record<string, unknown> ?? {})
}

function getOneOfTitle(branch: JsonSchema): string {
  if (branch.title)
    return branch.title
  // Find the const discriminator to use as title
  if (branch.properties) {
    for (const propSchema of Object.values(branch.properties)) {
      if (propSchema.const !== undefined)
        return String(propSchema.const)
    }
  }
  return 'Option'
}

function selectOneOfBranch(key: string, branchIndex: number) {
  const fieldSchema = resolvedSchema.value.properties?.[key]
  if (!fieldSchema?.oneOf)
    return
  const branch = fieldSchema.oneOf[branchIndex]
  if (!branch?.properties)
    return

  // Auto-set const values from the branch
  const newValue: Record<string, unknown> = {}
  for (const [propKey, propSchema] of Object.entries(branch.properties)) {
    if (propSchema.const !== undefined) {
      newValue[propKey] = propSchema.const
    }
  }
  updateField(key, newValue)
}

function handleOneOfSubfieldUpdate(key: string, subValue: Record<string, unknown>) {
  updateField(key, subValue)
}

// Array of strings handling
function getArrayValue(key: string): string {
  const val = props.modelValue[key]
  if (Array.isArray(val))
    return val.join(', ')
  return ''
}

function updateArrayField(key: string, text: string) {
  const arr = text ? text.split(',').map(s => s.trim()).filter(Boolean) : []
  updateField(key, arr)
}

// Nested object handling
function updateObjectField(key: string, subValue: Record<string, unknown>) {
  updateField(key, subValue)
}

// Track secret field visibility separately from form data
const showSecret = ref<Record<string, boolean>>({})

const inputClass = 'w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-sm'
const inputErrorClass = 'w-full px-3 py-2 border border-danger-500 rounded-lg bg-white dark:bg-gray-700 text-sm'
const labelClass = 'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1'
</script>

<template>
  <div class="space-y-4">
    <template v-for="key in orderedKeys" :key="key">
      <div v-if="resolvedSchema.properties?.[key] && !isConst(resolvedSchema.properties[key])">
        <!-- oneOf field -->
        <template v-if="resolvedSchema.properties[key].oneOf">
          <div>
            <label :class="labelClass">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger-500">*</span>
            </label>
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-gray-500 mb-2 [&_a]:text-primary [&_a]:underline" v-html="DOMPurify.sanitize(resolvedSchema.properties[key].description)" />

            <select
              :value="getOneOfBranchIndex(key)"
              :class="inputClass"
              class="mb-3"
              @change="selectOneOfBranch(key, Number(($event.target as HTMLSelectElement).value))"
            >
              <option
                v-for="(branch, idx) in resolvedSchema.properties[key].oneOf"
                :key="idx"
                :value="idx"
              >
                {{ getOneOfTitle(branch) }}
              </option>
            </select>

            <!-- Render active branch fields -->
            <div
              v-if="getOneOfBranchIndex(key) >= 0 && resolvedSchema.properties[key].oneOf![getOneOfBranchIndex(key)]?.properties"
              class="pl-4 border-l-2 border-gray-200 dark:border-gray-600"
            >
              <JsonSchemaForm
                :schema="resolvedSchema.properties[key].oneOf![getOneOfBranchIndex(key)]"
                :model-value="(modelValue[key] as Record<string, unknown>) ?? {}"
                :root-required="resolvedSchema.properties[key].oneOf![getOneOfBranchIndex(key)].required"
                @update:model-value="handleOneOfSubfieldUpdate(key, $event)"
              />
            </div>
          </div>
        </template>

        <!-- Nested object field -->
        <template v-else-if="resolvedSchema.properties[key].type === 'object' && resolvedSchema.properties[key].properties">
          <fieldset class="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
            <legend class="text-sm font-medium text-gray-700 dark:text-gray-300 px-2">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger-500">*</span>
            </legend>
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-gray-500 mb-3 [&_a]:text-primary [&_a]:underline" v-html="DOMPurify.sanitize(resolvedSchema.properties[key].description)" />
            <JsonSchemaForm
              :schema="resolvedSchema.properties[key]"
              :model-value="(modelValue[key] as Record<string, unknown>) ?? {}"
              :root-required="resolvedSchema.properties[key].required"
              @update:model-value="updateObjectField(key, $event)"
            />
          </fieldset>
        </template>

        <!-- Boolean -->
        <template v-else-if="resolvedSchema.properties[key].type === 'boolean'">
          <div class="flex items-center gap-2">
            <input
              :id="`field-${key}`"
              type="checkbox"
              :checked="!!modelValue[key]"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              @change="updateField(key, ($event.target as HTMLInputElement).checked)"
            >
            <label :for="`field-${key}`" class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger-500">*</span>
            </label>
            <span v-if="resolvedSchema.properties[key].description" class="text-xs text-gray-500 [&_a]:text-primary [&_a]:underline" v-html="DOMPurify.sanitize(resolvedSchema.properties[key].description)" />
          </div>
        </template>

        <!-- Enum select -->
        <template v-else-if="resolvedSchema.properties[key].enum">
          <div>
            <label :for="`field-${key}`" :class="labelClass">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger-500">*</span>
            </label>
            <select
              :id="`field-${key}`"
              :value="modelValue[key] ?? ''"
              :class="getError(key, resolvedSchema.properties[key]) ? inputErrorClass : inputClass"
              @change="updateField(key, ($event.target as HTMLSelectElement).value)"
            >
              <option value="" disabled>
                Select...
              </option>
              <option v-for="opt in resolvedSchema.properties[key].enum" :key="String(opt)" :value="opt">
                {{ opt }}
              </option>
            </select>
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-gray-500 mt-1 [&_a]:text-primary [&_a]:underline" v-html="DOMPurify.sanitize(resolvedSchema.properties[key].description)" />
            <p v-if="getError(key, resolvedSchema.properties[key])" class="text-xs text-danger-600 mt-1">
              {{ getError(key, resolvedSchema.properties[key]) }}
            </p>
          </div>
        </template>

        <!-- Array of strings -->
        <template v-else-if="resolvedSchema.properties[key].type === 'array'">
          <div>
            <label :for="`field-${key}`" :class="labelClass">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger-500">*</span>
            </label>
            <input
              :id="`field-${key}`"
              type="text"
              :value="getArrayValue(key)"
              :placeholder="getPlaceholder(resolvedSchema.properties[key]) || 'Comma-separated values'"
              :class="getError(key, resolvedSchema.properties[key]) ? inputErrorClass : inputClass"
              @input="updateArrayField(key, ($event.target as HTMLInputElement).value)"
              @blur="markTouched(key)"
            >
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-gray-500 mt-1 [&_a]:text-primary [&_a]:underline" v-html="DOMPurify.sanitize(resolvedSchema.properties[key].description)" />
            <p v-if="getError(key, resolvedSchema.properties[key])" class="text-xs text-danger-600 mt-1">
              {{ getError(key, resolvedSchema.properties[key]) }}
            </p>
          </div>
        </template>

        <!-- Number / Integer -->
        <template v-else-if="resolvedSchema.properties[key].type === 'integer' || resolvedSchema.properties[key].type === 'number'">
          <div>
            <label :for="`field-${key}`" :class="labelClass">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger-500">*</span>
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
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-gray-500 mt-1 [&_a]:text-primary [&_a]:underline" v-html="DOMPurify.sanitize(resolvedSchema.properties[key].description)" />
            <p v-if="getError(key, resolvedSchema.properties[key])" class="text-xs text-danger-600 mt-1">
              {{ getError(key, resolvedSchema.properties[key]) }}
            </p>
          </div>
        </template>

        <!-- Multiline string -->
        <template v-else-if="resolvedSchema.properties[key].multiline">
          <div>
            <label :for="`field-${key}`" :class="labelClass">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger-500">*</span>
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
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-gray-500 mt-1 [&_a]:text-primary [&_a]:underline" v-html="DOMPurify.sanitize(resolvedSchema.properties[key].description)" />
            <p v-if="getError(key, resolvedSchema.properties[key])" class="text-xs text-danger-600 mt-1">
              {{ getError(key, resolvedSchema.properties[key]) }}
            </p>
          </div>
        </template>

        <!-- Secret string -->
        <template v-else-if="resolvedSchema.properties[key].airbyte_secret">
          <div>
            <label :for="`field-${key}`" :class="labelClass">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger-500">*</span>
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
                class="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-gray-500 hover:text-gray-700"
                @click="showSecret[key] = !showSecret[key]"
              >
                {{ showSecret[key] ? 'Hide' : 'Show' }}
              </button>
            </div>
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-gray-500 mt-1 [&_a]:text-primary [&_a]:underline" v-html="DOMPurify.sanitize(resolvedSchema.properties[key].description)" />
            <p v-if="getError(key, resolvedSchema.properties[key])" class="text-xs text-danger-600 mt-1">
              {{ getError(key, resolvedSchema.properties[key]) }}
            </p>
          </div>
        </template>

        <!-- Default: text input -->
        <template v-else>
          <div>
            <label :for="`field-${key}`" :class="labelClass">
              {{ getLabel(key, resolvedSchema.properties[key]) }}
              <span v-if="requiredSet.has(key)" class="text-danger-500">*</span>
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
            <p v-if="resolvedSchema.properties[key].description" class="text-xs text-gray-500 mt-1 [&_a]:text-primary [&_a]:underline" v-html="DOMPurify.sanitize(resolvedSchema.properties[key].description)" />
            <p v-if="getError(key, resolvedSchema.properties[key])" class="text-xs text-danger-600 mt-1">
              {{ getError(key, resolvedSchema.properties[key]) }}
            </p>
          </div>
        </template>
      </div>
    </template>
  </div>
</template>
