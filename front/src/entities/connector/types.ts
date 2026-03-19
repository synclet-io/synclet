export type ConnectorType = 'source' | 'destination'

export type SupportLevel = 'community' | 'certified'
export type LicenseType = 'ELv2' | 'MIT'
export type ConnectorSourceType = 'api' | 'database' | 'file'
export type ReleaseStage = 'generally_available' | 'beta' | 'alpha' | 'custom' | 'unknown'

export interface Connector {
  dockerImage: string
  name: string
  iconUrl: string
  docsUrl: string
  releaseStage: ReleaseStage
  latestVersion: string
  type: ConnectorType
  supportLevel: SupportLevel
  license: LicenseType
  sourceType: ConnectorSourceType
}

export interface ConnectorVersionInfo {
  versions: string[]
  latestVersion: string
}

export interface BreakingChange {
  version: string
  message: string
  migrationDocumentationUrl: string
  upgradeDeadline: string
}

export interface UpdateInfo {
  availableVersion: string
  hasUpdate: boolean
  breakingChanges: BreakingChange[]
}

export interface ManagedConnector {
  id: string
  dockerImage: string
  dockerTag: string
  name: string
  connectorType: ConnectorType
  repositoryId?: string | null
  updateInfo?: UpdateInfo | null
}

export type ExternalDocumentationType
  = | 'authentication_guide'
    | 'api_reference'
    | 'api_status_page'
    | 'api_release_history'
    | 'rate_limits'

export interface ExternalDocumentationUrl {
  title: string
  type: ExternalDocumentationType
  url: string
}

export interface ConnectorSpecResult {
  spec: Record<string, unknown> | undefined
  externalDocumentationUrls: ExternalDocumentationUrl[]
}
