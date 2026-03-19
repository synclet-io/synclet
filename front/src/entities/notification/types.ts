export type ChannelType = 'slack' | 'email' | 'telegram'
export type NotificationCondition = 'on_failure' | 'on_consecutive_failures' | 'on_zero_records'

export interface NotificationChannel {
  id: string
  workspaceId: string
  name: string
  channelType: ChannelType
  config: Record<string, string>
  enabled: boolean
  createdAt: Date | undefined
  updatedAt: Date | undefined
}

export interface NotificationRule {
  id: string
  workspaceId: string
  channelId: string
  connectionId: string | undefined
  condition: NotificationCondition
  conditionValue: number
  enabled: boolean
  createdAt: Date | undefined
  updatedAt: Date | undefined
}
