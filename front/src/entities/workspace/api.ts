import type { InviteStatus, MemberRole, Workspace, WorkspaceInvite, WorkspaceMember } from './types'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { connectionClient, workspaceClient } from '@shared/api/services'
import { tsToDate } from '@shared/lib/formatting'
import type { WorkspaceInfo as ProtoWorkspaceInfo, WorkspaceInviteInfo as ProtoWorkspaceInviteInfo, WorkspaceMemberInfo as ProtoWorkspaceMemberInfo } from '@/gen/synclet/publicapi/workspace/v1/workspace_pb'
import {
  InviteStatus as ProtoInviteStatus,
  MemberRole as ProtoMemberRole,
  WorkspaceService,
} from '@/gen/synclet/publicapi/workspace/v1/workspace_pb'

// Public transport without auth interceptor for unauthenticated invite endpoints
const publicTransport = createConnectTransport({
  baseUrl: window.location.origin,
})
const publicWorkspaceClient = createClient(WorkspaceService, publicTransport)

function mapMemberRole(proto: ProtoMemberRole): MemberRole {
  switch (proto) {
    case ProtoMemberRole.ADMIN: return 'admin'
    case ProtoMemberRole.EDITOR: return 'editor'
    case ProtoMemberRole.VIEWER: return 'viewer'
    default: return 'viewer'
  }
}

function toProtoMemberRole(r: MemberRole): ProtoMemberRole {
  switch (r) {
    case 'admin': return ProtoMemberRole.ADMIN
    case 'editor': return ProtoMemberRole.EDITOR
    case 'viewer': return ProtoMemberRole.VIEWER
  }
}

function mapWorkspace(proto: ProtoWorkspaceInfo): Workspace {
  return {
    id: proto.id,
    name: proto.name,
    slug: proto.slug,
    createdAt: tsToDate(proto.createdAt),
    updatedAt: tsToDate(proto.updatedAt),
  }
}

function mapMember(proto: ProtoWorkspaceMemberInfo): WorkspaceMember {
  return {
    id: proto.id,
    workspaceId: proto.workspaceId,
    userId: proto.userId,
    role: mapMemberRole(proto.role),
    joinedAt: tsToDate(proto.joinedAt),
  }
}

export async function listWorkspaces(): Promise<Workspace[]> {
  const res = await workspaceClient.listWorkspaces({})
  return (res.workspaces || []).map(mapWorkspace)
}

export async function getWorkspace(id: string): Promise<Workspace | undefined> {
  const res = await workspaceClient.getWorkspace({ id })
  return res.workspace ? mapWorkspace(res.workspace) : undefined
}

export async function updateWorkspace(
  id: string,
  params: { name?: string },
): Promise<Workspace | undefined> {
  const res = await workspaceClient.updateWorkspace({
    id,
    name: params.name ?? '',
  })
  return res.workspace ? mapWorkspace(res.workspace) : undefined
}

export async function createWorkspace(name: string): Promise<Workspace | undefined> {
  const res = await workspaceClient.createWorkspace({ name, ownerUserId: '' })
  return res.workspace ? mapWorkspace(res.workspace) : undefined
}

export async function deleteWorkspace(id: string): Promise<void> {
  await workspaceClient.deleteWorkspace({ id })
}

export async function listMembers(workspaceId: string): Promise<WorkspaceMember[]> {
  const res = await workspaceClient.listMembers({ workspaceId })
  return (res.members || []).map(mapMember)
}

export async function removeMember(workspaceId: string, userId: string): Promise<void> {
  await workspaceClient.removeMember({ workspaceId, userId })
}

function mapInviteStatus(proto: ProtoInviteStatus): InviteStatus {
  switch (proto) {
    case ProtoInviteStatus.PENDING: return 'pending'
    case ProtoInviteStatus.ACCEPTED: return 'accepted'
    case ProtoInviteStatus.DECLINED: return 'declined'
    case ProtoInviteStatus.REVOKED: return 'revoked'
    case ProtoInviteStatus.EXPIRED: return 'expired'
    default: return 'pending'
  }
}

function mapInvite(proto: ProtoWorkspaceInviteInfo): WorkspaceInvite {
  return {
    id: proto.id,
    workspaceId: proto.workspaceId,
    workspaceName: proto.workspaceName,
    inviterUserId: proto.inviterUserId,
    inviterName: proto.inviterName,
    email: proto.email,
    role: mapMemberRole(proto.role),
    status: mapInviteStatus(proto.status),
    expiresAt: tsToDate(proto.expiresAt),
    createdAt: tsToDate(proto.createdAt),
  }
}

export async function createInvite(params: { email: string, role: MemberRole }): Promise<WorkspaceInvite> {
  const res = await workspaceClient.createInvite({ email: params.email, role: toProtoMemberRole(params.role) })
  return mapInvite(res.invite!)
}

export async function acceptInvite(token: string): Promise<{ workspaceId: string, workspaceName: string }> {
  const res = await workspaceClient.acceptInvite({ token })
  return { workspaceId: res.workspaceId, workspaceName: res.workspaceName }
}

export async function declineInvite(token: string): Promise<void> {
  await publicWorkspaceClient.declineInvite({ token })
}

export async function revokeInvite(inviteId: string): Promise<void> {
  await workspaceClient.revokeInvite({ inviteId })
}

export async function resendInvite(inviteId: string): Promise<void> {
  await workspaceClient.resendInvite({ inviteId })
}

export async function listInvites(): Promise<WorkspaceInvite[]> {
  const res = await workspaceClient.listInvites({})
  return (res.invites || []).map(mapInvite)
}

export async function getInviteByToken(token: string): Promise<WorkspaceInvite> {
  const res = await publicWorkspaceClient.getInviteByToken({ token })
  return mapInvite(res.invite!)
}

export async function exportConfig(): Promise<Blob> {
  const res = await connectionClient.exportWorkspaceConfig({ workspaceId: '' })
  return new Blob([res.configYaml as unknown as BlobPart], { type: 'application/x-yaml' })
}

export async function importConfig(yamlFile: File): Promise<{ created: number, updated: number, errors: string[] }> {
  const bytes = new Uint8Array(await yamlFile.arrayBuffer())
  const res = await connectionClient.importWorkspaceConfig({ workspaceId: '', configYaml: bytes })
  return { created: res.created, updated: res.updated, errors: res.errors || [] }
}
