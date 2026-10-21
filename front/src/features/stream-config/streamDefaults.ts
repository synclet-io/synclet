import type { DestinationSyncMode, SelectedField, SyncMode } from '@entities/connection'
import type { StreamRow } from './catalogStreams'
import { getLeafFields, parseJsonSchema } from './schemaParser'

export function allFieldSelection(jsonSchema: Record<string, any> | undefined): SelectedField[] {
  if (!jsonSchema)
    return []
  return getLeafFields(parseJsonSchema(jsonSchema)).map(f => ({ fieldPath: f.path }))
}

function bestSyncMode(s: StreamRow, hasPrimaryKey: boolean, hasCursor: boolean): { syncMode: SyncMode, destinationSyncMode: DestinationSyncMode } {
  const incremental = s.supportedSyncModes.includes('incremental')
  const fullRefresh = s.supportedSyncModes.includes('full_refresh')
  if (incremental && hasCursor)
    return { syncMode: 'incremental', destinationSyncMode: hasPrimaryKey ? 'append_dedup' : 'append' }
  if (fullRefresh || !incremental)
    return { syncMode: 'full_refresh', destinationSyncMode: 'overwrite' }
  return { syncMode: 'incremental', destinationSyncMode: 'append' }
}

/**
 * Returns the stream turned on with every field selected, the primary key and cursor the
 * source declares, and the strongest sync mode those allow.
 */
export function enableStream(s: StreamRow): StreamRow {
  const primaryKey = s.sourceDefinedPrimaryKey.map(p => [...p])
  const cursorField = [...s.defaultCursorField]
  return {
    ...s,
    enabled: true,
    configured: true,
    selectedFields: allFieldSelection(s.jsonSchema),
    primaryKey,
    cursorField,
    ...bestSyncMode(s, primaryKey.length > 0, s.sourceDefinedCursor || cursorField.length > 0),
  }
}
