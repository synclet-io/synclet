import type { ManagedConnector } from '@entities/connector'

export function formatConnector(connector: ManagedConnector | null | undefined): string {
  if (!connector)
    return ''
  return connector.dockerTag ? `${connector.name} (${connector.dockerTag})` : connector.name
}
