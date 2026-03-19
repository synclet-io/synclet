<script setup lang="ts">
import type { NamespaceDefinition, SchemaChangePolicy } from '@entities/connection'
import { useConnection, useUpdateConnection } from '@entities/connection'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SButton, SCard, SCronEditor, SInput, SSelect, useToast } from '@shared/ui'
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const id = route.params.id as string
const toast = useToast()

const { data: connection } = useConnection(id)
const updateMutation = useUpdateConnection()

const schemaChangePolicyOptions = [
  { label: 'Propagate changes', value: 'propagate' },
  { label: 'Ignore changes', value: 'ignore' },
  { label: 'Pause on change', value: 'pause' },
]

const namespaceOptions = [
  { label: 'Source namespace', value: 'source' },
  { label: 'Destination default', value: 'destination' },
  { label: 'Custom format', value: 'custom' },
]

const editForm = ref({
  name: '',
  schedule: '',
  schemaChangePolicy: 'propagate' as SchemaChangePolicy,
  maxAttempts: 3,
  namespaceDefinition: 'source' as NamespaceDefinition,
  customNamespaceFormat: '',
  streamPrefix: '',
})

watch(() => connection.value, (conn) => {
  if (conn) {
    editForm.value = {
      name: conn.name || '',
      schedule: conn.schedule || '',
      schemaChangePolicy: conn.schemaChangePolicy || 'propagate',
      maxAttempts: conn.maxAttempts || 3,
      namespaceDefinition: conn.namespaceDefinition || 'source',
      customNamespaceFormat: conn.customNamespaceFormat || '',
      streamPrefix: conn.streamPrefix || '',
    }
  }
}, { immediate: true })

async function saveSettings() {
  try {
    await updateMutation.mutateAsync({
      id,
      name: editForm.value.name,
      schedule: editForm.value.schedule,
      schemaChangePolicy: editForm.value.schemaChangePolicy,
      maxAttempts: Number(editForm.value.maxAttempts),
      namespaceDefinition: editForm.value.namespaceDefinition,
      customNamespaceFormat: editForm.value.customNamespaceFormat || undefined,
      streamPrefix: editForm.value.streamPrefix || undefined,
    })
    toast.success('Settings updated')
  }
  catch (e: unknown) {
    toast.error(`Error: ${getErrorMessage(e)}`)
  }
}
</script>

<template>
  <div class="space-y-4">
    <!-- Connection Name -->
    <SCard title="Connection Name">
      <div class="p-4">
        <SInput
          v-model="editForm.name"
          label="Name"
          placeholder="Enter connection name"
        />
      </div>
    </SCard>

    <!-- Schedule -->
    <SCard title="Schedule">
      <div class="p-4">
        <SCronEditor v-model="editForm.schedule" />
      </div>
    </SCard>

    <!-- Schema Change Policy -->
    <SCard title="Schema Change Policy">
      <div class="p-4">
        <SSelect
          v-model="editForm.schemaChangePolicy"
          :options="schemaChangePolicyOptions"
          label="When source schema changes"
        />
      </div>
    </SCard>

    <!-- Retry Policy -->
    <SCard title="Retry Policy">
      <div class="p-4">
        <SInput
          v-model="editForm.maxAttempts"
          type="number"
          min="1"
          max="10"
          label="Max Attempts"
        />
      </div>
    </SCard>

    <!-- Namespace -->
    <SCard title="Namespace">
      <div class="space-y-4 p-4">
        <div class="grid grid-cols-2 gap-4">
          <SSelect
            v-model="editForm.namespaceDefinition"
            :options="namespaceOptions"
            label="Namespace Mode"
          />
          <SInput
            v-model="editForm.streamPrefix"
            label="Stream Prefix"
            placeholder="e.g. production_"
          />
        </div>

        <SInput
          v-if="editForm.namespaceDefinition === 'custom'"
          v-model="editForm.customNamespaceFormat"
          label="Custom Namespace Format"
          placeholder="e.g. staging_${SOURCE_NAMESPACE}"
        />
      </div>
    </SCard>

    <div class="flex justify-end">
      <SButton
        variant="primary"
        :loading="updateMutation.isPending.value"
        @click="saveSettings"
      >
        Save Settings
      </SButton>
    </div>
  </div>
</template>
