<script setup lang="ts">
import type { APIKey } from '@entities/auth'
import type { Column } from '@shared/ui'
import { createAPIKey, listAPIKeys, revokeAPIKey, useAuth } from '@entities/auth'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SButton, SConfirmDialog, SInput, STable, useToast } from '@shared/ui'
import { Check, Copy, Trash2 } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'

const auth = useAuth()
const toast = useToast()

const keys = ref<APIKey[]>([])
const loading = ref(false)
const error = ref('')
const newKeyName = ref('')
const creating = ref(false)
const createdRawKey = ref<string | null>(null)
const copied = ref(false)
const confirmRevoke = ref<{ open: boolean, id: string, name: string }>({ open: false, id: '', name: '' })

async function loadKeys() {
  if (!auth.currentWorkspaceId.value)
    return
  loading.value = true
  try {
    keys.value = await listAPIKeys(auth.currentWorkspaceId.value)
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to load API keys'
  }
  finally {
    loading.value = false
  }
}

onMounted(loadKeys)

async function handleCreate() {
  if (!auth.currentWorkspaceId.value || !newKeyName.value)
    return
  creating.value = true
  error.value = ''
  createdRawKey.value = null
  try {
    const result = await createAPIKey(auth.currentWorkspaceId.value, newKeyName.value)
    createdRawKey.value = result.rawKey
    newKeyName.value = ''
    await loadKeys()
    toast.success('API key created')
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to create API key'
  }
  finally {
    creating.value = false
  }
}

async function handleRevoke() {
  const id = confirmRevoke.value.id
  confirmRevoke.value.open = false
  try {
    await revokeAPIKey(id)
    await loadKeys()
    toast.success('API key revoked')
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to revoke API key'
  }
}

async function copyKey() {
  if (!createdRawKey.value)
    return
  await navigator.clipboard.writeText(createdRawKey.value)
  copied.value = true
  setTimeout(() => {
    copied.value = false
  }, 2000)
}

const columns: Column[] = [
  { key: 'name', label: 'Name' },
  { key: 'createdAt', label: 'Created' },
  { key: 'lastUsedAt', label: 'Last used' },
  { key: 'expiresAt', label: 'Expires' },
  { key: 'actions', label: '', align: 'right', width: '40px' },
]
</script>

<template>
  <div class="max-w-2xl space-y-6">
    <h2 class="text-lg font-semibold text-heading">
      API Keys
    </h2>
    <SAlert v-if="error" variant="danger" dismissible @dismiss="error = ''">
      {{ error }}
    </SAlert>

    <!-- Created key banner -->
    <SAlert v-if="createdRawKey" variant="success">
      <p class="font-medium mb-2">
        API key created. Copy it now — it won't be shown again.
      </p>
      <div class="flex items-center gap-2">
        <code class="flex-1 px-3 py-2 bg-surface-raised rounded border border-border text-sm font-mono break-all">{{ createdRawKey }}</code>
        <button class="p-2 text-text-muted hover:text-text-primary transition-colors" title="Copy" @click="copyKey">
          <Check v-if="copied" class="w-5 h-5 text-success" />
          <Copy v-else class="w-5 h-5" />
        </button>
      </div>
    </SAlert>

    <form class="flex gap-3" @submit.prevent="handleCreate">
      <div class="flex-1">
        <SInput v-model="newKeyName" placeholder="Key name (e.g. CI/CD)" required />
      </div>
      <SButton type="submit" :loading="creating">
        Create key
      </SButton>
    </form>

    <STable :columns="columns" :data="keys" :loading="loading" empty-text="No API keys yet.">
      <template #cell-name="{ row }">
        <span class="font-medium text-text-primary">{{ row.name }}</span>
      </template>
      <template #cell-createdAt="{ row }">
        <span class="text-text-secondary">{{ row.createdAt?.toLocaleDateString() ?? '-' }}</span>
      </template>
      <template #cell-lastUsedAt="{ row }">
        <span class="text-text-secondary">{{ row.lastUsedAt ? row.lastUsedAt.toLocaleDateString() : 'Never' }}</span>
      </template>
      <template #cell-expiresAt="{ row }">
        <span class="text-text-secondary">{{ row.expiresAt ? row.expiresAt.toLocaleDateString() : 'Never' }}</span>
      </template>
      <template #cell-actions="{ row }">
        <button
          class="p-1 text-text-muted hover:text-danger transition-colors"
          title="Revoke key" @click="confirmRevoke = { open: true, id: row.id, name: row.name }"
        >
          <Trash2 class="w-4 h-4" />
        </button>
      </template>
    </STable>

    <SConfirmDialog
      :open="confirmRevoke.open"
      title="Revoke API key"
      :message="`Revoke &quot;${confirmRevoke.name}&quot;? This cannot be undone.`"
      confirm-text="Revoke"
      @confirm="handleRevoke"
      @cancel="confirmRevoke.open = false"
    />
  </div>
</template>
