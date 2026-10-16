export type RepositoryStatus = 'synced' | 'failed' | 'unknown'

export interface Repository {
  id: string
  name: string
  url: string
  hasAuth: boolean
  status: RepositoryStatus
  lastSyncedAt: Date | null
  connectorCount: number
  lastError: string | null
}
