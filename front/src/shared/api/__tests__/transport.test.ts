import { ConnectError, createClient } from '@connectrpc/connect'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { AuthService } from '@/gen/synclet/publicapi/auth/v1/auth_pb'

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function fakeBackend(refreshSucceeds: boolean) {
  let accessValid = false
  let refreshCalls = 0
  const fetch = vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input)
    if (url.endsWith('/RefreshToken')) {
      refreshCalls++
      await new Promise(resolve => setTimeout(resolve, 10))
      if (!refreshSucceeds)
        return json(401, { code: 'unauthenticated', message: 'refresh token revoked' })
      accessValid = true
      return json(200, {})
    }
    if (!accessValid)
      return json(401, { code: 'unauthenticated', message: 'access token expired' })
    return json(200, {})
  })
  return { fetch, refreshCalls: () => refreshCalls }
}

function concurrently<T>(count: number, call: () => Promise<T>) {
  const calls: Promise<T>[] = []
  for (let i = 0; i < count; i++)
    calls.push(call())
  return Promise.allSettled(calls)
}

async function loadTransport() {
  vi.resetModules()
  return import('../transport')
}

describe('transport auth refresh', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('shares one refresh across concurrent expired-token calls and keeps the session', async () => {
    const backend = fakeBackend(true)
    vi.stubGlobal('fetch', backend.fetch)
    const { transport, setOnAuthFailure } = await loadTransport()
    const onAuthFailure = vi.fn()
    setOnAuthFailure(onAuthFailure)
    const client = createClient(AuthService, transport)

    const results = await concurrently(4, () => client.getCurrentUser({}))

    expect(results.map(r => r.status)).toEqual(['fulfilled', 'fulfilled', 'fulfilled', 'fulfilled'])
    expect(backend.refreshCalls()).toBe(1)
    expect(onAuthFailure).not.toHaveBeenCalled()
  })

  it('reports auth failure once when the shared refresh is rejected', async () => {
    const backend = fakeBackend(false)
    vi.stubGlobal('fetch', backend.fetch)
    const { transport, setOnAuthFailure } = await loadTransport()
    const onAuthFailure = vi.fn()
    setOnAuthFailure(onAuthFailure)
    const client = createClient(AuthService, transport)

    const results = await concurrently(3, () => client.getCurrentUser({}))

    expect(results.every(r => r.status === 'rejected' && r.reason instanceof ConnectError)).toBe(true)
    expect(backend.refreshCalls()).toBe(1)
    expect(onAuthFailure).toHaveBeenCalledTimes(1)
  })
})

describe('transport workspace header', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it('sends the stored workspace unless the call names one', async () => {
    localStorage.setItem('synclet_workspace_id', 'ws-stored')
    const sent: Array<string | null> = []
    vi.stubGlobal('fetch', vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      sent.push(new Headers(init?.headers).get('Workspace-Id'))
      return json(200, {})
    }))
    const { transport } = await loadTransport()
    const client = createClient(AuthService, transport)

    await client.getCurrentUser({})
    await client.getCurrentUser({}, { headers: { 'Workspace-Id': 'ws-chosen' } })

    expect(sent).toEqual(['ws-stored', 'ws-chosen'])
  })
})
