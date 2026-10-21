import type { MapperTarget } from '@entities/connection'
import type { StreamRow } from '../catalogStreams'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import StreamExpandedRow from '../StreamExpandedRow.vue'

vi.mock('@entities/connection', () => ({ validateMappers: vi.fn() }))

const stream: StreamRow = {
  name: 'orders',
  namespace: '',
  syncMode: 'full_refresh',
  destinationSyncMode: 'overwrite',
  enabled: true,
  supportedSyncModes: ['full_refresh'],
  cursorField: [],
  primaryKey: [],
  selectedFields: [],
  jsonSchema: { properties: { id: { type: 'integer' } } },
  sourceDefinedCursor: false,
  defaultCursorField: [],
  sourceDefinedPrimaryKey: [],
  mappers: [],
}

function tabNames(connectionId: string, mapperTarget?: MapperTarget): string[] {
  const wrapper = mount(StreamExpandedRow, {
    props: { stream, schema: stream.jsonSchema, validationErrors: [], connectionId, mapperTarget, stateData: undefined, stateUpdatedAt: undefined },
  })
  return wrapper.findAll('[role="tab"]').map(t => t.text())
}

describe('streamExpandedRow', () => {
  it('offers the Transform tab when there is something to validate the steps against', () => {
    expect(tabNames('conn-1', { connectionId: 'conn-1' })).toEqual(['Fields', 'Transform', 'State'])
    expect(tabNames('', { sourceId: 'src-1', connectionName: 'Orders' })).toEqual(['Fields', 'Transform', 'State'])
    expect(tabNames('')).toEqual(['Fields', 'State'])
  })
})
