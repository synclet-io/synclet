import type { Connector, ConnectorSourceType, ConnectorSpecResult, ConnectorType, ConnectorVersionInfo, ExternalDocumentationUrl, LicenseType, ManagedConnector, ReleaseStage, SupportLevel, UpdateInfo } from './types'
import type { BreakingChange as ProtoBreakingChange, ConnectorInfo as ProtoConnectorInfo, ExternalDocumentationUrl as ProtoExternalDocumentationUrl, ManagedConnectorInfo as ProtoManagedConnectorInfo } from '@/gen/synclet/publicapi/registry/v1/registry_pb'
import { registryClient } from '@shared/api/services'
import {
  ConnectorType as ProtoConnectorType,
  License as ProtoLicense,
  ReleaseStage as ProtoReleaseStage,
  SourceType as ProtoSourceType,
  SupportLevel as ProtoSupportLevel,
} from '@/gen/synclet/publicapi/registry/v1/registry_pb'

function mapConnectorType(proto: ProtoConnectorType): ConnectorType {
  switch (proto) {
    case ProtoConnectorType.SOURCE: return 'source'
    case ProtoConnectorType.DESTINATION: return 'destination'
    default: return 'source'
  }
}

function toProtoConnectorType(t: ConnectorType): ProtoConnectorType {
  switch (t) {
    case 'source': return ProtoConnectorType.SOURCE
    case 'destination': return ProtoConnectorType.DESTINATION
  }
}

export function mapReleaseStage(proto: ProtoReleaseStage): ReleaseStage {
  switch (proto) {
    case ProtoReleaseStage.GENERALLY_AVAILABLE: return 'generally_available'
    case ProtoReleaseStage.BETA: return 'beta'
    case ProtoReleaseStage.ALPHA: return 'alpha'
    case ProtoReleaseStage.CUSTOM: return 'custom'
    default: return 'unknown'
  }
}

export function mapSupportLevel(proto: ProtoSupportLevel): SupportLevel {
  switch (proto) {
    case ProtoSupportLevel.COMMUNITY: return 'community'
    case ProtoSupportLevel.CERTIFIED: return 'certified'
    default: return 'community'
  }
}

export function mapLicense(proto: ProtoLicense): LicenseType {
  switch (proto) {
    case ProtoLicense.ELV2: return 'ELv2'
    case ProtoLicense.MIT: return 'MIT'
    default: return 'ELv2'
  }
}

export function mapSourceType(proto: ProtoSourceType): ConnectorSourceType {
  switch (proto) {
    case ProtoSourceType.API: return 'api'
    case ProtoSourceType.DATABASE: return 'database'
    case ProtoSourceType.FILE: return 'file'
    default: return 'api'
  }
}

function mapConnector(proto: ProtoConnectorInfo): Connector {
  return {
    dockerImage: proto.dockerImage,
    name: proto.name,
    iconUrl: proto.iconUrl,
    docsUrl: proto.docsUrl,
    releaseStage: mapReleaseStage(proto.releaseStage),
    latestVersion: proto.latestVersion,
    type: mapConnectorType(proto.type),
    supportLevel: mapSupportLevel(proto.supportLevel),
    license: mapLicense(proto.license),
    sourceType: mapSourceType(proto.sourceType),
  }
}

function mapUpdateInfo(proto: ProtoManagedConnectorInfo): UpdateInfo | null {
  if (!proto?.updateInfo)
    return null
  return {
    availableVersion: proto.updateInfo.availableVersion ?? '',
    hasUpdate: proto.updateInfo.hasUpdate ?? false,
    breakingChanges: (proto.updateInfo.breakingChanges || []).map((bc: ProtoBreakingChange) => ({
      version: bc.version ?? '',
      message: bc.message ?? '',
      migrationDocumentationUrl: bc.migrationDocumentationUrl ?? '',
      upgradeDeadline: bc.upgradeDeadline ?? '',
    })),
  }
}

function mapManagedConnector(proto: ProtoManagedConnectorInfo): ManagedConnector {
  return {
    id: proto.id,
    dockerImage: proto.dockerImage,
    dockerTag: proto.dockerTag,
    name: proto.name,
    connectorType: mapConnectorType(proto.connectorType),
    repositoryId: proto.repositoryId || null,
    updateInfo: mapUpdateInfo(proto),
  }
}

export async function listConnectors(type?: ConnectorType): Promise<Connector[]> {
  const res = await registryClient.listConnectors(type ? { type: toProtoConnectorType(type) } : {})
  return (res.connectors || []).map(mapConnector)
}

export async function getManagedConnector(id: string): Promise<ManagedConnector> {
  const res = await registryClient.getManagedConnector({ id })
  return mapManagedConnector(res)
}

export async function listManagedConnectors(params?: {
  repositoryId?: string | null
  search?: string
}): Promise<ManagedConnector[]> {
  const res = await registryClient.listManagedConnectors({})
  let connectors = (res.connectors || []).map(mapManagedConnector)

  // Client-side filtering until backend supports filter params
  if (params?.repositoryId !== undefined) {
    const filterRepoId = params.repositoryId
    if (filterRepoId === null) {
      connectors = connectors.filter(c => !c.repositoryId)
    }
    else {
      connectors = connectors.filter(c => c.repositoryId === filterRepoId)
    }
  }
  if (params?.search) {
    const q = params.search.toLowerCase()
    connectors = connectors.filter(c =>
      c.name.toLowerCase().includes(q)
      || c.dockerImage.toLowerCase().includes(q),
    )
  }

  return connectors
}

export async function addConnector(params: { dockerImage: string, dockerTag: string, name: string, connectorType: ConnectorType, repositoryId?: string }): Promise<{ id: string }> {
  const res = await registryClient.addConnector({
    dockerImage: params.dockerImage,
    dockerTag: params.dockerTag,
    name: params.name,
    connectorType: toProtoConnectorType(params.connectorType),
    repositoryId: params.repositoryId ?? '',
  })
  return { id: res.id }
}

export async function deleteManagedConnector(id: string): Promise<void> {
  await registryClient.deleteManagedConnector({ id })
}

export async function getConnectorSpec(id: string): Promise<ConnectorSpecResult> {
  const res = await registryClient.getConnectorSpec({ id })
  return {
    spec: res.spec ?? undefined,
    externalDocumentationUrls: (res.externalDocumentationUrls || []).map((d: ProtoExternalDocumentationUrl) => ({
      title: d.title,
      type: d.type as ExternalDocumentationUrl['type'],
      url: d.url,
    })),
  }
}

export async function getConnectorVersions(connectorImage: string): Promise<ConnectorVersionInfo> {
  const res = await registryClient.getConnectorVersions({ connectorImage })
  return { versions: res.versions || [], latestVersion: res.latestVersion }
}

export async function updateManagedConnector(id: string): Promise<ManagedConnector> {
  const res = await registryClient.updateManagedConnector({ id })
  if (!res.connector)
    throw new Error('connector not found in response')
  return mapManagedConnector(res.connector)
}

export async function batchUpdateConnectors(connectorIds: string[]): Promise<{ updatedCount: number, updatedConnectors: ManagedConnector[] }> {
  const res = await registryClient.batchUpdateConnectors({ connectorIds })
  return {
    updatedCount: res.updatedCount,
    updatedConnectors: (res.updatedConnectors || []).map(mapManagedConnector),
  }
}
