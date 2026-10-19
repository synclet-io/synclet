<script setup lang="ts">
import type { StreamRow } from './catalogStreams'
import type { StreamValidationError } from './useStreamValidation'
import { STabs } from '@shared/ui'
import { computed, ref } from 'vue'
import StreamFieldsTab from './StreamFieldsTab.vue'
import StreamStateTab from './StreamStateTab.vue'
import TransformTab from './transform/TransformTab.vue'
// Re-export StreamRow type for consumers
export type { StreamRow }

const props = defineProps<{
  stream: StreamRow
  schema: Record<string, any>
  validationErrors: StreamValidationError[]
  connectionId: string
  stateData: string | undefined
  stateUpdatedAt: Date | undefined
}>()

const emit = defineEmits<{
  'update:stream': [stream: StreamRow]
}>()

// The Transform tab checks steps against a saved connection, so it is offered
// only once the stream belongs to one.
const tabs = computed(() => [
  { name: 'Fields', value: 'fields' },
  ...(props.connectionId ? [{ name: 'Transform', value: 'transform' }] : []),
  { name: 'State', value: 'state' },
])
const activeTab = ref('fields')
</script>

<template>
  <div class="bg-surface-raised border-t border-border">
    <STabs v-model="activeTab" :tabs="tabs" class="!mb-0" />

    <StreamFieldsTab
      v-if="activeTab === 'fields'"
      :stream="stream"
      :schema="schema"
      :validation-errors="validationErrors"
      @update:stream="emit('update:stream', $event)"
    />
    <TransformTab
      v-if="activeTab === 'transform'"
      :stream="stream"
      :connection-id="connectionId"
      @update:stream="emit('update:stream', $event)"
    />
    <StreamStateTab
      v-if="activeTab === 'state'"
      :connection-id="connectionId"
      :stream-name="stream.name"
      :stream-namespace="stream.namespace"
      :state-data="stateData"
      :updated-at="stateUpdatedAt"
    />
  </div>
</template>
