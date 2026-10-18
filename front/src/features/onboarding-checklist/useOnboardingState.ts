import type { ComputedRef, MaybeRef } from 'vue'
import { useAuth } from '@entities/auth'
import { listConnections } from '@entities/connection/api'
import { listDestinations } from '@entities/destination/api'
import { listJobs } from '@entities/job/api'
import { listSources } from '@entities/source/api'
import { connectionKeys, destinationKeys, jobKeys, sourceKeys } from '@shared/lib/queryKeys'
import { useQuery } from '@tanstack/vue-query'
import { computed, ref, watch } from 'vue'

// Onboarding starts at "configure a source" because default registries are
// auto-seeded on workspace creation and their connectors auto-install on sync
// — neither step is a meaningful checkbox for the user.
export type OnboardingStepId
  = | 'source'
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

  const workspaceId = computed(() => currentWorkspaceId.value ?? '')
  const enabled = computed(() => !!workspaceId.value && !dismissed.value)
  const probe = { pageSize: 1 }

  const { data: sourcesPage } = useQuery({
    queryKey: computed(() => [...sourceKeys.list(workspaceId.value), 'onboarding']),
    queryFn: () => listSources(probe),
    enabled,
  })
  const { data: destinationsPage } = useQuery({
    queryKey: computed(() => [...destinationKeys.list(workspaceId.value), 'onboarding']),
    queryFn: () => listDestinations(probe),
    enabled,
  })
  const { data: connectionsPage } = useQuery({
    queryKey: computed(() => [...connectionKeys.list(workspaceId.value), 'onboarding']),
    queryFn: () => listConnections(probe),
    enabled,
  })
  const { data: completedJobsPage } = useQuery({
    queryKey: computed(() => [...jobKeys.list(workspaceId.value, ''), 'onboarding']),
    queryFn: () => listJobs('', { ...probe, statuses: ['completed'] }),
    enabled,
  })

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
      hasSource: computed(() => (sourcesPage.value?.total ?? 0) > 0),
      hasDestination: computed(() => (destinationsPage.value?.total ?? 0) > 0),
      hasConnection: computed(() => (connectionsPage.value?.total ?? 0) > 0),
      hasSuccessfulSync: computed(() => (completedJobsPage.value?.total ?? 0) > 0),
    },
    dismissed,
  )

  return { ...state, dismiss, reset }
}
