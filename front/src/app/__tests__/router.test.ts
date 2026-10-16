import { useAuth } from '@entities/auth'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { router } from '../router'

const authClient = vi.hoisted(() => ({
  register: vi.fn(),
  getCurrentUser: vi.fn(),
}))

vi.mock('@shared/api/services', async importOriginal => ({
  ...(await importOriginal<typeof import('@shared/api/services')>()),
  authClient,
}))

vi.mock('@entities/system/api', () => ({
  getSystemInfo: vi.fn().mockResolvedValue({ workspacesMode: 'multi', registrationEnabled: true }),
}))

describe('router guard after password sign-in', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('sends a new user without workspaces to workspace creation without calling GetCurrentUser', async () => {
    authClient.register.mockResolvedValue({
      user: { id: 'u1', email: 'a@example.com', name: 'A' },
      workspaces: [],
    })
    authClient.getCurrentUser.mockResolvedValue({ user: { id: 'u1' }, workspaces: [] })

    await useAuth().register('a@example.com', 'secret', 'A')
    await router.push('/')

    expect(router.currentRoute.value.name).toBe('create-workspace')
    expect(authClient.getCurrentUser).not.toHaveBeenCalled()
  })
})
