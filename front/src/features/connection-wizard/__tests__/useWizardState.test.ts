import { describe, expect, it } from 'vitest'
import { useWizardState } from '../useWizardState'

describe('useWizardState', () => {
  it('has no unsaved changes before a source or destination is picked', () => {
    const w = useWizardState()
    expect(w.hasUnsavedChanges.value).toBe(false)
  })

  it('reports unsaved changes once a source is picked', () => {
    const w = useWizardState()
    w.state.value.sourceId = 'src-1'
    expect(w.hasUnsavedChanges.value).toBe(true)
  })

  it('clears unsaved changes after the connection is committed', () => {
    const w = useWizardState()
    w.state.value.sourceId = 'src-1'
    w.state.value.destinationId = 'dst-1'
    w.markCommitted('conn-1')
    expect(w.committedConnectionId.value).toBe('conn-1')
    expect(w.hasUnsavedChanges.value).toBe(false)
    expect(w.hasAnyState.value).toBe(true)
  })
})
