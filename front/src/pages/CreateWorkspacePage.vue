<script setup lang="ts">
import { useAuth } from '@entities/auth'
import { useSystemInfo } from '@entities/system'
import { useCreateWorkspace } from '@entities/workspace'
import { listWorkspaces } from '@entities/workspace/api'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SButton, SCard, SInput, SyncletLogo } from '@shared/ui'
import { ref, watchEffect } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const auth = useAuth()
const { data: systemInfo } = useSystemInfo()
const createWorkspace = useCreateWorkspace()

// Redirect to dashboard in single-workspace mode.
watchEffect(() => {
  if (systemInfo.value?.workspacesMode === 'single') {
    router.replace({ name: 'dashboard' })
  }
})
const name = ref('')
const error = ref('')

async function handleSubmit() {
  error.value = ''
  try {
    await createWorkspace.mutateAsync(name.value)
    const ws = await listWorkspaces()
    auth.setWorkspaces(ws.map(({ workspace, role }) => ({
      workspaceId: workspace.id,
      workspaceName: workspace.name,
      role,
    })))
    router.push('/')
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to create workspace'
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-page">
    <div class="max-w-md w-full">
      <div class="flex items-center gap-2.5 mb-10">
        <SyncletLogo size="w-8 h-8" />
        <span class="text-lg font-semibold text-heading tracking-tight">Synclet</span>
      </div>
      <div class="mb-8">
        <h1 class="text-xl font-semibold text-heading">
          Create your workspace
        </h1>
        <p class="mt-1 text-sm text-text-secondary">
          Create your first workspace to get started
        </p>
      </div>
      <SCard>
        <form class="space-y-5" @submit.prevent="handleSubmit">
          <SAlert v-if="error" variant="danger" dismissible @dismiss="error = ''">
            {{ error }}
          </SAlert>
          <SInput v-model="name" label="Workspace name" placeholder="My Workspace" required />
          <SButton type="submit" :loading="createWorkspace.isPending.value" class="w-full">
            Create workspace
          </SButton>
        </form>
      </SCard>
    </div>
  </div>
</template>
