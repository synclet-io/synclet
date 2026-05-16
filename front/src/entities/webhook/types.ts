export type WebhookEvent
  = | 'sync.completed'
    | 'sync.failed'
    | 'connection.paused'
    | 'schema.changed'
    | '*'

export const WEBHOOK_EVENTS: { value: WebhookEvent, label: string }[] = [
  { value: 'sync.completed', label: 'Sync completed' },
  { value: 'sync.failed', label: 'Sync failed' },
  { value: 'connection.paused', label: 'Connection paused' },
  { value: 'schema.changed', label: 'Schema changed' },
  { value: '*', label: 'All events' },
]

export interface Webhook {
  id: string
  workspaceId: string
  url: string
  events: WebhookEvent[]
  enabled: boolean
  createdAt: Date | undefined
  updatedAt: Date | undefined
}

export interface CreateWebhookInput {
  url: string
  events: WebhookEvent[]
  secret?: string
}

export interface UpdateWebhookInput {
  id: string
  url?: string
  events?: WebhookEvent[]
  enabled?: boolean
}
