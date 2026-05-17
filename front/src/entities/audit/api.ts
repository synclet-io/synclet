import type { AuditAction, AuditActorType, AuditEvent, AuditFieldChange, AuditResourceType, ListAuditEventsParams } from './types'
import type { AuditEvent as ProtoAuditEvent } from '@/gen/synclet/publicapi/audit/v1/audit_pb'
import { timestampFromDate } from '@bufbuild/protobuf/wkt'
import { auditClient } from '@shared/api/services'
import { tsToDate } from '@shared/lib/formatting'
import { AuditAction as ProtoAuditAction, AuditActorType as ProtoAuditActorType, AuditResourceType as ProtoAuditResourceType } from '@/gen/synclet/publicapi/audit/v1/audit_pb'

function mapAction(proto: ProtoAuditAction): AuditAction {
  switch (proto) {
    case ProtoAuditAction.CREATE: return 'create'
    case ProtoAuditAction.UPDATE: return 'update'
    case ProtoAuditAction.DELETE: return 'delete'
    case ProtoAuditAction.ENABLE: return 'enable'
    case ProtoAuditAction.DISABLE: return 'disable'
    case ProtoAuditAction.TEST: return 'test'
    case ProtoAuditAction.SYNC: return 'sync'
    case ProtoAuditAction.LOGIN: return 'login'
    case ProtoAuditAction.LOGOUT: return 'logout'
    default: return 'update'
  }
}

function toProtoAction(a: AuditAction): ProtoAuditAction {
  switch (a) {
    case 'create': return ProtoAuditAction.CREATE
    case 'update': return ProtoAuditAction.UPDATE
    case 'delete': return ProtoAuditAction.DELETE
    case 'enable': return ProtoAuditAction.ENABLE
    case 'disable': return ProtoAuditAction.DISABLE
    case 'test': return ProtoAuditAction.TEST
    case 'sync': return ProtoAuditAction.SYNC
    case 'login': return ProtoAuditAction.LOGIN
    case 'logout': return ProtoAuditAction.LOGOUT
  }
}

function mapResourceType(proto: ProtoAuditResourceType): AuditResourceType {
  switch (proto) {
    case ProtoAuditResourceType.SOURCE: return 'source'
    case ProtoAuditResourceType.DESTINATION: return 'destination'
    case ProtoAuditResourceType.CONNECTION: return 'connection'
    case ProtoAuditResourceType.CONNECTOR: return 'connector'
    case ProtoAuditResourceType.REPOSITORY: return 'repository'
    case ProtoAuditResourceType.NOTIFICATION_CHANNEL: return 'notification_channel'
    case ProtoAuditResourceType.NOTIFICATION_RULE: return 'notification_rule'
    case ProtoAuditResourceType.WEBHOOK: return 'webhook'
    case ProtoAuditResourceType.WORKSPACE: return 'workspace'
    case ProtoAuditResourceType.WORKSPACE_MEMBER: return 'workspace_member'
    case ProtoAuditResourceType.WORKSPACE_INVITE: return 'workspace_invite'
    case ProtoAuditResourceType.API_KEY: return 'api_key'
    case ProtoAuditResourceType.USER: return 'user'
    default: return 'connection'
  }
}

function toProtoResourceType(t: AuditResourceType): ProtoAuditResourceType {
  switch (t) {
    case 'source': return ProtoAuditResourceType.SOURCE
    case 'destination': return ProtoAuditResourceType.DESTINATION
    case 'connection': return ProtoAuditResourceType.CONNECTION
    case 'connector': return ProtoAuditResourceType.CONNECTOR
    case 'repository': return ProtoAuditResourceType.REPOSITORY
    case 'notification_channel': return ProtoAuditResourceType.NOTIFICATION_CHANNEL
    case 'notification_rule': return ProtoAuditResourceType.NOTIFICATION_RULE
    case 'webhook': return ProtoAuditResourceType.WEBHOOK
    case 'workspace': return ProtoAuditResourceType.WORKSPACE
    case 'workspace_member': return ProtoAuditResourceType.WORKSPACE_MEMBER
    case 'workspace_invite': return ProtoAuditResourceType.WORKSPACE_INVITE
    case 'api_key': return ProtoAuditResourceType.API_KEY
    case 'user': return ProtoAuditResourceType.USER
  }
}

function mapActorType(proto: ProtoAuditActorType): AuditActorType {
  switch (proto) {
    case ProtoAuditActorType.USER: return 'user'
    case ProtoAuditActorType.API_KEY: return 'api_key'
    case ProtoAuditActorType.SYSTEM: return 'system'
    default: return 'user'
  }
}

function parseChanges(diffJson: string): AuditFieldChange[] {
  if (!diffJson)
    return []
  try {
    const parsed = JSON.parse(diffJson)
    if (Array.isArray(parsed))
      return parsed as AuditFieldChange[]
  }
  catch {
    // Corrupt diff — fall through and return empty list.
  }
  return []
}

function mapEvent(proto: ProtoAuditEvent): AuditEvent {
  return {
    id: proto.id,
    workspaceId: proto.workspaceId,
    actorType: mapActorType(proto.actorType),
    actorId: proto.actorId,
    actorLabel: proto.actorLabel,
    action: mapAction(proto.action),
    resourceType: mapResourceType(proto.resourceType),
    resourceId: proto.resourceId,
    resourceLabel: proto.resourceLabel,
    changes: parseChanges(proto.diffJson),
    diffTruncated: proto.diffTruncated,
    createdAt: tsToDate(proto.createdAt),
  }
}

export async function listAuditEvents(params: ListAuditEventsParams = {}): Promise<AuditEvent[]> {
  const res = await auditClient.listAuditEvents({
    actorId: params.actorId,
    resourceType: params.resourceType ? toProtoResourceType(params.resourceType) : undefined,
    resourceId: params.resourceId,
    action: params.action ? toProtoAction(params.action) : undefined,
    since: params.since ? timestampFromDate(params.since) : undefined,
    until: params.until ? timestampFromDate(params.until) : undefined,
    limit: params.limit ?? 50,
    offset: params.offset ?? 0,
  })
  return (res.events ?? []).map(mapEvent)
}

export async function getAuditEvent(id: string): Promise<AuditEvent | undefined> {
  const res = await auditClient.getAuditEvent({ id })
  return res.event ? mapEvent(res.event) : undefined
}
