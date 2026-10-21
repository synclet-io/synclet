<script setup lang="ts">
import type { NamespaceDefinition, SchemaChangePolicy } from '@entities/connection'
import DestinationNamingSettings from '@features/destination-naming/DestinationNamingSettings.vue'
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
</script>

<template>
  <div class="max-w-lg mx-auto space-y-5">
    <SInput
      :model-value="name"
      label="Connection Name"
      placeholder="My Connection"
      tooltip="A friendly label for this connection. Often combines the source and destination, e.g. 'Production Postgres → Snowflake'."
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
      tooltip="What Synclet does when the source schema drifts: propagate applies changes automatically, pause stops syncs until you review, ignore continues but logs the drift."
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
        Destination names
      </h3>

      <DestinationNamingSettings
        :namespace-definition="namespaceDefinition"
        :custom-namespace-format="customNamespaceFormat"
        :stream-prefix="streamPrefix"
        @update:namespace-definition="emit('update:namespaceDefinition', $event)"
        @update:custom-namespace-format="emit('update:customNamespaceFormat', $event)"
        @update:stream-prefix="emit('update:streamPrefix', $event)"
      />
    </div>
  </div>
</template>
