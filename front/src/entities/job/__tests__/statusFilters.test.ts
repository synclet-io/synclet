import { statusVariant } from '@shared/lib/format'
import { describe, expect, it } from 'vitest'
import { JOB_STATUS_FILTERS } from '../statusFilters'

describe('job status filters', () => {
  it('has no option of its own for the starting status', () => {
    expect(JOB_STATUS_FILTERS.map(f => f.label.toLowerCase())).not.toContain('starting')
    expect(JOB_STATUS_FILTERS.map(f => f.value as string)).not.toContain('starting')
  })

  it('keeps a starting job inside the Active filter', () => {
    expect(JOB_STATUS_FILTERS.find(f => f.value === 'active')?.statuses).toContain('starting')
  })

  it('still renders a starting job with a badge variant', () => {
    expect(statusVariant('starting')).toBe('warning')
  })
})
