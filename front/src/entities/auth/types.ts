import type { MemberRole } from '@entities/workspace'

export interface User {
  id: string
  email: string
  name: string
  createdAt: Date | undefined
}

export interface LoginCredentials {
  email: string
  password: string
}

export interface RegisterCredentials {
  email: string
  password: string
  name: string
}

export interface WorkspaceMembership {
  workspaceId: string
  workspaceName: string
  role: MemberRole
}

export interface OIDCProvider {
  slug: string
  displayName: string
}
