import type { ChannelType, NotificationChannel, NotificationCondition, NotificationRule } from './types'
import type { NotificationChannel as ProtoNotificationChannel, NotificationRule as ProtoNotificationRule } from '@/gen/synclet/publicapi/notify/v1/notify_pb'
import { notificationClient } from '@shared/api/services'
import { requireField } from '@shared/lib/errorUtils'
import { tsToDate } from '@shared/lib/formatting'
import { NotificationChannelType, NotificationCondition as ProtoNotificationCondition } from '@/gen/synclet/publicapi/notify/v1/notify_pb'

const channelTypeFromProto: Record<number, ChannelType> = {
  [NotificationChannelType.SLACK]: 'slack',
  [NotificationChannelType.EMAIL]: 'email',
  [NotificationChannelType.TELEGRAM]: 'telegram',
}

const channelTypeToProto: Record<ChannelType, NotificationChannelType> = {
  slack: NotificationChannelType.SLACK,
  email: NotificationChannelType.EMAIL,
  telegram: NotificationChannelType.TELEGRAM,
}

const conditionFromProto: Record<number, NotificationCondition> = {
  [ProtoNotificationCondition.ON_FAILURE]: 'on_failure',
  [ProtoNotificationCondition.ON_CONSECUTIVE_FAILURES]: 'on_consecutive_failures',
  [ProtoNotificationCondition.ON_ZERO_RECORDS]: 'on_zero_records',
}

const conditionToProto: Record<NotificationCondition, ProtoNotificationCondition> = {
  on_failure: ProtoNotificationCondition.ON_FAILURE,
  on_consecutive_failures: ProtoNotificationCondition.ON_CONSECUTIVE_FAILURES,
  on_zero_records: ProtoNotificationCondition.ON_ZERO_RECORDS,
}

function parseConfig(config: string | undefined | null): { config: Record<string, string>, invalid: boolean } {
  if (!config)
    return { config: {}, invalid: false }
  try {
    const parsed: unknown = JSON.parse(config)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed))
      return { config: parsed as Record<string, string>, invalid: false }
  }
  catch {}
  return { config: {}, invalid: true }
}

function mapChannel(proto: ProtoNotificationChannel): NotificationChannel {
  const { config, invalid } = parseConfig(proto.config)
  return {
    id: proto.id || '',
    workspaceId: proto.workspaceId || '',
    name: proto.name || '',
    channelType: channelTypeFromProto[proto.channelType] ?? 'slack',
    config,
    configInvalid: invalid,
    enabled: proto.enabled ?? false,
    createdAt: tsToDate(proto.createdAt),
    updatedAt: tsToDate(proto.updatedAt),
  }
}

function mapRule(proto: ProtoNotificationRule): NotificationRule {
  return {
    id: proto.id || '',
    workspaceId: proto.workspaceId || '',
    channelId: proto.channelId || '',
    connectionId: proto.connectionId || undefined,
    condition: conditionFromProto[proto.condition] ?? 'on_failure',
    conditionValue: proto.conditionValue ?? 0,
    enabled: proto.enabled ?? false,
    createdAt: tsToDate(proto.createdAt),
    updatedAt: tsToDate(proto.updatedAt),
  }
}

// Channel operations

export async function listChannels(): Promise<NotificationChannel[]> {
  const res = await notificationClient.listNotificationChannels({})
  return res.channels.map(mapChannel)
}

export async function createChannel(params: {
  name: string
  channelType: ChannelType
  config: Record<string, string>
  enabled: boolean
}): Promise<NotificationChannel> {
  const res = await notificationClient.createNotificationChannel({
    name: params.name,
    channelType: channelTypeToProto[params.channelType],
    config: JSON.stringify(params.config),
    enabled: params.enabled,
  })
  return mapChannel(requireField(res.channel, 'channel'))
}

export async function updateChannel(id: string, params: {
  name: string
  config: Record<string, string>
  enabled: boolean
}): Promise<void> {
  await notificationClient.updateNotificationChannel({
    id,
    name: params.name,
    config: JSON.stringify(params.config),
    enabled: params.enabled,
  })
}

export async function deleteChannel(id: string): Promise<void> {
  await notificationClient.deleteNotificationChannel({ id })
}

export interface TestChannelResult {
  delivered: boolean
  deliveryError: string
}

export async function testChannel(id: string): Promise<TestChannelResult> {
  const resp = await notificationClient.testNotificationChannel({ id })
  return { delivered: resp.delivered, deliveryError: resp.deliveryError }
}

// Rule operations

export async function listRules(params?: {
  channelId?: string
  connectionId?: string
}): Promise<NotificationRule[]> {
  const res = await notificationClient.listNotificationRules({
    channelId: params?.channelId,
    connectionId: params?.connectionId,
  })
  return res.rules.map(mapRule)
}

export async function createRule(params: {
  channelId: string
  connectionId?: string
  condition: NotificationCondition
  conditionValue: number
  enabled: boolean
}): Promise<NotificationRule> {
  const res = await notificationClient.createNotificationRule({
    channelId: params.channelId,
    connectionId: params.connectionId,
    condition: conditionToProto[params.condition],
    conditionValue: params.conditionValue,
    enabled: params.enabled,
  })
  return mapRule(requireField(res.rule, 'rule'))
}

export async function updateRule(id: string, params: {
  condition: NotificationCondition
  conditionValue: number
  enabled: boolean
}): Promise<void> {
  await notificationClient.updateNotificationRule({
    id,
    condition: conditionToProto[params.condition],
    conditionValue: params.conditionValue,
    enabled: params.enabled,
  })
}

export async function deleteRule(id: string): Promise<void> {
  await notificationClient.deleteNotificationRule({ id })
}
