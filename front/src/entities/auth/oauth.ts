import { authClient } from '@shared/api/services'
import { OAuthAccessLevel } from '@/gen/synclet/publicapi/auth/v1/auth_pb'

export type OAuthLevel = 'read' | 'actions' | 'write'

export interface OAuthClient {
  clientId: string
  clientName: string
}

export interface OAuthAuthorization {
  clientId: string
  redirectUri: string
  codeChallenge: string
  codeChallengeMethod: string
  level: OAuthLevel
  state: string
}

export const oauthLevelOptions: Array<{ value: OAuthLevel, label: string, description: string }> = [
  { value: 'read', label: 'Read', description: 'List and inspect sources, destinations, connections, jobs and workflows.' },
  { value: 'actions', label: 'Read + actions', description: 'Also trigger and cancel syncs, pause and resume connections, run workflows.' },
  { value: 'write', label: 'Read + actions + write', description: 'Also create, change and delete sources, destinations, connections and workflows.' },
]

const toProtoLevel: Record<OAuthLevel, OAuthAccessLevel> = {
  read: OAuthAccessLevel.OAUTH_ACCESS_LEVEL_READ,
  actions: OAuthAccessLevel.OAUTH_ACCESS_LEVEL_ACTIONS,
  write: OAuthAccessLevel.OAUTH_ACCESS_LEVEL_WRITE,
}

export function mapOAuthLevel(proto: OAuthAccessLevel): OAuthLevel | null {
  switch (proto) {
    case OAuthAccessLevel.OAUTH_ACCESS_LEVEL_READ: return 'read'
    case OAuthAccessLevel.OAUTH_ACCESS_LEVEL_ACTIONS: return 'actions'
    case OAuthAccessLevel.OAUTH_ACCESS_LEVEL_WRITE: return 'write'
    default: return null
  }
}

export function oauthLevelLabel(level: OAuthLevel): string {
  return oauthLevelOptions.find(o => o.value === level)?.label ?? level
}

export async function getOAuthClient(clientId: string, redirectUri: string): Promise<OAuthClient> {
  const res = await authClient.getOAuthClient({ clientId, redirectUri })
  return { clientId: res.clientId, clientName: res.clientName }
}

/** Issues an authorization code for `workspaceId` and returns the URL to send the browser to. */
export async function authorizeOAuth(workspaceId: string, req: OAuthAuthorization): Promise<string> {
  const res = await authClient.authorizeOAuth(
    { ...req, level: toProtoLevel[req.level] },
    { headers: { 'Workspace-Id': workspaceId } },
  )
  return res.redirectUrl
}
