import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemberRole } from '@/gen/synclet/publicapi/workspace/v1/workspace_pb'
import { useAuth } from '../composables'

const authClient = vi.hoisted(() => ({
  login: vi.fn(),
  register: vi.fn(),
  getCurrentUser: vi.fn(),
  logout: vi.fn(),
}))

vi.mock('@shared/api/services', async importOriginal => ({
  ...(await importOriginal<typeof import('@shared/api/services')>()),
  authClient,
}))

const response = {
  user: { id: 'u1', email: 'a@example.com', name: 'A' },
  workspaces: [
    { workspaceId: 'w1', workspaceName: 'First', role: MemberRole.ADMIN },
    { workspaceId: 'w2', workspaceName: 'Second', role: MemberRole.VIEWER },
  ],
}

const expected = [
  { workspaceId: 'w1', workspaceName: 'First', role: 'admin' },
  { workspaceId: 'w2', workspaceName: 'Second', role: 'viewer' },
]

describe('useAuth password sign-in', () => {
  beforeEach(async () => {
    vi.clearAllMocks()
    localStorage.clear()
    authClient.logout.mockResolvedValue({})
    await useAuth().logout()
  })

  it('takes workspaces from the login response without calling GetCurrentUser', async () => {
    authClient.login.mockResolvedValue(response)
    const auth = useAuth()

    await auth.login('a@example.com', 'secret')

    expect(auth.user.value?.id).toBe('u1')
    expect(auth.workspaces.value).toEqual(expected)
    expect(auth.currentWorkspaceId.value).toBe('w1')
    expect(authClient.getCurrentUser).not.toHaveBeenCalled()
  })

  it('takes workspaces from the register response without calling GetCurrentUser', async () => {
    authClient.register.mockResolvedValue(response)
    const auth = useAuth()

    await auth.register('a@example.com', 'secret', 'A')

    expect(auth.user.value?.id).toBe('u1')
    expect(auth.workspaces.value).toEqual(expected)
    expect(auth.currentWorkspaceId.value).toBe('w1')
    expect(authClient.getCurrentUser).not.toHaveBeenCalled()
  })
})
