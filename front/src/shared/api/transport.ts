import type { Interceptor } from '@connectrpc/connect'
import { ConnectError, createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { getWorkspaceId } from '@entities/auth/token'
import { AuthService } from '@/gen/synclet/publicapi/auth/v1/auth_pb'

let onAuthFailure: (() => void) | null = null

export function setOnAuthFailure(callback: () => void) {
  onAuthFailure = callback
}

// Separate transport without auth interceptor for refresh token calls to avoid infinite loop.
const refreshTransport = createConnectTransport({
  baseUrl: window.location.origin,
  fetch: (input, init) => globalThis.fetch(input, { ...init, credentials: 'same-origin' }),
})

const refreshClient = createClient(AuthService, refreshTransport)

const workspaceInterceptor: Interceptor = next => async (req) => {
  const workspaceId = getWorkspaceId()
  if (workspaceId) {
    req.header.set('Workspace-Id', workspaceId)
  }

  try {
    return await next(req)
  }
  catch (err: unknown) {
    if (err instanceof ConnectError && err.code === 16 /* Unauthenticated */) {
      const refreshed = await tryRefreshToken()
      if (refreshed) {
        return await next(req)
      }
      onAuthFailure?.()
    }
    throw err
  }
}

async function tryRefreshToken(): Promise<boolean> {
  try {
    await refreshClient.refreshToken({})
    return true
  }
  catch {
    return false
  }
}

export const transport = createConnectTransport({
  baseUrl: window.location.origin,
  fetch: (input, init) => globalThis.fetch(input, { ...init, credentials: 'same-origin' }),
  interceptors: [workspaceInterceptor],
})
