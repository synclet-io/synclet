<script setup lang="ts">
import type { OAuthClient, OAuthLevel } from '@entities/auth/oauth'
import { useAuth } from '@entities/auth'
import { authorizeOAuth, getOAuthClient, oauthLevelOptions } from '@entities/auth/oauth'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { redirectTo } from '@shared/lib/navigation'
import { SAlert, SButton, SCard, SSelect, SyncletLogo } from '@shared/ui'
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const auth = useAuth()

function queryParam(name: string): string {
  const value = route.query[name]
  return typeof value === 'string' ? value : ''
}

const clientId = queryParam('client_id')
const redirectUri = queryParam('redirect_uri')
const state = queryParam('state')
const responseType = queryParam('response_type')

const client = ref<OAuthClient | null>(null)
const loading = ref(true)
const error = ref('')
const submitting = ref(false)

const workspaceOptions = computed(() => auth.workspaces.value.map(w => ({ label: w.workspaceName, value: w.workspaceId })))
const workspaceId = ref<string | number>(
  auth.workspaces.value.some(w => w.workspaceId === auth.currentWorkspaceId.value)
    ? auth.currentWorkspaceId.value ?? ''
    : auth.workspaces.value[0]?.workspaceId ?? '',
)
const level = ref<OAuthLevel>('read')

onMounted(async () => {
  if (!clientId || !redirectUri) {
    error.value = 'The authorization request is missing client_id or redirect_uri.'
    loading.value = false
    return
  }
  if (responseType && responseType !== 'code') {
    error.value = `Unsupported response_type "${responseType}".`
    loading.value = false
    return
  }
  try {
    client.value = await getOAuthClient(clientId, redirectUri)
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'This application is not registered.'
  }
  finally {
    loading.value = false
  }
})

async function allow() {
  error.value = ''
  submitting.value = true
  try {
    const url = await authorizeOAuth(String(workspaceId.value), {
      clientId,
      redirectUri,
      codeChallenge: queryParam('code_challenge'),
      codeChallengeMethod: queryParam('code_challenge_method'),
      level: level.value,
      state,
    })
    redirectTo(url)
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to authorize the application'
    submitting.value = false
  }
}

function deny() {
  const url = new URL(redirectUri)
  url.searchParams.set('error', 'access_denied')
  if (state)
    url.searchParams.set('state', state)
  redirectTo(url.toString())
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-page px-4">
    <div class="max-w-md w-full">
      <div class="flex items-center gap-2.5 mb-10">
        <SyncletLogo size="w-8 h-8" />
        <span class="text-lg font-semibold text-heading tracking-tight">Synclet</span>
      </div>
      <SCard>
        <p v-if="loading" class="text-sm text-text-secondary">
          Loading…
        </p>
        <SAlert v-else-if="!client" variant="danger">
          {{ error }}
        </SAlert>
        <form v-else class="space-y-5" @submit.prevent>
          <div>
            <h1 class="text-xl font-semibold text-heading">
              Allow {{ client.clientName }} to access Synclet?
            </h1>
            <p class="mt-1 text-sm text-text-secondary">
              {{ client.clientName }} will act as you in the workspace you choose, with the access level you pick. You can revoke it any time in Settings → API Keys.
            </p>
          </div>
          <SAlert v-if="error" variant="danger" dismissible @dismiss="error = ''">
            {{ error }}
          </SAlert>
          <SSelect v-model="workspaceId" label="Workspace" :options="workspaceOptions" required />
          <fieldset class="space-y-2">
            <legend class="text-sm font-medium text-heading mb-1.5">
              Access level
            </legend>
            <label
              v-for="option in oauthLevelOptions"
              :key="option.value"
              class="flex items-start gap-3 rounded border border-border px-3 py-2 cursor-pointer"
            >
              <input v-model="level" type="radio" name="level" :value="option.value" class="mt-1">
              <span>
                <span class="block text-sm font-medium text-text-primary">{{ option.label }}</span>
                <span class="block text-xs text-text-secondary">{{ option.description }}</span>
              </span>
            </label>
          </fieldset>
          <div class="flex gap-3">
            <SButton type="button" variant="secondary" class="flex-1" data-testid="oauth-deny" :disabled="submitting" @click="deny">
              Deny
            </SButton>
            <SButton type="button" class="flex-1" data-testid="oauth-allow" :loading="submitting" :disabled="!workspaceId" @click="allow">
              Allow
            </SButton>
          </div>
        </form>
      </SCard>
    </div>
  </div>
</template>
