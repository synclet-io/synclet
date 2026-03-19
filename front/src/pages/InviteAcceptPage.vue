<script setup lang="ts">
import type { OIDCProvider } from '@entities/auth'
import type { MemberRole } from '@entities/workspace'
import { getOIDCProviders, useAuth } from '@entities/auth'
import { useAcceptInvite, useDeclineInvite, useInviteByToken } from '@entities/workspace'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SBadge, SButton, SCard, SConfirmDialog, SInput, SSkeleton } from '@shared/ui'
import { useToast } from '@shared/ui/useToast'
import { Zap } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'

import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const auth = useAuth()
const toast = useToast()

const token = computed(() => route.params.token as string)
const { data: invite, isLoading, isError } = useInviteByToken(token)

const acceptMutation = useAcceptInvite()
const declineMutation = useDeclineInvite()

const showDeclineConfirm = ref(false)
const loginEmail = ref('')
const loginPassword = ref('')
const loginError = ref('')
const loginLoading = ref(false)
const oidcProviders = ref<OIDCProvider[]>([])

// Store pending invite token for unauthenticated users
onMounted(async () => {
  if (!auth.isAuthenticated.value) {
    localStorage.setItem('pendingInviteToken', token.value)
  }
  try {
    oidcProviders.value = await getOIDCProviders()
  }
  catch {
    // OIDC not configured, ignore.
  }
})

// Determine the current display state
type PageState = 'loading' | 'invalid' | 'expired' | 'revoked' | 'accepted' | 'declined' | 'logged-out' | 'correct-user' | 'wrong-account'

const pageState = computed<PageState>(() => {
  if (isLoading.value)
    return 'loading'
  if (isError.value || !invite.value)
    return 'invalid'

  const status = invite.value.status
  if (status === 'expired')
    return 'expired'
  if (status === 'revoked')
    return 'revoked'
  if (status === 'accepted')
    return 'accepted'
  if (status === 'declined')
    return 'declined'

  // Valid pending invite
  if (!auth.isAuthenticated.value)
    return 'logged-out'

  // Authenticated: check email match
  const userEmail = auth.user.value?.email?.toLowerCase() ?? ''
  const inviteEmail = invite.value.email.toLowerCase()
  if (userEmail === inviteEmail)
    return 'correct-user'
  return 'wrong-account'
})

// Pre-fill login email from invite data when available
const displayEmail = computed(() => invite.value?.email ?? '')

// Update login email when invite loads
onMounted(() => {
  if (invite.value?.email) {
    loginEmail.value = invite.value.email
  }
})
watch(() => invite.value?.email, (email) => {
  if (email && !loginEmail.value) {
    loginEmail.value = email
  }
})

const roleBadgeVariant = computed(() => {
  const map: Record<MemberRole, 'info' | 'success' | 'gray'> = {
    admin: 'info',
    editor: 'success',
    viewer: 'gray',
  }
  return invite.value ? map[invite.value.role] : 'gray'
})

const roleLabel = computed(() => {
  if (!invite.value)
    return ''
  return invite.value.role.charAt(0).toUpperCase() + invite.value.role.slice(1)
})

async function handleLogin() {
  loginLoading.value = true
  loginError.value = ''
  try {
    await auth.login(loginEmail.value, loginPassword.value)
    // Clear pending invite token since we're already on the invite page
    localStorage.removeItem('pendingInviteToken')
    // The page will reactively switch to authenticated state
  }
  catch (e: unknown) {
    loginError.value = getErrorMessage(e) || 'Login failed'
  }
  finally {
    loginLoading.value = false
  }
}

async function handleAccept() {
  try {
    await acceptMutation.mutateAsync(token.value)
    toast.success(`You joined ${invite.value?.workspaceName}`)
    router.push('/')
  }
  catch (e: unknown) {
    toast.error(getErrorMessage(e) || 'Failed to join workspace')
  }
}

async function handleDecline() {
  try {
    await declineMutation.mutateAsync(token.value)
    showDeclineConfirm.value = false
    // Page will reactively show declined state via query refetch
  }
  catch (e: unknown) {
    toast.error(getErrorMessage(e) || 'Failed to decline invitation')
    showDeclineConfirm.value = false
  }
}

async function handleSwitchAccount() {
  await auth.logout()
  // The page will reactively switch to logged-out state
  // and the token is already stored in localStorage from mount
  localStorage.setItem('pendingInviteToken', token.value)
}

