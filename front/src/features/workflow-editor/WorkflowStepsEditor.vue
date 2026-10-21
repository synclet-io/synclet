<script setup lang="ts">
import type { WorkflowStepInput } from '@entities/workflow'
import { CONFIGURE_STREAMS_FIRST } from '@features/stream-config/streamsConfigured'
import StreamsNotConfiguredBadge from '@features/stream-config/StreamsNotConfiguredBadge.vue'
import { wouldCreateCycle } from '@features/workflow-graph'
import { SButton, SCheckbox, SSelect } from '@shared/ui'
import { Trash2 } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { addStep, removeStep, setAfter } from './steps'

const props = defineProps<{
  modelValue: WorkflowStepInput[]
  /** Connections a step can run; one without configured streams cannot be added. */
  connections: { id: string, name: string, streamsConfigured?: boolean }[]
  readonly?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [steps: WorkflowStepInput[]]
}>()

const names = computed(() => new Map(props.connections.map(c => [c.id, c.name])))

function nameOf(connectionId: string): string {
  return names.value.get(connectionId) ?? connectionId
}

const newConnectionId = ref('')

const addOptions = computed(() => [
  { label: 'Choose a connection…', value: '' },
  ...props.connections
    .filter(c => !props.modelValue.some(s => s.connectionId === c.id))
    .map(c => c.streamsConfigured === false
      ? { label: `${c.name} (${CONFIGURE_STREAMS_FIRST})`, value: c.id, disabled: true, title: CONFIGURE_STREAMS_FIRST }
      : { label: c.name, value: c.id }),
])

const unconfigured = computed(() => new Set(props.connections.filter(c => c.streamsConfigured === false).map(c => c.id)))

function add() {
  emit('update:modelValue', addStep(props.modelValue, newConnectionId.value))
  newConnectionId.value = ''
}

function remove(connectionId: string) {
  emit('update:modelValue', removeStep(props.modelValue, connectionId))
}

function toggleAfter(step: WorkflowStepInput, upstream: string, checked: boolean) {
  const after = checked
    ? [...step.afterConnectionIds, upstream]
    : step.afterConnectionIds.filter(id => id !== upstream)
  emit('update:modelValue', setAfter(props.modelValue, step.connectionId, after))
}

function upstreamOptions(step: WorkflowStepInput) {
  return props.modelValue
    .filter(s => s.connectionId !== step.connectionId)
    .map(s => ({
      connectionId: s.connectionId,
      checked: step.afterConnectionIds.includes(s.connectionId),
      disabled: !step.afterConnectionIds.includes(s.connectionId) && wouldCreateCycle(props.modelValue, step.connectionId, s.connectionId),
    }))
}
</script>

<template>
  <div class="space-y-3">
    <p v-if="modelValue.length === 0" class="text-sm text-text-secondary">
      No steps yet. Add the connections this workflow runs.
    </p>
    <ol class="space-y-3">
      <li
        v-for="(step, index) in modelValue"
        :key="step.connectionId"
        class="rounded-lg border border-border p-3"
        :data-step="step.connectionId"
      >
        <div class="flex items-center justify-between gap-2">
          <span class="flex items-center gap-2 text-sm font-medium text-heading">
            <span><span class="text-text-muted">{{ index + 1 }}.</span> {{ nameOf(step.connectionId) }}</span>
            <StreamsNotConfiguredBadge v-if="unconfigured.has(step.connectionId)" :connection-id="step.connectionId" />
          </span>
          <SButton
            v-if="!readonly"
            variant="ghost"
            size="icon"
            :aria-label="`Remove step ${nameOf(step.connectionId)}`"
            :title="`Remove step ${nameOf(step.connectionId)}`"
            @click="remove(step.connectionId)"
          >
            <Trash2 class="w-4 h-4" />
          </SButton>
        </div>
        <fieldset v-if="modelValue.length > 1" class="mt-2">
          <legend class="text-xs font-medium text-text-secondary mb-1">
            Runs after
          </legend>
          <div v-if="!readonly" class="flex flex-wrap gap-x-4 gap-y-1" role="group" :aria-label="`Steps ${nameOf(step.connectionId)} runs after`">
            <SCheckbox
              v-for="option in upstreamOptions(step)"
              :key="option.connectionId"
              :model-value="option.checked"
              :disabled="option.disabled"
              :label="nameOf(option.connectionId)"
              size="sm"
              :data-after="option.connectionId"
              @update:model-value="toggleAfter(step, option.connectionId, $event)"
            />
          </div>
          <p v-else class="text-sm text-text-secondary">
            {{ step.afterConnectionIds.length ? step.afterConnectionIds.map(nameOf).join(', ') : 'Starts the run' }}
          </p>
        </fieldset>
      </li>
    </ol>
    <div v-if="!readonly" class="flex items-end gap-2">
      <div class="w-full sm:w-72">
        <SSelect v-model="newConnectionId" :options="addOptions" size="sm" aria-label="Connection to add" />
      </div>
      <SButton variant="secondary" size="sm" :disabled="!newConnectionId" @click="add">
        Add step
      </SButton>
    </div>
  </div>
</template>
