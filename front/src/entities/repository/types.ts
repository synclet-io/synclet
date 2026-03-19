export type RepositoryStatus = 'syncing' | 'synced' | 'failed'

export interface Repository {
  id: string
  name: string
  url: string
  hasAuth: boolean
  status: RepositoryStatus
  lastSyncedAt: string | null
  connectorCount: number
  lastError: string | null
}
