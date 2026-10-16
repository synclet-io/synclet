<script setup lang="ts">
import type { ChannelType, NotificationChannel } from '@entities/notification'
import type { Column } from '@shared/ui'
import { useDeleteChannel, useNotificationChannels } from '@entities/notification'
import NotificationChannelForm from '@features/notification-channels/NotificationChannelForm.vue'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SBadge, SButton, SCard, SConfirmDialog, SEmptyState, SSkeleton, STable } from '@shared/ui'
import { Bell, Hash, Mail, Pencil, Plus, Send, Trash2 } from 'lucide-vue-next'
import { computed, inject, ref } from 'vue'
import { workspaceAdminKey } from './SettingsLayout.vue'

const isAdmin = inject(workspaceAdminKey, computed(() => false))

const { data: channels, isLoading } = useNotificationChannels()
const deleteMutation = useDeleteChannel()

const formOpen = ref(false)
const editingChannel = ref<NotificationChannel | undefined>()
const deleteDialogOpen = ref(false)
const deletingChannel = ref<NotificationChannel | undefined>()
const deleteError = ref('')

const channelTypeIcons: Record<ChannelType, typeof Hash> = {
  slack: Hash,
  email: Mail,
  telegram: Send,
}

const channelTypeLabels: Record<ChannelType, string> = {
  slack: 'Slack',
  email: 'Email',
  telegram: 'Telegram',
}

const columns: Column[] = [
  { key: 'type', label: 'Type' },
  { key: 'name', label: 'Name' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: 'Actions', align: 'right' },
]
const visibleColumns = computed(() => isAdmin.value ? columns : columns.filter(c => c.key !== 'actions'))

function openCreate() {
  editingChannel.value = undefined
  formOpen.value = true
}

function openEdit(channel: NotificationChannel) {
  editingChannel.value = channel
  formOpen.value = true
}

function openDelete(channel: NotificationChannel) {
  deletingChannel.value = channel
  deleteError.value = ''
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingChannel.value)
    return
  deleteError.value = ''
  try {
    await deleteMutation.mutateAsync(deletingChannel.value.id)
  }
  catch (err: unknown) {
    deleteError.value = getErrorMessage(err) || 'Failed to delete channel'
    return
  }
  deleteDialogOpen.value = false
  deletingChannel.value = undefined
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Notifications" :level="2">
      <template v-if="isAdmin" #actions>
        <SButton @click="openCreate">
          <Plus class="w-4 h-4" />
          Add channel
        </SButton>
      </template>
    </PageHeader>
    <SAlert v-if="!isAdmin" variant="info">
      You need admin permissions to manage notification channels.
    </SAlert>
    <SCard title="Notification Channels">
      <SSkeleton v-if="isLoading" variant="rect" height="200px" />

      <template v-else-if="!channels || channels.length === 0">
        <SEmptyState
          :icon="Bell"
          title="No notification channels"
          description="Add a channel to receive alerts about sync failures."
        >
          <SButton v-if="isAdmin" @click="openCreate">
            <Plus class="w-4 h-4" />
            Add channel
          </SButton>
        </SEmptyState>
      </template>

      <template v-else>
        <STable :columns="visibleColumns" :data="channels" :padding="false">
          <template #cell-type="{ row }">
            <div class="flex items-center gap-2">
              <component :is="channelTypeIcons[row.channelType as ChannelType]" class="w-4 h-4 text-text-secondary" />
              <span class="text-sm text-heading">{{ channelTypeLabels[row.channelType as ChannelType] || row.channelType }}</span>
            </div>
          </template>
          <template #cell-name="{ row }">
            <span class="text-sm font-medium text-heading">{{ row.name }}</span>
          </template>
          <template #cell-status="{ row }">
            <SBadge :variant="row.enabled ? 'success' : 'gray'" dot>
              {{ row.enabled ? 'Enabled' : 'Disabled' }}
            </SBadge>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center justify-end gap-1">
              <SButton variant="ghost" size="icon" title="Edit channel" :aria-label="`Edit ${row.name}`" @click="openEdit(row)">
                <Pencil class="w-4 h-4" />
              </SButton>
              <SButton variant="ghost" size="icon" title="Delete channel" :aria-label="`Delete ${row.name}`" @click="openDelete(row)">
                <Trash2 class="w-4 h-4" />
              </SButton>
            </div>
          </template>
        </STable>
      </template>
    </SCard>

    <NotificationChannelForm
      :open="formOpen"
      :channel="editingChannel"
      @close="formOpen = false"
      @saved="formOpen = false"
    />

    <SConfirmDialog
      :open="deleteDialogOpen"
      title="Delete notification channel"
      message="Are you sure? This will stop all notifications through this channel."
      confirm-text="Delete Channel"
      variant="danger"
      :loading="deleteMutation.isPending.value"
      :error="deleteError"
      @confirm="confirmDelete"
      @cancel="deleteDialogOpen = false"
    />
  </div>
</template>
