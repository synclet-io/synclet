<script setup lang="ts">
import type { Webhook, WebhookEvent } from '@entities/webhook'
import type { Column } from '@shared/ui'
import { useAuth } from '@entities/auth'
import {
  useCreateWebhook,
  useDeleteWebhook,
  useTestWebhook,
  useUpdateWebhook,
  useWebhooks,
  WEBHOOK_EVENTS,
} from '@entities/webhook'
import { getErrorMessage } from '@shared/lib/errorUtils'
import {
  SAlert,
  SBadge,
  SButton,
  SConfirmDialog,
  SEmptyState,
  SInput,
  SModal,
  STable,
  useToast,
} from '@shared/ui'
import { Pencil, Send, Trash2, Webhook as WebhookIcon } from 'lucide-vue-next'
import { computed, reactive, ref } from 'vue'

const LOCAL_URL_RE = /^http:\/\/(?:localhost|127\.\d{1,3}\.\d{1,3}\.\d{1,3})(?::\d+)?(?:$|\/)/

const auth = useAuth()
const toast = useToast()

const workspaceId = computed(() => auth.currentWorkspaceId.value ?? '')
const currentRole = computed(() => {
  const ws = auth.workspaces.value.find(w => w.workspaceId === workspaceId.value)
  return ws?.role ?? 'viewer'
})
const isAdmin = computed(() => currentRole.value === 'admin')

const { data: webhooks, isLoading, error: loadError } = useWebhooks()
const createMutation = useCreateWebhook()
const updateMutation = useUpdateWebhook()
const deleteMutation = useDeleteWebhook()
const testMutation = useTestWebhook()

const error = ref('')

interface WebhookForm {
  url: string
  events: WebhookEvent[]
  secret: string
  enabled: boolean
}

function newForm(): WebhookForm {
  return { url: '', events: [], secret: '', enabled: true }
}

const dialog = reactive<{
  open: boolean
  mode: 'create' | 'edit'
  webhookId: string
  form: WebhookForm
}>({ open: false, mode: 'create', webhookId: '', form: newForm() })

const confirmDelete = ref<{ open: boolean, id: string, url: string }>({ open: false, id: '', url: '' })

const columns: Column[] = [
  { key: 'url', label: 'URL' },
  { key: 'events', label: 'Events' },
  { key: 'status', label: 'Status' },
  { key: 'createdAt', label: 'Created' },
  { key: 'actions', label: '', align: 'right', width: '140px' },
]

function openCreate() {
  dialog.mode = 'create'
  dialog.webhookId = ''
  dialog.form = newForm()
  dialog.open = true
  error.value = ''
}

function openEdit(webhook: Webhook) {
  dialog.mode = 'edit'
  dialog.webhookId = webhook.id
  dialog.form = {
    url: webhook.url,
    events: [...webhook.events],
    secret: '',
    enabled: webhook.enabled,
  }
  dialog.open = true
  error.value = ''
}

function toggleEvent(value: WebhookEvent) {
  const idx = dialog.form.events.indexOf(value)
  if (idx >= 0)
    dialog.form.events.splice(idx, 1)
  else
    dialog.form.events.push(value)
}

function validateForm(): string | null {
  const url = dialog.form.url.trim()
  if (!url)
    return 'URL is required'

  const httpsOk = url.startsWith('https://')
  const localhostOk = LOCAL_URL_RE.test(url)
  if (!httpsOk && !localhostOk)
    return 'URL must use HTTPS (HTTP allowed only for localhost / 127.x.x.x)'

  if (dialog.form.events.length === 0)
    return 'Select at least one event'

  return null
}

async function submitForm() {
  const validationError = validateForm()
  if (validationError) {
    error.value = validationError
    return
  }

  try {
    if (dialog.mode === 'create') {
      await createMutation.mutateAsync({
        url: dialog.form.url.trim(),
        events: dialog.form.events,
        secret: dialog.form.secret || undefined,
      })
      toast.success('Webhook created')
    }
    else {
      await updateMutation.mutateAsync({
        id: dialog.webhookId,
        url: dialog.form.url.trim(),
        events: dialog.form.events,
        enabled: dialog.form.enabled,
      })
      toast.success('Webhook updated')
    }
    dialog.open = false
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to save webhook'
  }
}

async function handleDelete() {
  const id = confirmDelete.value.id
  confirmDelete.value.open = false
  try {
    await deleteMutation.mutateAsync(id)
    toast.success('Webhook deleted')
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to delete webhook'
  }
}

async function handleTest(webhook: Webhook) {
  try {
    const result = await testMutation.mutateAsync(webhook.id)
    if (result.deliveryError)
      toast.error(`Test failed (${result.statusCode || 'no response'}): ${result.deliveryError}`)
    else
      toast.success(`Test delivered: HTTP ${result.statusCode}`)
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to test webhook'
  }
}

async function toggleEnabled(webhook: Webhook) {
  try {
    await updateMutation.mutateAsync({
      id: webhook.id,
      enabled: !webhook.enabled,
    })
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to update webhook'
  }
}

function eventLabel(value: WebhookEvent): string {
  return WEBHOOK_EVENTS.find(e => e.value === value)?.label ?? value
}
</script>

