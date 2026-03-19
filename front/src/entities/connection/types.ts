export type ConnectionStatus = 'active' | 'inactive' | 'paused'
export type SchemaChangePolicy = 'propagate' | 'ignore' | 'pause'
export type SyncMode = 'full_refresh' | 'incremental'
export type DestinationSyncMode = 'overwrite' | 'append' | 'append_dedup'
export type NamespaceDefinition = 'source' | 'destination' | 'custom'

export interface SelectedField {
  fieldPath: string[]
}

export interface Connection {
  id: string
  workspaceId: string
  name: string
  status: ConnectionStatus
  sourceId: string
  destinationId: string
  schedule: string
  schemaChangePolicy: SchemaChangePolicy
  createdAt: Date | undefined
  updatedAt: Date | undefined
  maxAttempts: number
  namespaceDefinition: NamespaceDefinition
  customNamespaceFormat: string
  streamPrefix: string
}

export interface ConfiguredStream {
  streamName: string
  namespace: string
  syncMode: SyncMode
  destinationSyncMode: DestinationSyncMode
  cursorField: string[]
  primaryKey: string[][]
  selectedFields: SelectedField[]
}

export interface StreamState {
  streamName: string
  streamNamespace: string
  stateData: string
  updatedAt: Date | undefined
}

export type StateType = 'STREAM' | 'GLOBAL' | 'LEGACY'

export interface StreamStatesResult {
  stateType: StateType
  states: StreamState[]
}

export interface SyncModePair {
  label: string
  syncMode: SyncMode
  destinationSyncMode: DestinationSyncMode
}

export const SYNC_MODE_PAIRS: SyncModePair[] = [
  { label: 'Full Refresh | Overwrite', syncMode: 'full_refresh', destinationSyncMode: 'overwrite' },
  { label: 'Full Refresh | Append', syncMode: 'full_refresh', destinationSyncMode: 'append' },
  { label: 'Incremental | Append', syncMode: 'incremental', destinationSyncMode: 'append' },
  { label: 'Incremental | Append+Dedup', syncMode: 'incremental', destinationSyncMode: 'append_dedup' },
]

export function getAvailableSyncModePairs(supportedSyncModes: string[]): SyncModePair[] {
  return SYNC_MODE_PAIRS.filter(p => supportedSyncModes.includes(p.syncMode))
}
