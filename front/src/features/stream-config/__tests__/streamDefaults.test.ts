import type { StreamRow } from '../catalogStreams'
import { describe, expect, it } from 'vitest'
import { enableStream } from '../streamDefaults'

function row(overrides: Partial<StreamRow> = {}): StreamRow {
  return {
    name: 'orders',
    namespace: '',
    syncMode: 'full_refresh',
    destinationSyncMode: 'overwrite',
    enabled: false,
    supportedSyncModes: ['full_refresh', 'incremental'],
    cursorField: [],
    primaryKey: [],
    selectedFields: [],
    jsonSchema: {
      properties: {
        id: { type: 'integer' },
        updated_at: { type: 'string' },
        customer: { type: 'object', properties: { email: { type: 'string' } } },
      },
    },
    sourceDefinedCursor: false,
    defaultCursorField: [],
    sourceDefinedPrimaryKey: [],
    mappers: [],
    ...overrides,
  }
}

describe('enableStream', () => {
  it('selects every leaf field and enables the stream', () => {
    const s = enableStream(row())
    expect(s.enabled).toBe(true)
    expect(s.selectedFields.map(f => f.fieldPath)).toEqual([['id'], ['updated_at'], ['customer', 'email']])
  })

  it('keeps the mappers of the stream', () => {
    const mappers = [{ type: 'hash' }]
    expect(enableStream(row({ mappers })).mappers).toBe(mappers)
  })

  it.each([
    {
      case: 'primary key and source-defined cursor',
      input: { sourceDefinedPrimaryKey: [['id']], sourceDefinedCursor: true, defaultCursorField: ['updated_at'] },
      want: { syncMode: 'incremental', destinationSyncMode: 'append_dedup', primaryKey: [['id']], cursorField: ['updated_at'] },
    },
    {
      case: 'primary key and default cursor field',
      input: { sourceDefinedPrimaryKey: [['id']], defaultCursorField: ['updated_at'] },
      want: { syncMode: 'incremental', destinationSyncMode: 'append_dedup', primaryKey: [['id']], cursorField: ['updated_at'] },
    },
    {
      case: 'cursor only',
      input: { defaultCursorField: ['updated_at'] },
      want: { syncMode: 'incremental', destinationSyncMode: 'append', primaryKey: [], cursorField: ['updated_at'] },
    },
    {
      case: 'source-defined cursor without a field',
      input: { sourceDefinedCursor: true },
      want: { syncMode: 'incremental', destinationSyncMode: 'append', primaryKey: [], cursorField: [] },
    },
    {
      case: 'primary key only',
      input: { sourceDefinedPrimaryKey: [['id']] },
      want: { syncMode: 'full_refresh', destinationSyncMode: 'overwrite', primaryKey: [['id']], cursorField: [] },
    },
    {
      case: 'nothing known',
      input: {},
      want: { syncMode: 'full_refresh', destinationSyncMode: 'overwrite', primaryKey: [], cursorField: [] },
    },
    {
      case: 'cursor known but incremental unsupported',
      input: { sourceDefinedPrimaryKey: [['id']], defaultCursorField: ['updated_at'], supportedSyncModes: ['full_refresh'] },
      want: { syncMode: 'full_refresh', destinationSyncMode: 'overwrite', primaryKey: [['id']], cursorField: ['updated_at'] },
    },
    {
      case: 'incremental only without a cursor',
      input: { supportedSyncModes: ['incremental'] },
      want: { syncMode: 'incremental', destinationSyncMode: 'append', primaryKey: [], cursorField: [] },
    },
  ] as const)('$case', ({ input, want }) => {
    const s = enableStream(row(input as Partial<StreamRow>))
    expect({
      syncMode: s.syncMode,
      destinationSyncMode: s.destinationSyncMode,
      primaryKey: s.primaryKey,
      cursorField: s.cursorField,
    }).toEqual(want)
  })
})
