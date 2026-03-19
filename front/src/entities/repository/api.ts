import type { Connector, ConnectorSourceType, ConnectorType, LicenseType, SupportLevel } from '@entities/connector/types'
import type { Repository, RepositoryStatus } from './types'
import type {
  Repository as ProtoRepository,
} from '@/gen/synclet/publicapi/registry/v1/registry_pb'
import { mapLicense, mapReleaseStage, mapSourceType, mapSupportLevel } from '@entities/connector/api'
import { registryClient } from '@shared/api/services'
import {
  ConnectorType as ProtoConnectorType,
  License as ProtoLicense,
  RepositoryStatus as ProtoRepositoryStatus,
  SourceType as ProtoSourceType,
  SupportLevel as ProtoSupportLevel,
} from '@/gen/synclet/publicapi/registry/v1/registry_pb'

function mapRepositoryStatus(proto: ProtoRepositoryStatus): RepositoryStatus {
  switch (proto) {
    case ProtoRepositoryStatus.SYNCING: return 'syncing'
    case ProtoRepositoryStatus.SYNCED: return 'synced'
    case ProtoRepositoryStatus.FAILED: return 'failed'
    default: return 'syncing'
  }
}

function mapRepository(proto: ProtoRepository): Repository {
  return {
    id: proto.id,
    name: proto.name,
    url: proto.url,
    hasAuth: proto.hasAuth,
    status: mapRepositoryStatus(proto.status),
    lastSyncedAt: proto.lastSyncedAt || null,
    connectorCount: proto.connectorCount,
    lastError: proto.lastError || null,
  }
}

function toProtoConnectorType(t: ConnectorType): ProtoConnectorType {
  switch (t) {
    case 'source': return ProtoConnectorType.SOURCE
    case 'destination': return ProtoConnectorType.DESTINATION
  }
}

function mapToProtoSupportLevel(value?: SupportLevel): ProtoSupportLevel {
  switch (value) {
    case 'community': return ProtoSupportLevel.COMMUNITY
    case 'certified': return ProtoSupportLevel.CERTIFIED
    default: return ProtoSupportLevel.UNSPECIFIED
  }
}

function mapToProtoLicense(value?: LicenseType): ProtoLicense {
  switch (value) {
    case 'ELv2': return ProtoLicense.ELV2
    case 'MIT': return ProtoLicense.MIT
    default: return ProtoLicense.UNSPECIFIED
  }
}

function mapToProtoSourceType(value?: ConnectorSourceType): ProtoSourceType {
  switch (value) {
    case 'api': return ProtoSourceType.API
    case 'database': return ProtoSourceType.DATABASE
    case 'file': return ProtoSourceType.FILE
    default: return ProtoSourceType.UNSPECIFIED
  }
}

export async function listRepositories(): Promise<Repository[]> {
  const res = await registryClient.listRepositories({})
  return res.repositories.map(mapRepository)
}

export async function addRepository(params: { name: string, url: string, authHeader?: string }): Promise<Repository> {
  const res = await registryClient.addRepository({
    name: params.name,
    url: params.url,
    authHeader: params.authHeader ?? '',
  })
  return mapRepository(res.repository!)
}

export async function deleteRepository(id: string): Promise<{ affectedConnectors: number }> {
  const res = await registryClient.deleteRepository({ id })
  return { affectedConnectors: res.affectedConnectors }
}

export async function syncRepository(id: string): Promise<Repository> {
  const res = await registryClient.syncRepository({ id })
  return mapRepository(res.repository!)
}

export async function listRepositoryConnectors(
  repositoryId: string,
  type?: ConnectorType,
  search?: string,
  filter?: { supportLevel?: SupportLevel, license?: LicenseType, sourceType?: ConnectorSourceType },
): Promise<Connector[]> {
  const res = await registryClient.listRepositoryConnectors({
    repositoryId,
    type: type ? toProtoConnectorType(type) : ProtoConnectorType.UNSPECIFIED,
    search: search ?? '',
    filter: {
      supportLevel: mapToProtoSupportLevel(filter?.supportLevel),
      license: mapToProtoLicense(filter?.license),
      sourceType: mapToProtoSourceType(filter?.sourceType),
    },
  })
  return res.connectors.map(c => ({
    dockerImage: c.dockerImage,
    name: c.name,
    iconUrl: c.iconUrl,
    docsUrl: c.docsUrl,
    releaseStage: mapReleaseStage(c.releaseStage),
    latestVersion: c.latestVersion,
    type: (c.type === ProtoConnectorType.SOURCE ? 'source' : 'destination') as ConnectorType,
    supportLevel: mapSupportLevel(c.supportLevel),
    license: mapLicense(c.license),
    sourceType: mapSourceType(c.sourceType),
  }))
}
