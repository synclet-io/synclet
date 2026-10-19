<script setup lang="ts">
import type { DropdownItem } from '@shared/ui/types'
import type { StreamRow } from '../catalogStreams'
import type { ColumnMark, TransformIssue } from './columns'
import type { MapperKind, StepDraft } from './model'
import { SBadge, SButton, SDropdown } from '@shared/ui'
import { AlertCircle, AlertTriangle, KeyRound, Plus } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { schemaColumns, schemaType, walkColumns } from './columns'
import { addStep, fromStepJson, MAPPER_KINDS, MAPPER_LABELS, moveStep, removeStep, toStepJson, updateStep } from './model'
import StepCard from './StepCard.vue'
import TestPanel from './TestPanel.vue'
import { useMapperValidation } from './useMapperValidation'

const props = defineProps<{
  stream: StreamRow
  connectionId: string
}>()

const emit = defineEmits<{
  'update:stream': [stream: StreamRow]
}>()

const steps = ref<StepDraft[]>(props.stream.mappers.map(fromStepJson))

watch(() => props.stream.mappers, (mappers) => {
  if (JSON.stringify(mappers) !== JSON.stringify(steps.value.map(toStepJson)))
    steps.value = mappers.map(fromStepJson)
})

const inputColumns = computed(() => schemaColumns(props.stream.jsonSchema))
const primaryKey = computed(() => props.stream.sourceDefinedPrimaryKey.length > 0
  ? props.stream.sourceDefinedPrimaryKey
  : props.stream.primaryKey)

const walk = computed(() => walkColumns(
  inputColumns.value,
  steps.value,
  props.stream.selectedFields,
  primaryKey.value,
  props.stream.destinationSyncMode,
))

const hasBrowserErrors = computed(() => walk.value.issues.some(i => i.severity === 'error'))

// The stream carries the count of browser errors so the page refuses to save
// steps whose JSON cannot hold what the cards show, such as a field listed twice.
function commit(next: StepDraft[]) {
  steps.value = next
  const transformErrors = walk.value.issues.filter(i => i.severity === 'error').length
  emit('update:stream', { ...props.stream, mappers: next.map(toStepJson), transformErrors })
}

const sampleText = ref('')
const sampleParse = computed<{ record?: Record<string, unknown>, error: string }>(() => {
  const text = sampleText.value.trim()
  if (!text)
    return { error: '' }
  try {
    const parsed: unknown = JSON.parse(text)
    if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed))
      return { error: 'The sample must be a JSON object.' }
    return { record: parsed as Record<string, unknown>, error: '' }
  }
  catch {
    return { error: 'The sample is not valid JSON.' }
  }
})

const validation = useMapperValidation({
  connectionId: () => props.connectionId,
  namespace: () => props.stream.namespace,
  streamName: () => props.stream.name,
  steps: () => steps.value,
  selectedFields: () => props.stream.selectedFields,
  primaryKey: () => primaryKey.value,
  destinationSyncMode: () => props.stream.destinationSyncMode,
  sample: () => sampleParse.value.record,
  blocked: () => hasBrowserErrors.value,
})

function issuesFor(step: number): TransformIssue[] {
  const seen = new Set<string>()
  return [...walk.value.issues, ...validation.issues.value]
    .filter(i => i.step === step)
    .filter((i) => {
      const key = `${i.severity}:${i.message.toLowerCase()}`
      if (seen.has(key))
        return false
      seen.add(key)
      return true
    })
}

const selectionIssues = computed(() => issuesFor(-1))

function isSelected(name: string): boolean {
  const selected = props.stream.selectedFields
  return selected.length === 0 || selected.some(sf => sf.fieldPath[0] === name)
}

const outputColumns = computed(() => {
  const serverProps = validation.outputSchema.value?.properties as Record<string, unknown> | undefined
  return walk.value.output.map((c) => {
    const serverType = serverProps && c.name in serverProps ? schemaType(serverProps[c.name]) : undefined
    return { ...c, type: c.type === 'any' && serverType ? serverType : c.type }
  })
})

const markVariant: Record<ColumnMark, 'success' | 'info' | 'warning' | 'gray'> = {
  new: 'success',
  renamed: 'info',
  retyped: 'warning',
  dropped: 'gray',
}

const markLabel: Record<ColumnMark, string> = {
  new: 'new',
  renamed: 'renamed',
  retyped: 'type changed',
  dropped: 'dropped',
}

const addItems: DropdownItem[] = MAPPER_KINDS.map(kind => ({ label: MAPPER_LABELS[kind], value: kind }))

function onAdd(item: DropdownItem) {
  commit(addStep(steps.value, item.value as MapperKind))
}

const dragFrom = ref<number | null>(null)

function onDrop(to: number) {
  const from = dragFrom.value
  dragFrom.value = null
  if (from === null || from === to)
    return
  commit(moveStep(steps.value, from, to - from))
}
</script>

