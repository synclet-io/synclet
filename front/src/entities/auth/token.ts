const WORKSPACE_ID_KEY = 'synclet_workspace_id'

export interface AuthMeta {
  accessExpiresAt: number
  refreshExpiresAt: number
}

export function getAuthMeta(): AuthMeta | null {
  const cookie = document.cookie
    .split('; ')
    .find(c => c.startsWith('synclet_auth='))
  if (!cookie) return null

  const params = new URLSearchParams(cookie.substring(cookie.indexOf('=') + 1))
  const accessExpires = params.get('access_expires')
  const refreshExpires = params.get('refresh_expires')
  if (!accessExpires || !refreshExpires) return null

  return {
    accessExpiresAt: Number(accessExpires),
    refreshExpiresAt: Number(refreshExpires),
  }
}

export function getWorkspaceId(): string | null {
  return localStorage.getItem(WORKSPACE_ID_KEY)
}

export function setWorkspaceId(id: string) {
  localStorage.setItem(WORKSPACE_ID_KEY, id)
}

export function clearWorkspaceId() {
  localStorage.removeItem(WORKSPACE_ID_KEY)
}
