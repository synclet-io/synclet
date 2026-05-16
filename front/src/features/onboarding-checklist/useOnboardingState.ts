import type { ComputedRef, MaybeRef } from 'vue'
import { useAuth } from '@entities/auth'
import { useConnections } from '@entities/connection'
import { useManagedConnectors } from '@entities/connector'
import { useDestinations } from '@entities/destination'
import { useRepositories } from '@entities/repository'
import { useSources } from '@entities/source'
import { useWorkspaceStats } from '@entities/stats'
import { computed, ref, watch } from 'vue'

export type OnboardingStepId
  = | 'repository'
    | 'connector'
    | 'source'
    | 'destination'
    | 'connection'
    | 'first-sync'

export interface OnboardingStep {
  id: OnboardingStepId
  title: string
  description: string
  cta: { label: string, to: string }
  done: boolean
}

const STORAGE_PREFIX = 'synclet:onboarding-dismissed'

function dismissKey(workspaceId: string, userId: string): string {
  return `${STORAGE_PREFIX}:${workspaceId}:${userId}`
}

export interface OnboardingState {
  steps: ComputedRef<OnboardingStep[]>
  completedCount: ComputedRef<number>
  totalSteps: ComputedRef<number>
  allComplete: ComputedRef<boolean>
  isDismissed: ComputedRef<boolean>
  shouldShow: ComputedRef<boolean>
  dismiss: () => void
  reset: () => void
}

interface OnboardingSignals {
  hasRepository: MaybeRef<boolean>
  hasConnector: MaybeRef<boolean>
  hasSource: MaybeRef<boolean>
  hasDestination: MaybeRef<boolean>
  hasConnection: MaybeRef<boolean>
  hasSuccessfulSync: MaybeRef<boolean>
}

/**
 * Pure derivation of the onboarding state from boolean signals. Exposed
 * separately so unit tests can drive it without mounting query hooks.
 */
export function deriveOnboardingState(
  signals: OnboardingSignals,
  dismissed: MaybeRef<boolean>,
): OnboardingState {
  function toBool(value: MaybeRef<boolean>): boolean {
    return typeof value === 'boolean' ? value : value.value
  }

  const steps = computed<OnboardingStep[]>(() => [
    {
      id: 'repository',
      title: 'Add a repository',
      description: 'Connect a connector registry to browse and install connectors.',
      cta: { label: 'Manage repositories', to: '/settings/connectors' },
      done: toBool(signals.hasRepository),
    },
    {
      id: 'connector',
      title: 'Install a connector',
      description: 'Pick a source or destination connector from the catalog.',
      cta: { label: 'Browse catalog', to: '/connectors/catalog' },
      done: toBool(signals.hasConnector),
    },
    {
      id: 'source',
      title: 'Configure a source',
      description: 'Point Synclet at the database or API you want to sync from.',
      cta: { label: 'Add source', to: '/sources/new' },
      done: toBool(signals.hasSource),
    },
    {
      id: 'destination',
      title: 'Configure a destination',
      description: 'Tell Synclet where to land the synced data.',
      cta: { label: 'Add destination', to: '/destinations/new' },
      done: toBool(signals.hasDestination),
    },
    {
      id: 'connection',
      title: 'Create a connection',
      description: 'Wire the source to the destination and pick streams.',
      cta: { label: 'New connection', to: '/connections/new' },
      done: toBool(signals.hasConnection),
    },
    {
      id: 'first-sync',
      title: 'Observe your first sync',
      description: 'Trigger a sync from the connection page and watch it succeed.',
      cta: { label: 'Open connections', to: '/connections' },
      done: toBool(signals.hasSuccessfulSync),
    },
  ])

  const completedCount = computed(() => steps.value.filter(s => s.done).length)
  const totalSteps = computed(() => steps.value.length)
  const allComplete = computed(() => completedCount.value === totalSteps.value)
  const isDismissed = computed(() => toBool(dismissed))
  const shouldShow = computed(() => !isDismissed.value)

  return {
    steps,
    completedCount,
    totalSteps,
    allComplete,
    isDismissed,
    shouldShow,
    dismiss: () => { /* overridden by useOnboardingState */ },
    reset: () => { /* overridden by useOnboardingState */ },
  }
}

/**
 * useOnboardingState wires the pure deriver into live entity queries and
 * persists the dismissal flag in localStorage keyed by workspace + user.
 */
export function useOnboardingState(): OnboardingState {
  const { currentWorkspaceId, user } = useAuth()

  const { data: repositories } = useRepositories()
  const { data: managed } = useManagedConnectors()
  const { data: sourcesPage } = useSources()
  const { data: destinationsPage } = useDestinations()
  const { data: connectionsPage } = useConnections()
  const { data: stats } = useWorkspaceStats('30d')

  const dismissed = ref(false)

  function readDismissed(): boolean {
    const ws = currentWorkspaceId.value
    const usr = user.value?.id
    if (!ws || !usr)
      return false
    try {
      return localStorage.getItem(dismissKey(ws, usr)) === '1'
    }
    catch {
      return false
    }
  }

  watch(
    [currentWorkspaceId, () => user.value?.id],
    () => { dismissed.value = readDismissed() },
    { immediate: true },
  )

  function dismiss() {
    const ws = currentWorkspaceId.value
    const usr = user.value?.id
    if (!ws || !usr)
      return
    try {
      localStorage.setItem(dismissKey(ws, usr), '1')
    }
    catch { /* storage unavailable, fall back to in-memory only */ }
    dismissed.value = true
  }

  function reset() {
    const ws = currentWorkspaceId.value
    const usr = user.value?.id
    if (ws && usr) {
      try {
        localStorage.removeItem(dismissKey(ws, usr))
      }
      catch { /* ignore */ }
    }
    dismissed.value = false
  }

  const state = deriveOnboardingState(
    {
      hasRepository: computed(() => (repositories.value?.length ?? 0) > 0),
      hasConnector: computed(() => (managed.value?.length ?? 0) > 0),
      hasSource: computed(() => (sourcesPage.value?.items?.length ?? 0) > 0),
      hasDestination: computed(() => (destinationsPage.value?.items?.length ?? 0) > 0),
      hasConnection: computed(() => (connectionsPage.value?.items?.length ?? 0) > 0),
      hasSuccessfulSync: computed(() => (stats.value?.totalSyncs ?? 0) > 0),
    },
    dismissed,
  )

  return { ...state, dismiss, reset }
}
