import type { JsonObject } from '@bufbuild/protobuf'
import type { Destination } from './types'
import type { Destination as ProtoDestination } from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'
import { destinationClient } from '@shared/api/services'
import { tsToDate } from '@shared/lib/formatting'

function mapDestination(proto: ProtoDestination): Destination {
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

export async function listDestinations(params: { pageSize?: number, offset?: number } = {}): Promise<{ items: Destination[], total: number }> {
  const res = await destinationClient.listDestinations({
    pageSize: params.pageSize ?? 0,
    offset: params.offset ?? 0,
  })
  return {
    items: (res.destinations || []).map(mapDestination),
    total: Number(res.total),
  }
}

export async function getDestination(id: string): Promise<Destination | undefined> {
  const res = await destinationClient.getDestination({ id })
  return res.destination ? mapDestination(res.destination) : undefined
}

export async function createDestination(params: { name: string, managedConnectorId: string, config: Record<string, unknown> }): Promise<Destination | undefined> {
  const res = await destinationClient.createDestination({ name: params.name, managedConnectorId: params.managedConnectorId, config: params.config as JsonObject })
  return res.destination ? mapDestination(res.destination) : undefined
}

export async function updateDestination(params: { id: string, name?: string, config?: Record<string, unknown>, runtimeConfig?: string | null }): Promise<Destination | undefined> {
  const req: any = { id: params.id }
  if (params.name !== undefined)
    req.name = params.name
  if (params.config !== undefined)
    req.config = params.config as JsonObject
  if (params.runtimeConfig !== undefined)
    req.runtimeConfig = params.runtimeConfig
  const res = await destinationClient.updateDestination(req)
  return res.destination ? mapDestination(res.destination) : undefined
}

export async function deleteDestination(id: string): Promise<void> {
  await destinationClient.deleteDestination({ id })
}

export async function testConnection(params: { id: string } | { managedConnectorId: string, config: Record<string, unknown> }): Promise<{ taskId: string }> {
  const res = await destinationClient.testDestinationConnection(params as Parameters<typeof destinationClient.testDestinationConnection>[0])
  return { taskId: res.taskId }
}
