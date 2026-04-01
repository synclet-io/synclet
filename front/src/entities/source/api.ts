import type { JsonObject } from '@bufbuild/protobuf'
import type { Source, SourceCatalog } from './types'
import type { Source as ProtoSource } from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'
import { sourceClient } from '@shared/api/services'
import { tsToDate } from '@shared/lib/formatting'

function mapSource(proto: ProtoSource): Source {
  return {
    id: proto.id,
    workspaceId: proto.workspaceId,
    name: proto.name,
    managedConnectorId: proto.managedConnectorId,
    config: proto.config ?? {},
    runtimeConfig: proto.runtimeConfig ?? null,
    createdAt: tsToDate(proto.createdAt),
    updatedAt: tsToDate(proto.updatedAt),
  }
}

export async function listSources(params: { pageSize?: number, offset?: number } = {}): Promise<{ items: Source[], total: number }> {
  const res = await sourceClient.listSources({
    pageSize: params.pageSize ?? 0,
    offset: params.offset ?? 0,
  })
  return {
    items: (res.sources || []).map(mapSource),
    total: Number(res.total),
  }
}

export async function getSource(id: string): Promise<Source | undefined> {
  const res = await sourceClient.getSource({ id })
  return res.source ? mapSource(res.source) : undefined
}

export async function createSource(params: { name: string, managedConnectorId: string, config: Record<string, unknown> }): Promise<{ source: Source | undefined, discoverTaskId?: string }> {
  const res = await sourceClient.createSource({ name: params.name, managedConnectorId: params.managedConnectorId, config: params.config as JsonObject })
  return {
    source: res.source ? mapSource(res.source) : undefined,
    discoverTaskId: res.discoverTaskId || undefined,
  }
}

export async function updateSource(params: { id: string, name?: string, config?: Record<string, unknown>, runtimeConfig?: string | null }): Promise<{ source: Source | undefined, discoverTaskId?: string }> {
  const req: any = { id: params.id }
  if (params.name !== undefined)
    req.name = params.name
  if (params.config !== undefined)
    req.config = params.config as JsonObject
  if (params.runtimeConfig !== undefined)
    req.runtimeConfig = params.runtimeConfig
  const res = await sourceClient.updateSource(req)
  return {
    source: res.source ? mapSource(res.source) : undefined,
    discoverTaskId: res.discoverTaskId || undefined,
  }
}

export async function deleteSource(id: string): Promise<void> {
  await sourceClient.deleteSource({ id })
}

export async function testConnection(params: { id: string } | { managedConnectorId: string, config: Record<string, unknown> }): Promise<{ taskId: string }> {
  const res = await sourceClient.testSourceConnection(params as Parameters<typeof sourceClient.testSourceConnection>[0])
  return { taskId: res.taskId }
}

export async function getSourceCatalog(sourceId: string): Promise<SourceCatalog | null> {
  try {
    const res = await sourceClient.getSourceCatalog({ sourceId })
    return {
      catalog: res.catalog ? (res.catalog as Record<string, unknown>) : {},
      version: res.version,
      discoveredAt: tsToDate(res.discoveredAt),
    }
  }
  catch {
    return null
  }
}

export async function discoverSourceSchema(sourceId: string): Promise<{ taskId: string }> {
  const res = await sourceClient.discoverSourceSchema({ sourceId })
  return { taskId: res.taskId }
}
