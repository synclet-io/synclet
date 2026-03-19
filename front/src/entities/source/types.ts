export interface Source {
  id: string
  workspaceId: string
  name: string
  managedConnectorId: string
  config: Record<string, unknown>
  runtimeConfig: string | null
  createdAt: Date | undefined
  updatedAt: Date | undefined
}

export interface SourceCatalog {
  catalog: Record<string, unknown>
  version: number
  discoveredAt: Date | undefined
}
