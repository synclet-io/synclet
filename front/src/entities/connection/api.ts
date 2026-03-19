import type { ConfiguredStream, Connection, ConnectionStatus, DestinationSyncMode, NamespaceDefinition, SchemaChangePolicy, StateType, StreamStatesResult, SyncMode } from './types'
import { ConfiguredStreamSchema, connectionClient, create } from '@shared/api/services'
import { tsToDate } from '@shared/lib/formatting'
import type { Connection as ProtoConnection } from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'
import {
  CompositeKeySchema,
  ConnectionStatus as ProtoConnectionStatus,
  DestinationSyncMode as ProtoDestinationSyncMode,
  NamespaceDefinition as ProtoNamespaceDefinition,
  SchemaChangePolicy as ProtoSchemaChangePolicy,
  SyncMode as ProtoSyncMode,
  SelectedFieldSchema,
} from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'

function mapConnectionStatus(proto: ProtoConnectionStatus): ConnectionStatus {
  switch (proto) {
    case ProtoConnectionStatus.ACTIVE: return 'active'
    case ProtoConnectionStatus.INACTIVE: return 'inactive'
    case ProtoConnectionStatus.PAUSED: return 'paused'
    default: return 'active'
  }
}

function mapSchemaChangePolicy(proto: ProtoSchemaChangePolicy): SchemaChangePolicy {
  switch (proto) {
    case ProtoSchemaChangePolicy.PROPAGATE: return 'propagate'
    case ProtoSchemaChangePolicy.IGNORE: return 'ignore'
    case ProtoSchemaChangePolicy.PAUSE: return 'pause'
    default: return 'propagate'
  }
}

function toProtoSchemaChangePolicy(p: SchemaChangePolicy): ProtoSchemaChangePolicy {
  switch (p) {
    case 'propagate': return ProtoSchemaChangePolicy.PROPAGATE
    case 'ignore': return ProtoSchemaChangePolicy.IGNORE
    case 'pause': return ProtoSchemaChangePolicy.PAUSE
  }
}

function toProtoSyncMode(m: SyncMode): ProtoSyncMode {
  switch (m) {
    case 'full_refresh': return ProtoSyncMode.FULL_REFRESH
    case 'incremental': return ProtoSyncMode.INCREMENTAL
  }
}

function toProtoDestinationSyncMode(m: DestinationSyncMode): ProtoDestinationSyncMode {
  switch (m) {
    case 'overwrite': return ProtoDestinationSyncMode.OVERWRITE
    case 'append': return ProtoDestinationSyncMode.APPEND
    case 'append_dedup': return ProtoDestinationSyncMode.APPEND_DEDUP
  }
}

function mapStateType(raw: string | undefined): StateType {
  const valid: StateType[] = ['STREAM', 'GLOBAL', 'LEGACY']
  const value = raw || 'STREAM'
  return valid.includes(value as StateType) ? (value as StateType) : 'STREAM'
}

function mapNamespaceDefinition(proto: ProtoNamespaceDefinition): NamespaceDefinition {
  switch (proto) {
    case ProtoNamespaceDefinition.SOURCE: return 'source'
    case ProtoNamespaceDefinition.DESTINATION: return 'destination'
    case ProtoNamespaceDefinition.CUSTOM: return 'custom'
    default: return 'source'
  }
}

function toProtoNamespaceDefinition(n: NamespaceDefinition): ProtoNamespaceDefinition {
  switch (n) {
    case 'source': return ProtoNamespaceDefinition.SOURCE
    case 'destination': return ProtoNamespaceDefinition.DESTINATION
    case 'custom': return ProtoNamespaceDefinition.CUSTOM
  }
}

function mapConnection(proto: ProtoConnection): Connection {
  return {
    id: proto.id,
    workspaceId: proto.workspaceId,
    name: proto.name,
    status: mapConnectionStatus(proto.status),
    sourceId: proto.sourceId,
    destinationId: proto.destinationId,
    schedule: proto.schedule,
    schemaChangePolicy: mapSchemaChangePolicy(proto.schemaChangePolicy),
    createdAt: tsToDate(proto.createdAt),
    updatedAt: tsToDate(proto.updatedAt),
    maxAttempts: proto.maxAttempts,
    namespaceDefinition: mapNamespaceDefinition(proto.namespaceDefinition),
    customNamespaceFormat: proto.customNamespaceFormat || '',
    streamPrefix: proto.streamPrefix || '',
  }
}

export async function listConnections(params: { pageSize?: number, offset?: number } = {}): Promise<{ items: Connection[], total: number }> {
  const res = await connectionClient.listConnections({
    pageSize: params.pageSize ?? 0,
    offset: params.offset ?? 0,
  })
  return {
    items: (res.connections || []).map(mapConnection),
    total: Number(res.total),
  }
}

