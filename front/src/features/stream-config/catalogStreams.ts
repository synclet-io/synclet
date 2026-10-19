import type { DestinationSyncMode, SelectedField, SyncMode } from '@entities/connection'

export interface StreamRow {
  name: string
  namespace: string
  syncMode: SyncMode
  destinationSyncMode: DestinationSyncMode
  enabled: boolean
  supportedSyncModes: string[]
  cursorField: string[]
  primaryKey: string[][]
  selectedFields: SelectedField[]
  jsonSchema: Record<string, any>
  sourceDefinedCursor: boolean
  defaultCursorField: string[]
  sourceDefinedPrimaryKey: string[][]
  mappers: Record<string, unknown>[]
  transformErrors?: number
}

export interface DiscoveredCatalog {
  streams: Array<{
    stream: {
      name: string
      namespace?: string
      json_schema?: Record<string, unknown>
      jsonSchema?: Record<string, unknown>
      supported_sync_modes?: string[]
      source_defined_cursor?: boolean
      default_cursor_field?: string[]
      source_defined_primary_key?: string[][]
    }
    sync_mode?: string
    destination_sync_mode?: string
  }>
}

type FieldPathInput = string[] | { field_path?: string[], fieldPath?: string[] }

export interface ConfiguredCatalogInput {
  streams?: Array<{
    stream?: { name?: string, namespace?: string }
    name?: string
    namespace?: string
    sync_mode?: string
    syncMode?: string
    destination_sync_mode?: string
    destinationSyncMode?: string
    cursor_field?: string[]
    cursorField?: string[]
    primary_key?: FieldPathInput[]
    primaryKey?: FieldPathInput[]
    selected_fields?: Array<{ field_path?: string[], fieldPath?: string[] }>
    selectedFields?: Array<{ field_path?: string[], fieldPath?: string[] }>
    mappers?: Record<string, unknown>[]
  }>
}

export interface StreamMergeResult {
  streams: StreamRow[]
  added: string[]
  removed: string[]
}

export function streamKey(s: { name: string, namespace: string }): string {
  return s.namespace ? `${s.namespace}.${s.name}` : s.name
}

function fieldPath(pk: FieldPathInput): string[] {
  if (Array.isArray(pk))
    return pk
  return pk?.field_path || pk?.fieldPath || []
}

export function parseStreams(cat: DiscoveredCatalog, configuredCat?: ConfiguredCatalogInput): StreamRow[] {
  if (!cat?.streams)
    return []

  type ConfiguredEntry = NonNullable<ConfiguredCatalogInput['streams']>[number]
  const configuredMap = new Map<string, ConfiguredEntry>()
  for (const cs of configuredCat?.streams ?? []) {
    const stream = cs.stream || cs
    configuredMap.set(streamKey({ name: stream.name || '', namespace: stream.namespace || '' }), cs)
  }
  const hasConfigured = configuredMap.size > 0

  return cat.streams.map((s) => {
    const stream = s.stream || s
    const name = stream.name || ''
    const namespace = stream.namespace || ''
    const discovered = {
      name,
      namespace,
      supportedSyncModes: stream.supported_sync_modes || ['full_refresh', 'incremental'],
      jsonSchema: stream.json_schema || stream.jsonSchema || {},
      sourceDefinedCursor: !!stream.source_defined_cursor,
      defaultCursorField: stream.default_cursor_field || [],
      sourceDefinedPrimaryKey: stream.source_defined_primary_key || [],
    }

    const configured = configuredMap.get(streamKey(discovered))
    if (configured) {
      return {
        ...discovered,
        syncMode: (configured.sync_mode || configured.syncMode || 'full_refresh') as SyncMode,
        destinationSyncMode: (configured.destination_sync_mode || configured.destinationSyncMode || 'overwrite') as DestinationSyncMode,
        enabled: true,
        cursorField: configured.cursor_field || configured.cursorField || [],
        primaryKey: (configured.primary_key || configured.primaryKey || []).map(fieldPath),
        selectedFields: (configured.selected_fields || configured.selectedFields || []).map(sf => ({ fieldPath: fieldPath(sf) })),
        mappers: configured.mappers || [],
      } satisfies StreamRow
    }

    return {
      ...discovered,
      syncMode: (s.sync_mode || 'full_refresh') as SyncMode,
      destinationSyncMode: (s.destination_sync_mode || 'overwrite') as DestinationSyncMode,
      enabled: !hasConfigured,
      cursorField: [] as string[],
      primaryKey: [] as string[][],
      selectedFields: [] as SelectedField[],
      mappers: [] as Record<string, unknown>[],
    } satisfies StreamRow
  })
}

// Rows present in `current` keep their user configuration and take the schema metadata from
// `discovered`; streams new to the catalog arrive disabled unless nothing was listed before.
export function mergeDiscoveredStreams(current: StreamRow[], discovered: StreamRow[]): StreamMergeResult {
  const currentByKey = new Map(current.map(s => [streamKey(s), s]))
  const discoveredKeys = new Set(discovered.map(streamKey))
  const added: string[] = []

  const streams = discovered.map((d) => {
    const key = streamKey(d)
    const existing = currentByKey.get(key)
    if (!existing) {
      added.push(key)
      return { ...d, enabled: current.length === 0 }
    }
    return {
      ...d,
      enabled: existing.enabled,
      syncMode: existing.syncMode,
      destinationSyncMode: existing.destinationSyncMode,
      cursorField: existing.cursorField,
      primaryKey: existing.primaryKey,
      selectedFields: existing.selectedFields,
      mappers: existing.mappers,
    }
  })

  const removed = current.map(streamKey).filter(key => !discoveredKeys.has(key))
  return { streams, added, removed }
}
