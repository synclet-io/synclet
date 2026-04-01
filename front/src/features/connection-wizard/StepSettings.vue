<script setup lang="ts">
import type { NamespaceDefinition, SchemaChangePolicy } from '@entities/connection'
import { SInput, SSelect } from '@shared/ui'
import SCronEditor from '@shared/ui/SCronEditor.vue'

defineProps<{
  name: string
  schedule: string
  schemaChangePolicy: SchemaChangePolicy
  maxAttempts: number
  namespaceDefinition: NamespaceDefinition
  customNamespaceFormat: string
  streamPrefix: string
}>()

const emit = defineEmits<{
  'update:name': [value: string]
  'update:schedule': [value: string]
  'update:schemaChangePolicy': [value: SchemaChangePolicy]
  'update:maxAttempts': [value: number]
  'update:namespaceDefinition': [value: NamespaceDefinition]
  'update:customNamespaceFormat': [value: string]
  'update:streamPrefix': [value: string]
}>()

const policyOptions = [
  { label: 'Propagate changes', value: 'propagate' },
  { label: 'Ignore changes', value: 'ignore' },
  { label: 'Pause on changes', value: 'pause' },
]

const namespaceOptions = [
  { label: 'Source namespace', value: 'source' },
  { label: 'Destination default', value: 'destination' },
  { label: 'Custom format', value: 'custom' },
]

function onName(v: string | number) {
  emit('update:name', String(v))
}
function onSchedule(v: string | number) {
  emit('update:schedule', String(v))
}
function onPolicy(v: string | number) {
  emit('update:schemaChangePolicy', v as SchemaChangePolicy)
}
function onMaxAttempts(v: string | number) {
  emit('update:maxAttempts', Number(v))
}
function onNamespace(v: string | number) {
  emit('update:namespaceDefinition', v as NamespaceDefinition)
}
function onCustomFormat(v: string | number) {
  emit('update:customNamespaceFormat', String(v))
}
function onStreamPrefix(v: string | number) {
  emit('update:streamPrefix', String(v))
}
</script>

<template>
  <div class="max-w-lg mx-auto space-y-5">
    <SInput
      :model-value="name"
      label="Connection Name"
      placeholder="My Connection"
      required
      @update:model-value="onName"
    />

    <SCronEditor
      :model-value="schedule"
      label="Schedule (optional)"
      @update:model-value="onSchedule"
    />

    <SSelect
      :model-value="schemaChangePolicy"
      label="Schema Change Policy"
      :options="policyOptions"
      @update:model-value="onPolicy"
    />

    <SInput
      :model-value="maxAttempts"
      label="Max Retry Attempts"
      type="number"
      @update:model-value="onMaxAttempts"
    />

    <div class="border-t border-border pt-5 mt-5 space-y-5">
      <h3 class="text-sm font-semibold text-heading">
        Namespace & Prefix
      </h3>

      <SSelect
        :model-value="namespaceDefinition"
        label="Namespace Mode"
        :options="namespaceOptions"
        @update:model-value="onNamespace"
      />

      <SInput
        v-if="namespaceDefinition === 'custom'"
        :model-value="customNamespaceFormat"
        label="Custom Format"
        placeholder="e.g. staging_${SOURCE_NAMESPACE}"
        hint="Use ${SOURCE_NAMESPACE} to include the source namespace"
        @update:model-value="onCustomFormat"
      />

      <SInput
        :model-value="streamPrefix"
        label="Stream Prefix"
        placeholder="e.g. production_"
        hint="Prepended to all stream names at the destination"
        @update:model-value="onStreamPrefix"
      />
    </div>
  </div>
</template>
