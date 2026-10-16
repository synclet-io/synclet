import { beforeEach, describe, expect, it } from 'vitest'
import { isInviteToken, postAuthPath, rememberOIDCRedirect, safeRedirect, storePendingInvite, takeOIDCRedirect } from '../pendingInvite'

const token = 'a'.repeat(32) + '0123456789abcdef'.repeat(2)

describe('pendingInvite', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
  })

  it('accepts the 64-char hex token the backend issues', () => {
    expect(isInviteToken(token)).toBe(true)
    expect(isInviteToken('3fa85f64-5717-4562-b3fc-2c963f66afa6')).toBe(false)
    expect(isInviteToken(`${token}0`)).toBe(false)
  })

  it('sends the user to the stored invite after sign-in, once', () => {
    localStorage.setItem('pendingInviteToken', token)

    expect(postAuthPath('/jobs')).toBe(`/invite/${token}`)
    expect(postAuthPath('/jobs')).toBe('/jobs')
  })

  it('ignores a malformed stored invite', () => {
    storePendingInvite('not-a-token')
    localStorage.setItem('pendingInviteToken', '../settings')

    expect(postAuthPath(undefined)).toBe('/')
  })

  it('keeps same-origin redirects and rejects external or auth targets', () => {
    expect(safeRedirect('/connections/42?tab=streams#x')).toBe('/connections/42?tab=streams#x')
    expect(safeRedirect('//evil.example/path')).toBeNull()
    expect(safeRedirect('/\\evil.example')).toBeNull()
    expect(safeRedirect('https://evil.example')).toBeNull()
    expect(safeRedirect('/login?redirect=/jobs')).toBeNull()
    expect(safeRedirect(['/jobs'])).toBeNull()
  })

  it('carries the redirect across the OIDC round trip', () => {
    rememberOIDCRedirect('/jobs/7')

    expect(postAuthPath(takeOIDCRedirect())).toBe('/jobs/7')
    expect(takeOIDCRedirect()).toBeNull()
  })
})
