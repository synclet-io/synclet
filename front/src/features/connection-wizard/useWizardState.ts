import type { ConfiguredStream, NamespaceDefinition, SchemaChangePolicy } from '@entities/connection'
import type { Destination } from '@entities/destination'
import type { Source } from '@entities/source'
import { computed, ref } from 'vue'

export interface WizardState {
  // Step 1
  sourceId: string
  source: Source | null
  // Step 2
  destinationId: string
  destination: Destination | null
  // Step 3
  name: string
  schedule: string
  schemaChangePolicy: SchemaChangePolicy
  maxAttempts: number
  namespaceDefinition: NamespaceDefinition
  customNamespaceFormat: string
  streamPrefix: string
  // Step 4
  discoveredCatalog: Record<string, unknown> | null
  streams: ConfiguredStream[]
  streamsSkipped: boolean
}

export const WIZARD_STEPS = [
  { label: 'Source' },
  { label: 'Destination' },
  { label: 'Settings' },
  { label: 'Streams' },
  { label: 'Review' },
]

export function useWizardState() {
  const currentStep = ref(1)
  const totalSteps = WIZARD_STEPS.length

  const state = ref<WizardState>({
    sourceId: '',
    source: null,
    destinationId: '',
    destination: null,
    name: '',
    schedule: '',
    schemaChangePolicy: 'propagate',
    maxAttempts: 3,
    namespaceDefinition: 'source',
    customNamespaceFormat: '',
    streamPrefix: '',
    discoveredCatalog: null,
    streams: [],
    streamsSkipped: false,
  })

  const canProceed = computed(() => {
    switch (currentStep.value) {
      case 1: return !!state.value.sourceId
      case 2: return !!state.value.destinationId
      case 3: return !!state.value.name.trim()
      case 4: return state.value.streams.length > 0 || state.value.streamsSkipped
      case 5: return true
      default: return false
    }
  })

  const hasAnyState = computed(() => !!state.value.sourceId || !!state.value.destinationId)

  function next() {
    if (canProceed.value && currentStep.value < totalSteps)
      currentStep.value++
  }

  function back() {
    if (currentStep.value > 1)
      currentStep.value--
  }

  function goTo(step: number) {
    if (step >= 1 && step <= currentStep.value)
      currentStep.value = step
  }

  return { currentStep, totalSteps, state, canProceed, hasAnyState, next, back, goTo }
}
