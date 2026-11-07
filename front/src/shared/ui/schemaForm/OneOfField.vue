<script setup lang="ts">
import type { JsonSchema } from '@shared/lib/jsonSchemaUtils'
import { branchDiscriminators, initialBranchValue, isObjectBranch, normalizeSchema, resolveOneOf, schemaType } from '@shared/lib/jsonSchemaUtils'
import { renderDescription } from '@shared/lib/schemaDescription'
import { computed, inject } from 'vue'
import JsonSchemaForm from '../JsonSchemaForm.vue'
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
  'update:modelValue': [value: unknown]
  'blur': []
}>()

const ctx = inject(schemaFormContextKey, null)

const branches = computed(() => (props.schema.oneOf ?? []).map(normalizeSchema))

const branchIndex = computed(() => ctx?.selections[props.path] ?? resolveOneOf(props.schema, props.modelValue))

const branch = computed<JsonSchema | undefined>(() => branches.value[branchIndex.value])

const objectBranch = computed(() => !!branch.value && isObjectBranch(branch.value))

const primitiveType = computed(() => {
  if (!branch.value || objectBranch.value || branch.value.const !== undefined)
    return undefined
  return schemaType(branch.value) ?? schemaType(props.schema) ?? 'string'
})

function branchTitle(b: JsonSchema): string {
  if (b.title)
    return b.title
  if (b.const !== undefined)
    return String(b.const)
  const discriminator = branchDiscriminators(b)[0]
  return discriminator ? String(discriminator[1]) : 'Option'
}

function selectBranch(index: number) {
  const next = branches.value[index]
  if (!next)
    return
  if (ctx)
    ctx.selections[props.path] = index
  emit('update:modelValue', initialBranchValue(next))
}

function onPrimitiveInput(target: HTMLInputElement) {
  if (primitiveType.value === 'boolean')
    emit('update:modelValue', target.checked)
  else if (primitiveType.value === 'integer' || primitiveType.value === 'number')
    emit('update:modelValue', target.value === '' ? undefined : Number(target.value))
  else
    emit('update:modelValue', target.value)
}
</script>

<template>
  <div>
    <label :for="`field-${fieldKey}`" :class="labelClass">
      {{ label }}
      <span v-if="required" class="text-danger">*</span>
    </label>
    <p v-if="schema.description" :class="descriptionClass" class="mb-2" v-html="renderDescription(schema.description)" />

    <select
      :id="`field-${fieldKey}`"
      :value="branchIndex"
      :class="inputClass"
      class="mb-3"
      @change="selectBranch(Number(($event.target as HTMLSelectElement).value))"
    >
      <option v-for="(b, idx) in branches" :key="idx" :value="idx">
        {{ branchTitle(b) }}
      </option>
    </select>

    <div v-if="branch && objectBranch && branch.properties" class="pl-4 border-l-2 border-border">
      <JsonSchemaForm
        :schema="branch"
        :model-value="(modelValue as Record<string, unknown>) ?? {}"
        :root-required="branch.required"
        :path="path"
        @update:model-value="emit('update:modelValue', $event)"
      />
    </div>

    <div v-else-if="primitiveType" class="pl-4 border-l-2 border-border">
      <input
        v-if="primitiveType === 'boolean'"
        :id="`field-${fieldKey}-value`"
        type="checkbox"
        :checked="!!modelValue"
        class="h-4 w-4 rounded border-border-strong accent-primary-solid focus:ring-primary"
        :aria-label="label"
        @change="onPrimitiveInput($event.target as HTMLInputElement)"
      >
      <input
        v-else
        :id="`field-${fieldKey}-value`"
        :type="primitiveType === 'integer' || primitiveType === 'number' ? 'number' : 'text'"
        :value="modelValue ?? ''"
        :aria-label="label"
        :class="error ? inputErrorClass : inputClass"
        @input="onPrimitiveInput($event.target as HTMLInputElement)"
        @blur="emit('blur')"
      >
      <p v-if="branch?.description" :class="descriptionClass" v-html="renderDescription(branch.description)" />
    </div>
    <p v-if="error" class="text-xs text-danger mt-1">
      {{ error }}
    </p>
  </div>
</template>
