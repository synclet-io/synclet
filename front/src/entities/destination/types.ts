export interface Destination {
  id: string
  workspaceId: string
  name: string
  managedConnectorId: string
  config: Record<string, unknown>
  runtimeConfig: string | null
  executorGroup: string
  createdAt: Date | undefined
  updatedAt: Date | undefined
}
