<script setup lang="ts">
import type { MemberRole, WorkspaceInvite, WorkspaceMember } from '@entities/workspace'
import type { Column } from '@shared/ui'
import { useAuth } from '@entities/auth'
import {
  useCreateInvite,
  useInvites,
  useRemoveMember,
  useResendInvite,
  useRevokeInvite,
  useWorkspaceMembers,
} from '@entities/workspace'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SBadge, SButton, SConfirmDialog, SEmptyState, SInput, SSelect, STable, useToast } from '@shared/ui'
import { RefreshCw, Trash2, UserPlus, X } from 'lucide-vue-next'
import { computed, ref } from 'vue'

const auth = useAuth()
const toast = useToast()

const workspaceId = computed(() => auth.currentWorkspaceId.value ?? '')
const currentRole = computed(() => {
  const ws = auth.workspaces.value.find(w => w.workspaceId === workspaceId.value)
  return ws?.role ?? 'viewer'
})
const isAdmin = computed(() => currentRole.value === 'admin')

// Data queries
const { data: members, isLoading: membersLoading } = useWorkspaceMembers(workspaceId)
const { data: invites, isLoading: invitesLoading } = useInvites()

const loading = computed(() => membersLoading.value || invitesLoading.value)

// Invite form state
const inviteEmail = ref('')
const inviteRole = ref<MemberRole>('viewer')
const error = ref('')

// Mutations
const createInvite = useCreateInvite()
const removeMember = useRemoveMember()
const revokeInvite = useRevokeInvite()
const resendInvite = useResendInvite()

const roleOptions = [
  { label: 'Admin', value: 'admin' },
  { label: 'Editor', value: 'editor' },
  { label: 'Viewer', value: 'viewer' },
]

// Combined table data
type TableRow
  = | { type: 'member', member: WorkspaceMember }
    | { type: 'invite', invite: WorkspaceInvite }

const tableRows = computed<TableRow[]>(() => {
  const rows: TableRow[] = []
  if (members.value) {
    for (const m of members.value) {
      rows.push({ type: 'member', member: m })
    }
  }
  if (invites.value) {
    const activeInvites = invites.value.filter(i => i.status !== 'accepted')
    const sorted = [...activeInvites].sort((a, b) => {
      const at = a.createdAt?.getTime() ?? 0
      const bt = b.createdAt?.getTime() ?? 0
      return bt - at
    })
    for (const i of sorted) {
      rows.push({ type: 'invite', invite: i })
    }
  }
  return rows
})

const isEmpty = computed(() => {
  const otherMembers = (members.value ?? []).filter(m => m.userId !== auth.user.value?.id)
  const activeInvites = (invites.value ?? []).filter(i => i.status !== 'accepted')
  return otherMembers.length === 0 && activeInvites.length === 0
})

// Confirm dialogs
const confirmRemove = ref<{ open: boolean, userId: string }>({ open: false, userId: '' })
const confirmRevoke = ref<{ open: boolean, inviteId: string, email: string }>({ open: false, inviteId: '', email: '' })

async function handleInvite() {
  if (!inviteEmail.value)
    return
  error.value = ''
  try {
    await createInvite.mutateAsync({ email: inviteEmail.value, role: inviteRole.value })
    toast.success(`Invitation sent to ${inviteEmail.value}`)
    inviteEmail.value = ''
    inviteRole.value = 'viewer'
  }
  catch (e: unknown) {
    const msg = getErrorMessage(e) || 'Failed to send invitation'
    if (msg.toLowerCase().includes('already a member') || msg.toLowerCase().includes('already_exists')) {
      error.value = 'This user is already a member of this workspace.'
    }
    else {
      error.value = msg
    }
  }
}

async function handleRemove() {
  const userId = confirmRemove.value.userId
  confirmRemove.value.open = false
  try {
    await removeMember.mutateAsync({ workspaceId: workspaceId.value, userId })
    toast.success('Member removed')
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to remove member'
  }
}

async function handleRevoke() {
  const inviteId = confirmRevoke.value.inviteId
  confirmRevoke.value = { open: false, inviteId: '', email: '' }
  try {
    await revokeInvite.mutateAsync(inviteId)
    toast.success('Invitation revoked')
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to revoke invitation'
  }
}

async function handleResend(invite: WorkspaceInvite) {
  try {
    await resendInvite.mutateAsync(invite.id)
    toast.success(`Invitation resent to ${invite.email}`)
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to resend invitation'
  }
}

function isExpired(invite: WorkspaceInvite): boolean {
  if (invite.status === 'expired')
    return true
  if (invite.expiresAt && invite.expiresAt < new Date())
    return true
  return false
}

function relativeTime(date: Date | undefined): string {
  if (!date)
    return ''
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1)
    return 'just now'
  if (minutes < 60)
    return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24)
    return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

function roleVariant(role: string) {
  const map: Record<string, 'info' | 'success' | 'gray'> = { admin: 'info', editor: 'success', viewer: 'gray' }
  return map[role] || 'gray'
}

