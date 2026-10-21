<script setup lang="ts">
import type { Column } from '@shared/ui'
import { useAuth } from '@entities/auth'
import { useAPIKeys, useCreateAPIKey, useRevokeAPIKey } from '@entities/auth/apiKeys'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SButton, SConfirmDialog, SInput, SSelect, STable, useToast } from '@shared/ui'
import { Check, Copy, Trash2 } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

const auth = useAuth()
const toast = useToast()

const { data: keys, isLoading, error: loadError } = useAPIKeys()
const createMutation = useCreateAPIKey()
const revokeMutation = useRevokeAPIKey()

const error = ref('')
const newKeyName = ref('')
const expiry = ref<string | number>('never')
const customExpiry = ref('')

const expiryOptions = [
  { label: 'Never expires', value: 'never' },
  { label: '30 days', value: 30 },
  { label: '90 days', value: 90 },
  { label: '1 year', value: 365 },
  { label: 'Custom date', value: 'custom' },
]

const minCustomExpiry = computed(() => {
  const tomorrow = new Date()
  tomorrow.setDate(tomorrow.getDate() + 1)
  return toDateInputValue(tomorrow)
})

function toDateInputValue(date: Date): string {
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${date.getFullYear()}-${month}-${day}`
}

function expiresAtFromSelection(): Date | undefined {
  if (typeof expiry.value === 'number') {
    const date = new Date()
    date.setDate(date.getDate() + expiry.value)
    return date
  }
  if (expiry.value === 'custom' && customExpiry.value) {
    const [year, month, day] = customExpiry.value.split('-').map(Number)
    return new Date(year, month - 1, day, 23, 59, 59)
  }
  return undefined
}
const createdRawKey = ref<string | null>(null)
const copied = ref(false)
const confirmRevoke = ref<{ open: boolean, id: string, name: string }>({ open: false, id: '', name: '' })

watch(auth.currentWorkspaceId, () => {
  createdRawKey.value = null
  error.value = ''
})

watch(loadError, (e) => {
  if (e)
    error.value = getErrorMessage(e) || 'Failed to load API keys'
})

async function handleCreate() {
  if (!newKeyName.value)
    return
  error.value = ''
  createdRawKey.value = null
  try {
    const result = await createMutation.mutateAsync({ name: newKeyName.value, expiresAt: expiresAtFromSelection() })
    createdRawKey.value = result.rawKey
    newKeyName.value = ''
    expiry.value = 'never'
    customExpiry.value = ''
    toast.success('API key created')
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to create API key'
  }
}

async function handleRevoke() {
  const id = confirmRevoke.value.id
  confirmRevoke.value.open = false
  try {
    await revokeMutation.mutateAsync(id)
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
  <div class="space-y-6">
    <PageHeader title="API Keys" :level="2" />
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
        <SButton variant="ghost" size="icon" title="Copy" :aria-label="copied ? 'Copied' : 'Copy API key'" @click="copyKey">
          <Check v-if="copied" class="w-4 h-4 text-success" />
          <Copy v-else class="w-4 h-4" />
        </SButton>
      </div>
    </SAlert>

    <form class="flex flex-wrap gap-3" @submit.prevent="handleCreate">
      <div class="flex-1 min-w-48">
        <SInput v-model="newKeyName" placeholder="Key name (e.g. CI/CD)" required />
      </div>
      <SSelect v-model="expiry" :options="expiryOptions" aria-label="Expiration" class="w-44" />
      <SInput
        v-if="expiry === 'custom'"
        v-model="customExpiry"
        type="date"
        :min="minCustomExpiry"
        aria-label="Expiration date"
        required
        class="w-44"
      />
      <SButton type="submit" :loading="createMutation.isPending.value">
        Create key
      </SButton>
    </form>

    <STable :columns="columns" :data="keys ?? []" :loading="isLoading" empty-text="No API keys yet.">
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
        <SButton
          variant="ghost"
          size="icon"
          title="Revoke key"
          :aria-label="`Revoke ${row.name}`"
          @click="confirmRevoke = { open: true, id: row.id, name: row.name }"
        >
          <Trash2 class="w-4 h-4" />
        </SButton>
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
