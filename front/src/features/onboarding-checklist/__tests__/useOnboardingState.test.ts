import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { deriveOnboardingState } from '../useOnboardingState'

function allFalse() {
  return {
    hasSource: false,
    hasDestination: false,
    hasConnection: false,
    hasSuccessfulSync: false,
  }
}

describe('deriveOnboardingState', () => {
  it('emits four steps in onboarding order', () => {
    const state = deriveOnboardingState(allFalse(), false)
    const ids = state.steps.value.map(s => s.id)
    expect(ids).toEqual([
      'source',
      'destination',
      'connection',
      'first-sync',
    ])
  })

  it('starts with zero complete on a fresh workspace', () => {
    const state = deriveOnboardingState(allFalse(), false)
    expect(state.completedCount.value).toBe(0)
    expect(state.totalSteps.value).toBe(4)
    expect(state.allComplete.value).toBe(false)
  })

  it('marks each step done when its signal flips to true', () => {
    const state = deriveOnboardingState({
      hasSource: true,
      hasDestination: true,
      hasConnection: false,
      hasSuccessfulSync: false,
    }, false)
    expect(state.completedCount.value).toBe(2)
    expect(state.steps.value.find(s => s.id === 'source')?.done).toBe(true)
    expect(state.steps.value.find(s => s.id === 'destination')?.done).toBe(true)
    expect(state.steps.value.find(s => s.id === 'connection')?.done).toBe(false)
  })

  it('reports allComplete when all four signals are true', () => {
    const state = deriveOnboardingState({
      hasSource: true,
      hasDestination: true,
      hasConnection: true,
      hasSuccessfulSync: true,
    }, false)
    expect(state.completedCount.value).toBe(4)
    expect(state.allComplete.value).toBe(true)
  })

  it('reacts to changing refs', () => {
    const hasSource = ref(false)
    const state = deriveOnboardingState({
      hasSource,
      hasDestination: false,
      hasConnection: false,
      hasSuccessfulSync: false,
    }, false)
    expect(state.completedCount.value).toBe(0)
    hasSource.value = true
    expect(state.completedCount.value).toBe(1)
    expect(state.steps.value[0].done).toBe(true)
  })

  it('hides the widget when dismissed', () => {
    const dismissed = ref(false)
    const state = deriveOnboardingState(allFalse(), dismissed)
    expect(state.shouldShow.value).toBe(true)
    dismissed.value = true
    expect(state.shouldShow.value).toBe(false)
    expect(state.isDismissed.value).toBe(true)
  })

  it('each step has a CTA with a non-empty target', () => {
    const state = deriveOnboardingState(allFalse(), false)
    for (const step of state.steps.value) {
      expect(step.cta.label).not.toBe('')
      expect(step.cta.to).toMatch(/^\//)
    }
  })
})