<template>
  <div class="p-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,2fr)_minmax(0,1fr)]" data-testid="transform-tab">
    <section class="space-y-2 min-w-0">
      <h4 class="text-sm font-semibold text-heading">
        Input columns
      </h4>
      <ul class="rounded-lg border border-border bg-surface divide-y divide-border-light text-sm">
        <li
          v-for="c in inputColumns"
          :key="c.name"
          data-testid="input-column"
          class="flex items-center justify-between gap-2 px-3 py-1.5"
          :class="isSelected(c.name) ? '' : 'opacity-50'"
          :title="isSelected(c.name) ? undefined : 'Not selected; still usable in steps'"
        >
          <span class="font-mono text-heading truncate">{{ c.name }}</span>
          <span class="text-xs text-text-muted font-mono">{{ c.type }}</span>
        </li>
        <li v-if="inputColumns.length === 0" class="px-3 py-2 text-text-muted">
          The stream's schema has no columns.
        </li>
      </ul>
    </section>

    <section class="space-y-2 min-w-0">
      <div class="flex items-center justify-between gap-2">
        <h4 class="text-sm font-semibold text-heading">
          Steps
        </h4>
        <SDropdown :items="addItems" align="right" @select="onAdd">
          <template #trigger="{ attrs }">
            <SButton v-bind="attrs" variant="secondary" size="sm" data-testid="add-step">
              <Plus class="w-3.5 h-3.5" /> Add step
            </SButton>
          </template>
        </SDropdown>
      </div>

      <ul v-if="selectionIssues.length > 0" class="space-y-1">
        <li
          v-for="(issue, i) in selectionIssues"
          :key="i"
          data-testid="selection-issue"
          class="flex items-start gap-1.5 text-xs"
          :class="issue.severity === 'error' ? 'text-danger' : 'text-warning'"
        >
          <AlertCircle v-if="issue.severity === 'error'" class="w-3.5 h-3.5 mt-px shrink-0" aria-hidden="true" />
          <AlertTriangle v-else class="w-3.5 h-3.5 mt-px shrink-0" aria-hidden="true" />
          <span>{{ issue.message }}</span>
        </li>
      </ul>

      <p v-if="steps.length === 0" class="rounded-lg border border-dashed border-border px-4 py-6 text-center text-sm text-text-muted">
        No steps. Records go to the destination as the source sends them.
      </p>

      <StepCard
        v-for="(step, i) in steps"
        :key="step.id"
        :step="step"
        :index="i"
        :count="steps.length"
        :columns="walk.stepInputs[i] ?? []"
        :issues="issuesFor(i)"
        :result-types="validation.types.value[i] ?? {}"
        @update="commit(updateStep(steps, i, $event))"
        @move="commit(moveStep(steps, i, $event))"
        @remove="commit(removeStep(steps, i))"
        @dragstart="dragFrom = i"
        @drop="onDrop(i)"
      />

      <p v-if="validation.requestError.value" class="text-xs text-danger">
        Could not check the expressions: {{ validation.requestError.value }}
      </p>
    </section>

    <section class="space-y-4 min-w-0">
      <div class="space-y-2">
        <h4 class="text-sm font-semibold text-heading">
          Output columns
        </h4>
        <ul class="rounded-lg border border-border bg-surface divide-y divide-border-light text-sm">
          <li
            v-for="c in outputColumns"
            :key="c.name"
            data-testid="output-column"
            class="px-3 py-1.5"
            :class="c.marks.includes('dropped') ? 'opacity-50' : ''"
          >
            <div class="flex items-center justify-between gap-2">
              <span class="flex items-center gap-1 min-w-0">
                <KeyRound v-if="c.primaryKey" class="w-3 h-3 text-text-muted shrink-0" aria-label="Primary key" />
                <span class="font-mono text-heading truncate" :class="c.marks.includes('dropped') ? 'line-through' : ''">{{ c.name }}</span>
              </span>
              <span class="text-xs text-text-muted font-mono">{{ c.type === 'any' ? '?' : c.type }}</span>
            </div>
            <div v-if="c.marks.length > 0" class="flex flex-wrap items-center gap-1 mt-1">
              <SBadge v-for="m in c.marks" :key="m" :variant="markVariant[m]" :label="markLabel[m]" />
              <span v-if="c.renamedFrom" class="text-xs text-text-muted">from <span class="font-mono">{{ c.renamedFrom }}</span></span>
            </div>
          </li>
        </ul>
      </div>

      <TestPanel
        v-model="sampleText"
        :columns="inputColumns"
        :steps="steps"
        :samples="validation.samples.value"
        :parse-error="sampleParse.error"
        :loading="validation.loading.value"
        :blocked="hasBrowserErrors"
      />
    </section>
  </div>
</template>
