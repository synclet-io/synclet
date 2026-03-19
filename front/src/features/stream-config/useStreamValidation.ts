import type { DestinationSyncMode, SelectedField, SyncMode } from '@entities/connection'

export interface StreamValidationError {
  streamKey: string
  type: 'missing_cursor' | 'missing_pk' | 'missing_fields' | 'incompatible_sync_mode'
  message: string
}

export interface ValidatableStream {
  name: string
  namespace: string
  enabled: boolean
  syncMode: SyncMode
  destinationSyncMode: DestinationSyncMode
  cursorField: string[]
  primaryKey: string[][]
  selectedFields: SelectedField[]
  sourceDefinedCursor: boolean
  sourceDefinedPrimaryKey: string[][]
}

function streamKey(s: ValidatableStream): string {
  return s.namespace ? `${s.namespace}.${s.name}` : s.name
}

export function useStreamValidation() {
  function validate(streams: ValidatableStream[]): StreamValidationError[] {
    const errors: StreamValidationError[] = []

    for (const s of streams) {
      if (!s.enabled)
        continue

      const key = streamKey(s)

      // D-16: each enabled stream needs at least one field selected
      if (s.selectedFields.length === 0) {
        errors.push({
          streamKey: key,
          type: 'missing_fields',
          message: 'Select at least one field for replication',
        })
      }

      // D-07: incremental without source cursor needs user cursor
      if (
        s.syncMode === 'incremental'
        && !s.sourceDefinedCursor
        && s.cursorField.length === 0
      ) {
        errors.push({
          streamKey: key,
          type: 'missing_cursor',
          message: 'Cursor field is required for Incremental sync',
        })
      }

      // D-08: append_dedup without source PK needs user PK
      if (
        s.destinationSyncMode === 'append_dedup'
        && s.sourceDefinedPrimaryKey.length === 0
        && s.primaryKey.length === 0
      ) {
        errors.push({
          streamKey: key,
          type: 'missing_pk',
          message: 'Primary key is required for Append + Dedup mode',
        })
      }
    }

    return errors
  }

  return { validate }
}
