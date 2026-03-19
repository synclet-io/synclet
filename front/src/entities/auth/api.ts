import type { MemberRole } from '@entities/workspace'
import type { OIDCProvider, User, WorkspaceMembership } from './types'
import { authClient } from '@shared/api/services'
import { tsToDate } from '@shared/lib/formatting'
import type { APIKeyInfo as ProtoAPIKeyInfo, UserInfo as ProtoUserInfo, WorkspaceMembership as ProtoWorkspaceMembership } from '@/gen/synclet/publicapi/auth/v1/auth_pb'
import { MemberRole as ProtoMemberRole } from '@/gen/synclet/publicapi/workspace/v1/workspace_pb'

function mapMemberRole(proto: ProtoMemberRole): MemberRole {
  switch (proto) {
    case ProtoMemberRole.ADMIN: return 'admin'
    case ProtoMemberRole.EDITOR: return 'editor'
    case ProtoMemberRole.VIEWER: return 'viewer'
    default: return 'viewer'
  }
}

function mapUser(proto: ProtoUserInfo): User {
  return {
    id: proto.id,
    email: proto.email,
    name: proto.name,
    createdAt: tsToDate(proto.createdAt),
  }
}

function mapMembership(proto: ProtoWorkspaceMembership): WorkspaceMembership {
  return {
    workspaceId: proto.workspaceId,
    workspaceName: proto.workspaceName,
    role: mapMemberRole(proto.role),
  }
}

export async function login(email: string, password: string): Promise<{ user: User | undefined }> {
  const res = await authClient.login({ email, password })
  return {
    user: res.user ? mapUser(res.user) : undefined,
  }
}

export async function register(email: string, password: string, name: string): Promise<{ user: User | undefined }> {
  const res = await authClient.register({ email, password, name })
  return {
    user: res.user ? mapUser(res.user) : undefined,
  }
}

export async function refreshToken(): Promise<void> {
  await authClient.refreshToken({})
}

export async function getCurrentUser(): Promise<{ user: User | undefined, workspaces: WorkspaceMembership[] }> {
  const res = await authClient.getCurrentUser({})
  return {
    user: res.user ? mapUser(res.user) : undefined,
    workspaces: (res.workspaces || []).map(mapMembership),
  }
}

export async function logout(): Promise<void> {
  await authClient.logout({})
}

export async function updateProfile(name: string): Promise<User | undefined> {
  const res = await authClient.updateProfile({ name })
  return res.user ? mapUser(res.user) : undefined
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  await authClient.changePassword({ currentPassword, newPassword })
}

// API Key types and operations
export interface APIKey {
  id: string
  workspaceId: string
  name: string
  createdAt: Date | undefined
  expiresAt: Date | undefined
  lastUsedAt: Date | undefined
}

function mapAPIKey(proto: ProtoAPIKeyInfo): APIKey {
  return {
    id: proto.id,
    workspaceId: proto.workspaceId,
    name: proto.name,
    createdAt: tsToDate(proto.createdAt),
    expiresAt: tsToDate(proto.expiresAt),
    lastUsedAt: tsToDate(proto.lastUsedAt),
  }
}

export async function listAPIKeys(workspaceId: string): Promise<APIKey[]> {
  const res = await authClient.listAPIKeys({ workspaceId })
  return (res.apiKeys || []).map(mapAPIKey)
}

export async function createAPIKey(workspaceId: string, name: string): Promise<{ key: APIKey | undefined, rawKey: string }> {
  const res = await authClient.createAPIKey({ workspaceId, name })
  return {
    key: res.apiKey ? mapAPIKey(res.apiKey) : undefined,
    rawKey: res.rawKey,
  }
}

export async function revokeAPIKey(id: string): Promise<void> {
  await authClient.revokeAPIKey({ id })
}

export async function getOIDCProviders(): Promise<OIDCProvider[]> {
  const res = await authClient.getOIDCProviders({})
  return (res.providers || []).map(p => ({
    slug: p.slug,
    displayName: p.displayName,
  }))
}
