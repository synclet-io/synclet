import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemberRole } from '@/gen/synclet/publicapi/workspace/v1/workspace_pb'
import { createInvite, listMembers, resendInvite } from '../api'

const workspaceClient = vi.hoisted(() => ({
  listMembers: vi.fn(),
  createInvite: vi.fn(),
  resendInvite: vi.fn(),
}))

vi.mock('@shared/api/services', () => ({
  workspaceClient,
  connectionClient: {},
}))

describe('workspace api', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('maps member identity', async () => {
    workspaceClient.listMembers.mockResolvedValue({
      members: [{ id: 'm1', workspaceId: 'w1', userId: 'u1', email: 'ada@example.com', name: 'Ada', role: MemberRole.ADMIN }],
    })

    const members = await listMembers('w1')

    expect(members).toEqual([expect.objectContaining({ userId: 'u1', email: 'ada@example.com', name: 'Ada', role: 'admin' })])
  })

  it('returns the accept url from create invite', async () => {
    workspaceClient.createInvite.mockResolvedValue({
      invite: { id: 'i1', workspaceId: 'w1', email: 'bob@example.com', role: MemberRole.VIEWER },
      acceptUrl: 'https://app.example.com/invite/abc',
    })

    const created = await createInvite({ email: 'bob@example.com', role: 'viewer' })

    expect(created.acceptUrl).toBe('https://app.example.com/invite/abc')
    expect(created.invite.email).toBe('bob@example.com')
  })

  it('returns the accept url from resend invite', async () => {
    workspaceClient.resendInvite.mockResolvedValue({ acceptUrl: 'https://app.example.com/invite/abc' })

    await expect(resendInvite('i1')).resolves.toBe('https://app.example.com/invite/abc')
  })
})
