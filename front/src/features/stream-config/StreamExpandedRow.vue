<script setup lang="ts">
import type { StreamRow } from './StreamFieldsTab.vue'
import type { StreamValidationError } from './useStreamValidation'
import { ref } from 'vue'
import StreamFieldsTab from './StreamFieldsTab.vue'
import StreamStateTab from './StreamStateTab.vue'
// Re-export StreamRow type for consumers
export type { StreamRow }

defineProps<{
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

const activeTab = ref<'fields' | 'state'>('fields')
</script>

<template>
  <div class="bg-surface-raised border-t border-border">
    <!-- Tab bar -->
    <div class="flex border-b border-border">
      <button
        class="px-4 py-2 text-sm font-medium transition-colors"
        :class="activeTab === 'fields'
          ? 'text-primary border-b-2 border-primary'
          : 'text-text-secondary hover:text-heading'"
        @click="activeTab = 'fields'"
      >
        Fields
      </button>
      <button
        class="px-4 py-2 text-sm font-medium transition-colors"
        :class="activeTab === 'state'
          ? 'text-primary border-b-2 border-primary'
          : 'text-text-secondary hover:text-heading'"
        @click="activeTab = 'state'"
      >
        State
      </button>
    </div>

    <!-- Tab content -->
    <StreamFieldsTab
      v-if="activeTab === 'fields'"
      :stream="stream"
      :schema="schema"
      :validation-errors="validationErrors"
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
