<script setup lang="ts">
import type { Column } from './columns'
import type { StepDraft } from './model'
import type { StepSample } from './useMapperValidation'
import { SButton } from '@shared/ui'
import { Wand2 } from 'lucide-vue-next'
import { MAPPER_LABELS } from './model'

const props = defineProps<{
  columns: Column[]
  steps: StepDraft[]
  samples: StepSample[]
  modelValue: string
  parseError: string
  loading: boolean
  blocked: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

function exampleValue(column: Column): unknown {
  switch (column.type) {
    case 'integer':
      return 1
    case 'number':
      return 1.5
    case 'boolean':
      return true
    case 'timestamp':
      return '2026-01-01T12:00:00Z'
    case 'date':
      return '2026-01-01'
    case 'object':
      return {}
    case 'array':
      return []
    case 'string':
      return column.name
    default:
      return null
  }
}

function generate() {
  const record = Object.fromEntries(props.columns.map(c => [c.name, exampleValue(c)]))
  emit('update:modelValue', JSON.stringify(record, null, 2))
}
</script>

<template>
  <section class="space-y-2" data-testid="test-panel">
    <div class="flex items-center justify-between gap-2">
      <h4 class="text-sm font-semibold text-heading">
        Test
      </h4>
      <SButton variant="ghost" size="sm" data-testid="test-generate" @click="generate">
        <Wand2 class="w-3.5 h-3.5" /> Generate from schema
      </SButton>
    </div>
    <textarea
      data-testid="test-sample"
      rows="6"
      class="block w-full px-2.5 py-1.5 border rounded-md text-xs font-mono bg-surface text-heading placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary resize-y"
      :class="parseError ? 'border-danger' : 'border-border'"
      :value="modelValue"
      placeholder="Paste a sample record as JSON"
      aria-label="Sample record"
      @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
    />
    <p v-if="parseError" class="text-xs text-danger">
      {{ parseError }}
    </p>

    <p v-if="!modelValue.trim()" class="text-xs text-text-muted">
      Add a sample record to see what each step makes of it.
    </p>
    <p v-else-if="blocked" class="text-xs text-text-muted">
      Fix the errors in the steps to run the test.
    </p>
    <ol v-else-if="samples.length > 0" class="space-y-2" :class="loading ? 'opacity-60' : ''">
      <li v-for="(sample, i) in samples" :key="i" data-testid="test-step-output" class="rounded-md border border-border-light bg-surface p-2">
        <p class="text-xs font-medium text-text-secondary mb-1">
          {{ i + 1 }}. {{ steps[i] ? MAPPER_LABELS[steps[i].kind] : '' }}
        </p>
        <p v-if="!sample.compiled" class="text-xs text-text-muted">
          Not run: a step up to here has errors.
        </p>
        <p v-else-if="sample.error" class="text-xs text-danger">
          {{ sample.error }}
        </p>
        <p v-else-if="!sample.kept" class="text-xs text-warning">
          Filtered out
        </p>
        <pre v-else class="text-xs font-mono text-heading whitespace-pre-wrap break-all">{{ JSON.stringify(sample.output ?? {}, null, 2) }}</pre>
      </li>
    </ol>
  </section>
</template>
