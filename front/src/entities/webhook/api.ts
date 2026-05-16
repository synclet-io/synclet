import type { CreateWebhookInput, UpdateWebhookInput, Webhook, WebhookEvent } from './types'
import type { WebhookInfo as ProtoWebhookInfo } from '@/gen/synclet/publicapi/webhook/v1/webhook_pb'
import { webhookClient } from '@shared/api/services'
import { tsToDate } from '@shared/lib/formatting'

function mapEvent(value: string): WebhookEvent | null {
  switch (value) {
    case 'sync.completed':
    case 'sync.failed':
    case 'connection.paused':
    case 'schema.changed':
    case '*':
      return value
    default:
      return null
  }
}

function mapWebhook(proto: ProtoWebhookInfo): Webhook {
  return {
    id: proto.id,
    workspaceId: proto.workspaceId,
    url: proto.url,
    events: (proto.events ?? [])
      .map(mapEvent)
      .filter((e): e is WebhookEvent => e !== null),
    enabled: proto.enabled,
    createdAt: tsToDate(proto.createdAt),
    updatedAt: tsToDate(proto.updatedAt),
  }
}

export async function listWebhooks(): Promise<Webhook[]> {
  const res = await webhookClient.listWebhooks({})
  return (res.webhooks ?? []).map(mapWebhook)
}

export async function createWebhook(input: CreateWebhookInput): Promise<Webhook> {
  const res = await webhookClient.createWebhook({
    url: input.url,
    events: input.events,
    secret: input.secret ?? '',
  })
  return mapWebhook(res.webhook!)
}

export async function updateWebhook(input: UpdateWebhookInput): Promise<Webhook> {
  const res = await webhookClient.updateWebhook({
    id: input.id,
    url: input.url,
    events: input.events ?? [],
    enabled: input.enabled,
  })
  return mapWebhook(res.webhook!)
}

export async function deleteWebhook(id: string): Promise<void> {
  await webhookClient.deleteWebhook({ id })
}

export async function testWebhook(id: string): Promise<{ statusCode: number, deliveryError: string }> {
  const res = await webhookClient.testWebhook({ id })
  return {
    statusCode: res.deliveryStatusCode,
    deliveryError: res.deliveryError ?? '',
  }
}
