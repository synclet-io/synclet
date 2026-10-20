<script setup lang="ts">
import type { WizardState } from './useWizardState'
import { useConnectionTags } from '@entities/connection'
import { useManagedConnector } from '@entities/connector'
import { SButton, SCard, STagInput } from '@shared/ui'
import { computed } from 'vue'
import { formatConnector } from './connectorLabel'

const props = defineProps<{
  state: WizardState
  tagsError?: string
}>()

const emit = defineEmits<{
  'goTo': [step: number]
  'update:tags': [tags: string[]]
}>()

const { data: workspaceTags } = useConnectionTags()
const tagSuggestions = computed(() => (workspaceTags.value ?? []).map(t => t.tag))

const { data: sourceConnector } = useManagedConnector(computed(() => props.state.source?.managedConnectorId ?? ''))
const { data: destinationConnector } = useManagedConnector(computed(() => props.state.destination?.managedConnectorId ?? ''))

function formatSchedule(schedule: string): string {
  return schedule || 'Manual'
}

function formatPolicy(policy: string): string {
  switch (policy) {
    case 'propagate': return 'Propagate changes'
    case 'ignore': return 'Ignore changes'
    case 'pause': return 'Pause on changes'
    default: return policy
  }
}

function formatNamespace(ns: string): string {
  switch (ns) {
    case 'source': return 'Source namespace'
    case 'destination': return 'Destination default'
    case 'custom': return 'Custom format'
    default: return ns
  }
}
</script>

<template>
  <div class="space-y-4">
    <!-- Source -->
    <SCard>
      <div class="flex items-center justify-between mb-3">
        <h3 class="text-base font-semibold text-heading">
          Source
        </h3>
        <SButton variant="ghost" size="sm" @click="$emit('goTo', 1)">
          Edit
        </SButton>
      </div>
      <dl class="space-y-2 text-sm">
        <div class="flex gap-2">
          <dt class="text-text-secondary w-32">
            Name
          </dt>
          <dd class="text-heading">
            {{ state.source?.name || '-' }}
          </dd>
        </div>
        <div class="flex gap-2">
          <dt class="text-text-secondary w-32">
            Connector
          </dt>
          <dd class="text-heading">
            {{ formatConnector(sourceConnector) || '-' }}
          </dd>
        </div>
      </dl>
    </SCard>

    <!-- Destination -->
    <SCard>
      <div class="flex items-center justify-between mb-3">
        <h3 class="text-base font-semibold text-heading">
          Destination
        </h3>
        <SButton variant="ghost" size="sm" @click="$emit('goTo', 2)">
          Edit
        </SButton>
      </div>
      <dl class="space-y-2 text-sm">
        <div class="flex gap-2">
          <dt class="text-text-secondary w-32">
            Name
          </dt>
          <dd class="text-heading">
            {{ state.destination?.name || '-' }}
          </dd>
        </div>
        <div class="flex gap-2">
          <dt class="text-text-secondary w-32">
            Connector
          </dt>
          <dd class="text-heading">
            {{ formatConnector(destinationConnector) || '-' }}
          </dd>
        </div>
      </dl>
    </SCard>

    <!-- Settings -->
    <SCard>
      <div class="flex items-center justify-between mb-3">
        <h3 class="text-base font-semibold text-heading">
          Settings
        </h3>
        <SButton variant="ghost" size="sm" @click="$emit('goTo', 3)">
          Edit
        </SButton>
      </div>
      <dl class="space-y-2 text-sm">
        <div class="flex gap-2">
          <dt class="text-text-secondary w-32">
            Name
          </dt>
          <dd class="text-heading">
            {{ state.name }}
          </dd>
        </div>
        <div class="flex gap-2">
          <dt class="text-text-secondary w-32">
            Schedule
          </dt>
          <dd class="text-heading">
            {{ formatSchedule(state.schedule) }}
          </dd>
        </div>
        <div class="flex gap-2">
          <dt class="text-text-secondary w-32">
            Schema Policy
          </dt>
          <dd class="text-heading">
            {{ formatPolicy(state.schemaChangePolicy) }}
          </dd>
        </div>
        <div class="flex gap-2">
          <dt class="text-text-secondary w-32">
            Namespace
          </dt>
          <dd class="text-heading">
            {{ formatNamespace(state.namespaceDefinition) }}
          </dd>
        </div>
        <div v-if="state.streamPrefix" class="flex gap-2">
          <dt class="text-text-secondary w-32">
            Stream Prefix
          </dt>
          <dd class="text-heading">
            {{ state.streamPrefix }}
          </dd>
        </div>
        <div class="flex gap-2">
          <dt class="text-text-secondary w-32">
            Max Attempts
          </dt>
          <dd class="text-heading">
            {{ state.maxAttempts }}
          </dd>
        </div>
      </dl>
    </SCard>

    <!-- Streams -->
    <SCard>
      <div class="flex items-center justify-between mb-3">
        <h3 class="text-base font-semibold text-heading">
          Streams
        </h3>
        <SButton variant="ghost" size="sm" @click="$emit('goTo', 4)">
          Edit
        </SButton>
      </div>
      <p v-if="state.streamsSkipped || state.streams.length === 0" class="text-sm text-text-secondary">
        No streams configured -- you can configure streams after creation.
      </p>
      <p v-else class="text-sm text-heading">
        {{ state.streams.length }} streams enabled
      </p>
    </SCard>

    <!-- Tags -->
    <SCard>
      <STagInput
        :model-value="state.tags"
        :suggestions="tagSuggestions"
        :error="tagsError"
        label="Tags"
        placeholder="Add a tag and press Enter"
        @update:model-value="emit('update:tags', $event)"
      />
    </SCard>
  </div>
</template>
