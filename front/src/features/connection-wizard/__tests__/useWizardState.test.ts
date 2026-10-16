import type { Destination } from '@entities/destination'
import type { Source } from '@entities/source'
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

  describe('preselect', () => {
    const source = { id: 'src-1', name: 'Postgres' } as Source
    const destination = { id: 'dst-1', name: 'BigQuery' } as Destination

    it('skips to the destination step when only the source is given', () => {
      const w = useWizardState()
      w.preselect({ source })
      expect(w.state.value.sourceId).toBe('src-1')
      expect(w.state.value.source).toStrictEqual(source)
      expect(w.currentStep.value).toBe(2)
    })

    it('skips to settings when both are given', () => {
      const w = useWizardState()
      w.preselect({ source, destination })
      expect(w.state.value.destinationId).toBe('dst-1')
      expect(w.currentStep.value).toBe(3)
    })

    it('keeps the source step when only the destination is given', () => {
      const w = useWizardState()
      w.preselect({ destination })
      expect(w.state.value.destinationId).toBe('dst-1')
      expect(w.currentStep.value).toBe(1)
    })

    it('counts only changes made after the preselection as unsaved', () => {
      const w = useWizardState()
      w.preselect({ source, destination })
      expect(w.hasUnsavedChanges.value).toBe(false)
      w.state.value.name = 'PG to BQ'
      expect(w.hasUnsavedChanges.value).toBe(true)
    })

    it('leaves the wizard untouched when nothing resolved', () => {
      const w = useWizardState()
      w.preselect({})
      expect(w.hasAnyState.value).toBe(false)
      expect(w.currentStep.value).toBe(1)
    })
  })
})