export async function getConnection(id: string): Promise<Connection | undefined> {
  const res = await connectionClient.getConnection({ id })
  return res.connection ? mapConnection(res.connection) : undefined
}

export async function createConnection(params: {
  name: string
  sourceId: string
  destinationId: string
  schedule?: string
  schemaChangePolicy: SchemaChangePolicy
  maxAttempts: number
  namespaceDefinition?: NamespaceDefinition
  customNamespaceFormat?: string
  streamPrefix?: string
}): Promise<Connection | undefined> {
  const res = await connectionClient.createConnection({
    name: params.name,
    sourceId: params.sourceId,
    destinationId: params.destinationId,
    schedule: params.schedule,
    schemaChangePolicy: toProtoSchemaChangePolicy(params.schemaChangePolicy),
    maxAttempts: params.maxAttempts,
    namespaceDefinition: params.namespaceDefinition ? toProtoNamespaceDefinition(params.namespaceDefinition) : undefined,
    customNamespaceFormat: params.customNamespaceFormat || '',
    streamPrefix: params.streamPrefix || '',
  })
  return res.connection ? mapConnection(res.connection) : undefined
}

export async function deleteConnection(id: string): Promise<void> {
  await connectionClient.deleteConnection({ id })
}

export async function enableConnection(id: string): Promise<Connection | undefined> {
  const res = await connectionClient.enableConnection({ id })
  return res.connection ? mapConnection(res.connection) : undefined
}

export async function disableConnection(id: string): Promise<Connection | undefined> {
  const res = await connectionClient.disableConnection({ id })
  return res.connection ? mapConnection(res.connection) : undefined
}

export async function getDiscoveredCatalog(connectionId: string): Promise<Record<string, unknown> | undefined> {
  const res = await connectionClient.getDiscoveredCatalog({ connectionId })
  return res.catalog ?? undefined
}

export async function discoverSchema(connectionId: string): Promise<{ taskId: string }> {
  const res = await connectionClient.discoverSchema({ connectionId })
  return { taskId: res.taskId }
}

export async function configureStreams(connectionId: string, streams: ConfiguredStream[]): Promise<void> {
  const protoStreams = streams.map(s =>
    create(ConfiguredStreamSchema, {
      streamName: s.streamName,
      namespace: s.namespace,
      syncMode: toProtoSyncMode(s.syncMode),
      destinationSyncMode: toProtoDestinationSyncMode(s.destinationSyncMode),
      cursorField: s.cursorField,
      primaryKey: s.primaryKey.map(pk => create(CompositeKeySchema, { fieldPath: pk })),
      selectedFields: s.selectedFields.map(sf => create(SelectedFieldSchema, { fieldPath: sf.fieldPath })),
    }),
  )
  await connectionClient.configureStreams({ connectionId, streams: protoStreams })
}

export async function updateConnection(params: {
  id: string
  name?: string
  schedule?: string
  schemaChangePolicy?: SchemaChangePolicy
  maxAttempts?: number
  namespaceDefinition?: NamespaceDefinition
  customNamespaceFormat?: string
  streamPrefix?: string
}): Promise<Connection | undefined> {
  const res = await connectionClient.updateConnection({
    id: params.id,
    name: params.name,
    schedule: params.schedule,
    schemaChangePolicy: params.schemaChangePolicy ? toProtoSchemaChangePolicy(params.schemaChangePolicy) : undefined,
    maxAttempts: params.maxAttempts,
    namespaceDefinition: params.namespaceDefinition ? toProtoNamespaceDefinition(params.namespaceDefinition) : undefined,
    customNamespaceFormat: params.customNamespaceFormat,
    streamPrefix: params.streamPrefix,
  })
  return res.connection ? mapConnection(res.connection) : undefined
}

export async function resetStreamState(connectionId: string, streamNamespace: string, streamName: string): Promise<void> {
  await connectionClient.resetStreamState({ connectionId, streamNamespace, streamName })
}

export async function getConfiguredCatalog(connectionId: string): Promise<Record<string, unknown> | undefined> {
  const res = await connectionClient.getConfiguredCatalog({ connectionId })
  return res.catalog ?? undefined
}

export async function resetConnectionState(connectionId: string): Promise<void> {
  await connectionClient.resetConnectionState({ connectionId })
}

export async function listStreamStates(connectionId: string): Promise<StreamStatesResult> {
  const res = await connectionClient.listStreamStates({ connectionId })
  return {
    stateType: mapStateType(res.stateType),
    states: (res.states || []).map(s => ({
      streamName: s.streamName,
      streamNamespace: s.streamNamespace,
      stateData: s.stateData,
      updatedAt: tsToDate(s.updatedAt),
    })),
  }
}

export async function updateStreamState(connectionId: string, streamName: string, streamNamespace: string, stateData: string): Promise<void> {
  await connectionClient.updateStreamState({ connectionId, streamName, streamNamespace, stateData })
}
