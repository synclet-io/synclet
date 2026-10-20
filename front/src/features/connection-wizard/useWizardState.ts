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
  // Step 5
  tags: string[]
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
    tags: [],
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

  const committedConnectionId = ref<string | null>(null)

  const hasAnyState = computed(() => !!state.value.sourceId || !!state.value.destinationId)
  const preselectedSnapshot = ref<string | null>(null)
  const hasUnsavedChanges = computed(() => {
    if (committedConnectionId.value)
      return false
    if (preselectedSnapshot.value !== null)
      return JSON.stringify(state.value) !== preselectedSnapshot.value
    return hasAnyState.value
  })

  function markCommitted(connectionId: string) {
    committedConnectionId.value = connectionId
  }

  function next() {
    if (canProceed.value && currentStep.value < totalSteps)
      currentStep.value++
  }

  function back() {
    if (currentStep.value > 1)
      currentStep.value--
  }

  function goTo(step: number, options?: { force?: boolean }) {
    if (step < 1 || step > totalSteps)
      return
    if (options?.force || step <= currentStep.value)
      currentStep.value = step
  }

  /** Applies entities picked before the wizard opened and jumps to the first step still missing a choice. */
  function preselect(picks: { source?: Source, destination?: Destination }) {
    if (picks.source) {
      state.value.sourceId = picks.source.id
      state.value.source = picks.source
    }
    if (picks.destination) {
      state.value.destinationId = picks.destination.id
      state.value.destination = picks.destination
    }
    preselectedSnapshot.value = JSON.stringify(state.value)
    goTo(!state.value.sourceId ? 1 : !state.value.destinationId ? 2 : 3, { force: true })
  }

  return { currentStep, totalSteps, state, canProceed, hasAnyState, hasUnsavedChanges, committedConnectionId, markCommitted, next, back, goTo, preselect }
}