<template>
  <div class="max-w-3xl space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="text-lg font-semibold text-heading">
        Webhooks
      </h2>
      <SButton v-if="isAdmin && (webhooks?.length ?? 0) > 0" @click="openCreate">
        Create webhook
      </SButton>
    </div>

    <SAlert v-if="!isAdmin" variant="info">
      You need admin permissions to manage webhooks.
    </SAlert>

    <SAlert v-if="error" variant="danger" dismissible @dismiss="error = ''">
      {{ error }}
    </SAlert>
    <SAlert v-if="loadError" variant="danger">
      Failed to load webhooks.
    </SAlert>

    <!-- Empty state when admin and no webhooks -->
    <template v-if="isAdmin && !isLoading && (webhooks?.length ?? 0) === 0">
      <SEmptyState
        title="No webhooks yet"
        description="Forward sync and schema events to your tooling. Webhooks deliver a signed POST when configured events fire."
        :icon="WebhookIcon"
      >
        <template #actions>
          <SButton @click="openCreate">
            Create webhook
          </SButton>
        </template>
      </SEmptyState>
    </template>

    <STable
      v-else-if="isAdmin"
      :columns="columns"
      :data="webhooks ?? []"
      :loading="isLoading"
    >
      <template #cell-url="{ row }">
        <span class="font-mono text-xs text-text-primary break-all">{{ row.url }}</span>
      </template>
      <template #cell-events="{ row }">
        <div class="flex flex-wrap gap-1">
          <SBadge v-for="event in row.events" :key="event" variant="gray">
            {{ eventLabel(event) }}
          </SBadge>
        </div>
      </template>
      <template #cell-status="{ row }">
        <button
          class="cursor-pointer"
          :title="row.enabled ? 'Click to disable' : 'Click to enable'"
          @click="toggleEnabled(row)"
        >
          <SBadge :variant="row.enabled ? 'success' : 'gray'" dot>
            {{ row.enabled ? 'Enabled' : 'Disabled' }}
          </SBadge>
        </button>
      </template>
      <template #cell-createdAt="{ row }">
        <span class="text-sm text-text-secondary">{{ row.createdAt?.toLocaleDateString() ?? '-' }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="flex items-center justify-end gap-1">
          <button
            class="p-1 text-text-muted hover:text-text-primary transition-colors"
            title="Test delivery"
            :disabled="testMutation.isPending.value"
            @click="handleTest(row)"
          >
            <Send class="w-4 h-4" />
          </button>
          <button
            class="p-1 text-text-muted hover:text-text-primary transition-colors"
            title="Edit"
            @click="openEdit(row)"
          >
            <Pencil class="w-4 h-4" />
          </button>
          <button
            class="p-1 text-text-muted hover:text-danger transition-colors"
            title="Delete"
            @click="confirmDelete = { open: true, id: row.id, url: row.url }"
          >
            <Trash2 class="w-4 h-4" />
          </button>
        </div>
      </template>
    </STable>

    <!-- Create / Edit dialog -->
    <SModal :open="dialog.open" :title="dialog.mode === 'create' ? 'Create webhook' : 'Edit webhook'" size="lg" @close="dialog.open = false">
      <form class="space-y-4" @submit.prevent="submitForm">
        <div>
          <label class="block text-sm font-medium text-text-primary mb-1">URL</label>
          <SInput v-model="dialog.form.url" type="url" placeholder="https://example.com/webhook" required />
          <p class="mt-1 text-xs text-text-muted">
            Must be HTTPS. HTTP is only allowed for localhost / 127.x.x.x.
          </p>
        </div>

        <div>
          <span class="block text-sm font-medium text-text-primary mb-1">Events</span>
          <div class="space-y-2">
            <label v-for="event in WEBHOOK_EVENTS" :key="event.value" class="flex items-center gap-2 text-sm text-text-primary">
              <input
                type="checkbox"
                :checked="dialog.form.events.includes(event.value)"
                class="rounded border-border"
                @change="toggleEvent(event.value)"
              >
              {{ event.label }}
            </label>
          </div>
        </div>

        <div v-if="dialog.mode === 'create'">
          <label class="block text-sm font-medium text-text-primary mb-1">Secret (optional)</label>
          <SInput v-model="dialog.form.secret" type="password" placeholder="Used to sign deliveries with HMAC-SHA256" />
          <p class="mt-1 text-xs text-text-muted">
            Secret cannot be changed after creation.
          </p>
        </div>

        <div v-else class="flex items-center gap-2">
          <input id="enabled-toggle" v-model="dialog.form.enabled" type="checkbox" class="rounded border-border">
          <label for="enabled-toggle" class="text-sm text-text-primary">Enabled</label>
        </div>
      </form>

      <template #footer>
        <SButton variant="ghost" @click="dialog.open = false">
          Cancel
        </SButton>
        <SButton :loading="createMutation.isPending.value || updateMutation.isPending.value" @click="submitForm">
          {{ dialog.mode === 'create' ? 'Create' : 'Save' }}
        </SButton>
      </template>
    </SModal>

    <SConfirmDialog
      :open="confirmDelete.open"
      title="Delete webhook"
      :message="`Delete webhook for ${confirmDelete.url}? Deliveries will stop immediately.`"
      confirm-text="Delete"
      @confirm="handleDelete"
      @cancel="confirmDelete = { open: false, id: '', url: '' }"
    />
  </div>
</template>