const columns: Column[] = [
  { key: 'user', label: 'User' },
  { key: 'role', label: 'Role' },
  { key: 'date', label: 'Joined / Sent' },
  { key: 'actions', label: '', align: 'right', width: '100px' },
]
</script>

<template>
  <div class="max-w-2xl space-y-6">
    <h2 class="text-lg font-semibold text-heading">
      Members
    </h2>

    <SAlert v-if="error" variant="danger" dismissible @dismiss="error = ''">
      {{ error }}
    </SAlert>

    <!-- Invite form (admin-only) -->
    <form v-if="isAdmin" class="flex gap-3" @submit.prevent="handleInvite">
      <div class="flex-1">
        <SInput v-model="inviteEmail" type="email" placeholder="Email address" required />
      </div>
      <SSelect v-model="inviteRole" :options="roleOptions" />
      <SButton type="submit" :loading="createInvite.isPending.value">
        Send Invite
      </SButton>
    </form>

    <!-- Empty state -->
    <template v-if="!loading && isEmpty">
      <SEmptyState
        title="Invite your team"
        description="Send invitations to collaborate in this workspace."
        :icon="UserPlus"
      />
    </template>

    <!-- Combined members + invites table -->
    <STable v-else :columns="columns" :data="tableRows" :loading="loading">
      <template #cell-user="{ row }">
        <template v-if="row.type === 'member'">
          <span class="text-sm text-text-primary">{{ row.member.userId }}</span>
        </template>
        <template v-else>
          <div class="flex items-center gap-2">
            <span class="text-sm text-text-primary">{{ row.invite.email }}</span>
            <SBadge v-if="row.invite.status === 'pending' && !isExpired(row.invite)" variant="warning" dot>
              Pending
            </SBadge>
            <SBadge v-else-if="row.invite.status === 'declined'" variant="danger">
              Declined
            </SBadge>
            <SBadge v-else-if="isExpired(row.invite)" variant="gray">
              Expired
            </SBadge>
            <SBadge v-else-if="row.invite.status === 'revoked'" variant="gray">
              Revoked
            </SBadge>
          </div>
        </template>
      </template>

      <template #cell-role="{ row }">
        <SBadge :variant="roleVariant(row.type === 'member' ? row.member.role : row.invite.role)">
          {{ row.type === 'member' ? row.member.role : row.invite.role }}
        </SBadge>
      </template>

      <template #cell-date="{ row }">
        <template v-if="row.type === 'member'">
          <span class="text-sm text-text-secondary">{{ row.member.joinedAt?.toLocaleDateString() ?? '-' }}</span>
        </template>
        <template v-else>
          <span class="text-sm text-text-secondary">Invited {{ relativeTime(row.invite.createdAt) }}</span>
        </template>
      </template>

      <template #cell-actions="{ row }">
        <div class="flex items-center justify-end gap-1">
          <!-- Member actions -->
          <template v-if="row.type === 'member'">
            <button
              v-if="isAdmin && row.member.userId !== auth.user.value?.id"
              class="p-1 text-text-muted hover:text-danger transition-colors"
              title="Remove member"
              @click="confirmRemove = { open: true, userId: row.member.userId }"
            >
              <Trash2 class="w-4 h-4" />
            </button>
          </template>

          <!-- Invite actions (admin-only) -->
          <template v-else-if="isAdmin">
            <!-- Resend (only for pending, non-expired) -->
            <button
              v-if="row.invite.status === 'pending' && !isExpired(row.invite)"
              class="p-1 text-text-muted hover:text-text-primary transition-colors"
              title="Resend invitation"
              @click="handleResend(row.invite)"
            >
              <RefreshCw class="w-4 h-4" />
            </button>
            <!-- Revoke / dismiss -->
            <button
              v-if="row.invite.status === 'pending' || row.invite.status === 'declined' || isExpired(row.invite)"
              class="p-1 text-text-muted hover:text-danger transition-colors"
              title="Revoke invitation"
              @click="confirmRevoke = { open: true, inviteId: row.invite.id, email: row.invite.email }"
            >
              <X class="w-4 h-4" />
            </button>
          </template>
        </div>
      </template>
    </STable>

    <!-- Remove member confirmation -->
    <SConfirmDialog
      :open="confirmRemove.open"
      title="Remove member"
      message="Are you sure you want to remove this member?"
      confirm-text="Remove"
      @confirm="handleRemove"
      @cancel="confirmRemove.open = false"
    />

    <!-- Revoke invite confirmation -->
    <SConfirmDialog
      :open="confirmRevoke.open"
      title="Revoke invitation"
      :message="`This will invalidate the invitation link sent to ${confirmRevoke.email}. Continue?`"
      confirm-text="Revoke"
      @confirm="handleRevoke"
      @cancel="confirmRevoke = { open: false, inviteId: '', email: '' }"
    />
  </div>
</template>
