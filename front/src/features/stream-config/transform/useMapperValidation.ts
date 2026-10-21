import type { DestinationSyncMode, MapperTarget, SelectedField } from '@entities/connection'
import type { TransformIssue } from './columns'
import type { StepDraft } from './model'
import { validateMappers } from '@entities/connection'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { onBeforeUnmount, ref, watch } from 'vue'
import { schemaType } from './columns'
import { toStepJson } from './model'

export const VALIDATION_DEBOUNCE_MS = 400

export interface StepSample {
  output: Record<string, unknown> | undefined
  kept: boolean
  error: string
  compiled: boolean
}

export interface MapperValidationSource {
  target: () => MapperTarget
  namespace: () => string
  streamName: () => string
  steps: () => StepDraft[]
  selectedFields: () => SelectedField[]
  primaryKey: () => string[][]
  destinationSyncMode: () => DestinationSyncMode
  sample: () => Record<string, unknown> | undefined
  blocked: () => boolean
}

// Fields a step writes, keyed by name; a filter's result is keyed by ''.
function writtenFields(step: StepDraft): string[] {
  if (step.kind === 'filter')
    return ['']
  if (step.kind === 'rename')
    return step.entries.map(e => e.target ?? '')
  if (step.kind === 'drop')
    return []
  return step.entries.map(e => e.field)
}

function resultTypes(step: StepDraft, schema: Record<string, unknown> | undefined): Record<string, string> {
  const props = (schema?.properties ?? {}) as Record<string, unknown>
  const out: Record<string, string> = {}
  for (const field of writtenFields(step)) {
    if (field === '')
      out[field] = 'boolean'
    else if (field in props)
      out[field] = schemaType(props[field])
  }
  return out
}

// Checks the step list with ValidateMappers, once per prefix of the list, so
// each step gets its own result types and sample output.
export function useMapperValidation(source: MapperValidationSource) {
  const issues = ref<TransformIssue[]>([])
  const types = ref<Record<string, string>[]>([])
  const samples = ref<StepSample[]>([])
  const outputSchema = ref<Record<string, unknown>>()
  const loading = ref(false)
  const requestError = ref('')

  let generation = 0
  let timer: ReturnType<typeof setTimeout> | undefined

  function clear() {
    issues.value = []
    types.value = []
    samples.value = []
    outputSchema.value = undefined
    requestError.value = ''
    loading.value = false
  }

  async function run() {
    const id = ++generation
    const steps = source.steps()
    if (steps.length === 0) {
      clear()
      return
    }
    const mappers = steps.map(toStepJson)
    const base = {
      namespace: source.namespace(),
      streamName: source.streamName(),
      selectedFields: source.selectedFields(),
      primaryKey: source.primaryKey(),
      destinationSyncMode: source.destinationSyncMode(),
      sampleRecord: source.sample(),
    }
    loading.value = true
    try {
      const results = await Promise.all(mappers.map((_, i) =>
        validateMappers(source.target(), { ...base, mappers: mappers.slice(0, i + 1) })))
      if (id !== generation)
        return
      const full = results.at(-1)!
      issues.value = [
        ...full.errors.map(e => ({ ...e, severity: 'error' as const })),
        ...full.warnings.map(w => ({ ...w, severity: 'warning' as const })),
      ]
      types.value = results.map((r, i) => resultTypes(steps[i], r.outputSchema))
      samples.value = results.map(r => ({ output: r.sampleOutput, kept: r.sampleKept, error: r.sampleError, compiled: r.errors.length === 0 }))
      outputSchema.value = full.outputSchema
      requestError.value = ''
    }
    catch (e: unknown) {
      if (id === generation)
        requestError.value = getErrorMessage(e)
    }
    finally {
      if (id === generation)
        loading.value = false
    }
  }

  watch(
    () => JSON.stringify([
      source.target(),
      source.steps().map(toStepJson),
      source.sample(),
      source.selectedFields(),
      source.primaryKey(),
      source.destinationSyncMode(),
      source.blocked(),
    ]),
    () => {
      clearTimeout(timer)
      if (source.blocked()) {
        generation++
        clear()
        return
      }
      timer = setTimeout(run, VALIDATION_DEBOUNCE_MS)
    },
    { immediate: true },
  )

  onBeforeUnmount(() => {
    clearTimeout(timer)
    generation++
  })

  return { issues, types, samples, outputSchema, loading, requestError }
}
