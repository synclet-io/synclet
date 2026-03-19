export type MemberRole = 'admin' | 'editor' | 'viewer'

export interface Workspace {
  id: string
  name: string
  slug: string
  createdAt: Date | undefined
  updatedAt: Date | undefined
}

export interface WorkspaceMember {
  id: string
  workspaceId: string
  userId: string
  role: MemberRole
  joinedAt: Date | undefined
}

export type InviteStatus = 'pending' | 'accepted' | 'declined' | 'revoked' | 'expired'

export interface WorkspaceInvite {
  id: string
  workspaceId: string
  workspaceName: string
  inviterUserId: string
  inviterName: string
  email: string
  role: MemberRole
  status: InviteStatus
  expiresAt: Date | undefined
  createdAt: Date | undefined
}