function startOIDCLogin(slug: string) {
  // pendingInviteToken is already in localStorage from onMounted
  window.location.href = `/auth/oidc/${slug}/login`
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-page p-6">
    <div class="w-full max-w-sm">
      <!-- Logo -->
      <div class="flex items-center gap-2.5 mb-10">
        <div class="w-8 h-8 bg-primary rounded-lg flex items-center justify-center">
          <Zap class="w-4.5 h-4.5 text-white" />
        </div>
        <span class="text-lg font-semibold text-heading tracking-tight">Synclet</span>
      </div>

      <!-- Loading state -->
      <SCard v-if="pageState === 'loading'">
        <SSkeleton :count="3" />
      </SCard>

      <!-- Error: Invalid token -->
      <SAlert v-else-if="pageState === 'invalid'" variant="danger">
        Invalid invitation link. Check that you have the correct link, or ask the admin to resend.
      </SAlert>

      <!-- Error: Expired -->
      <SAlert v-else-if="pageState === 'expired'" variant="danger">
        This invite has expired. Ask the workspace admin to send a new invitation.
      </SAlert>

      <!-- Error: Revoked -->
      <SAlert v-else-if="pageState === 'revoked'" variant="danger">
        This invite is no longer valid. The workspace admin revoked this invitation.
      </SAlert>

      <!-- Already accepted -->
      <template v-else-if="pageState === 'accepted'">
        <SAlert variant="info" class="mb-4">
          You've already joined this workspace.
        </SAlert>
        <SButton variant="secondary" class="w-full" to="/">
          Go to workspace
        </SButton>
      </template>

      <!-- Declined -->
      <SAlert v-else-if="pageState === 'declined'" variant="warning">
        You declined this invitation. Contact the workspace admin if you changed your mind.
      </SAlert>

      <!-- Valid invite + NOT authenticated -->
      <template v-else-if="pageState === 'logged-out'">
        <div class="mb-8">
          <h1 class="text-xl font-semibold text-heading">
            You've been invited to <strong>{{ invite!.workspaceName }}</strong>
          </h1>
          <p class="mt-1 text-sm text-text-secondary">
            Sign in or create an account to accept this invitation.
          </p>
        </div>

        <SCard>
          <div class="mb-4 text-sm text-text-secondary">
            {{ invite!.inviterName }} invited you as
            <SBadge :variant="roleBadgeVariant" class="ml-1">
              {{ roleLabel }}
            </SBadge>
          </div>

          <form class="space-y-4" @submit.prevent="handleLogin">
            <SAlert v-if="loginError" variant="danger" dismissible @dismiss="loginError = ''">
              {{ loginError }}
            </SAlert>
            <SInput v-model="loginEmail" label="Email" type="email" placeholder="you@example.com" required readonly />
            <SInput v-model="loginPassword" label="Password" type="password" placeholder="Enter your password" required />
            <SButton type="submit" :loading="loginLoading" class="w-full">
              Sign in
            </SButton>
          </form>

          <div v-if="oidcProviders.length > 0" class="mt-4">
            <div class="relative my-4">
              <div class="absolute inset-0 flex items-center">
                <div class="w-full border-t border-border" />
              </div>
              <div class="relative flex justify-center text-xs">
                <span class="bg-surface px-2 text-text-secondary">or continue with</span>
              </div>
            </div>
            <div class="space-y-2">
              <SButton
                v-for="provider in oidcProviders"
                :key="provider.slug"
                variant="secondary"
                class="w-full"
                @click="startOIDCLogin(provider.slug)"
              >
                Sign in with {{ provider.displayName }}
              </SButton>
            </div>
          </div>

          <p class="mt-4 text-center text-sm text-text-secondary">
            Don't have an account?
            <RouterLink
              :to="{ path: '/register', query: { email: displayEmail } }"
              class="text-primary hover:text-primary-hover font-medium ml-1"
            >
              Create an account
            </RouterLink>
          </p>
        </SCard>
      </template>

      <!-- Valid invite + authenticated + correct email -->
      <template v-else-if="pageState === 'correct-user'">
        <div class="mb-8">
          <h1 class="text-xl font-semibold text-heading">
            You've been invited to <strong>{{ invite!.workspaceName }}</strong>
          </h1>
        </div>

        <SCard>
          <div class="mb-6 text-sm text-text-secondary">
            {{ invite!.inviterName }} invited you as
            <SBadge :variant="roleBadgeVariant" class="ml-1">
              {{ roleLabel }}
            </SBadge>
          </div>

          <div class="space-y-3">
            <SButton class="w-full" :loading="acceptMutation.isPending.value" @click="handleAccept">
              Join Workspace
            </SButton>
            <SButton variant="secondary" class="w-full" @click="showDeclineConfirm = true">
              Decline
            </SButton>
          </div>
        </SCard>

        <SConfirmDialog
          :open="showDeclineConfirm"
          title="Decline invitation"
          message="Are you sure you want to decline this invitation?"
          confirm-text="Decline"
          variant="danger"
          :loading="declineMutation.isPending.value"
          @confirm="handleDecline"
          @cancel="showDeclineConfirm = false"
        />
      </template>

      <!-- Valid invite + authenticated + wrong email -->
      <template v-else-if="pageState === 'wrong-account'">
        <SAlert variant="warning" class="mb-4">
          This invite is for {{ invite!.email }}. You are signed in as {{ auth.user.value?.email }}.
        </SAlert>
        <SButton variant="secondary" class="w-full" @click="handleSwitchAccount">
          Sign out and switch account
        </SButton>
      </template>
    </div>
  </div>
</template>
