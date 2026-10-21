export type ConnectionStatus = 'active' | 'inactive' | 'paused' | 'unknown'
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
  pendingSchemaChanges: number
  sourceName: string
  destinationName: string
  sourceConnectorName: string
  destinationConnectorName: string
  tags: string[]
  /** Workflows the connection is a step of; such a connection has no schedule of its own. */
  workflows: ConnectionWorkflow[]
}

export interface ConnectionWorkflow {
  id: string
  name: string
}

export interface ConnectionTag {
  tag: string
  connectionCount: number
}

export interface ConfiguredStream {
  streamName: string
  namespace: string
  syncMode: SyncMode
  destinationSyncMode: DestinationSyncMode
  cursorField: string[]
  primaryKey: string[][]
  selectedFields: SelectedField[]
  mappers?: Record<string, unknown>[]
}

export interface MapperIssue {
  step: number
  field: string
  message: string
}

// What ValidateMappers checks steps against: a saved connection, or the source
// and name of a connection not created yet.
export type MapperTarget = { connectionId: string } | { sourceId: string, connectionName: string }

export interface ValidateMappersParams {
  namespace: string
  streamName: string
  mappers: Record<string, unknown>[]
  selectedFields: SelectedField[]
  primaryKey: string[][]
  destinationSyncMode: DestinationSyncMode
  sampleRecord?: Record<string, unknown>
}

export interface ValidateMappersResult {
  errors: MapperIssue[]
  warnings: MapperIssue[]
  outputSchema: Record<string, unknown> | undefined
  outputPrimaryKey: string[][]
  sampleOutput: Record<string, unknown> | undefined
  sampleKept: boolean
  sampleError: string
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

export type SchemaChangeType = 'added' | 'removed' | 'modified' | 'unknown'

export interface SchemaChange {
  type: SchemaChangeType
  streamName: string
  namespace: string
  columnName: string
  oldType?: string
  newType?: string
}
