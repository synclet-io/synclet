import type { ValidatableStream } from '../useStreamValidation'
import { describe, expect, it } from 'vitest'
import { useStreamValidation } from '../useStreamValidation'

function stream(partial: Partial<ValidatableStream> = {}): ValidatableStream {
  return {
    name: 'orders',
    namespace: 'public',
    enabled: true,
    syncMode: 'full_refresh',
    destinationSyncMode: 'overwrite',
    cursorField: [],
    primaryKey: [],
    selectedFields: [{ fieldPath: ['id'] }],
    sourceDefinedCursor: false,
    sourceDefinedPrimaryKey: [],
    ...partial,
  }
}

describe('useStreamValidation', () => {
  it('blocks a stream whose Transform steps have errors', () => {
    const { validate } = useStreamValidation()
    expect(validate([stream({ transformErrors: 2 })])).toEqual([
      { streamKey: 'public.orders', type: 'transform', message: 'Fix the errors in the Transform steps' },
    ])
    expect(validate([stream({ transformErrors: 0 }), stream({ name: 'off', enabled: false, transformErrors: 3 })])).toEqual([])
  })
})
