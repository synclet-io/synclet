export type AuditAction = 'create' | 'update' | 'delete' | 'enable' | 'disable' | 'test' | 'sync' | 'login' | 'logout'

export type AuditResourceType
  = | 'source'
    | 'destination'
    | 'connection'
    | 'connector'
    | 'repository'
    | 'notification_channel'
    | 'notification_rule'
    | 'webhook'
    | 'workspace'
    | 'workspace_member'
    | 'workspace_invite'
    | 'api_key'
    | 'user'

export type AuditActorType = 'user' | 'api_key' | 'system'

export interface AuditFieldChange {
  path: string
  before?: unknown
  after?: unknown
}

export interface AuditEvent {
  id: string
  workspaceId: string
  actorType: AuditActorType
  actorId: string
  actorLabel: string
  action: AuditAction
  resourceType: AuditResourceType
  resourceId: string
  resourceLabel: string
  changes: AuditFieldChange[]
  diffTruncated: boolean
  createdAt: Date | undefined
}

export interface ListAuditEventsParams {
  actorId?: string
  resourceType?: AuditResourceType
  resourceId?: string
  action?: AuditAction
  since?: Date
  until?: Date
  limit?: number
  offset?: number
}
