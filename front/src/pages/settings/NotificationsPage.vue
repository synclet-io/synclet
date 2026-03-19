<script setup lang="ts">
import type { ChannelType, NotificationChannel } from '@entities/notification'
import type { Column } from '@shared/ui'
import { useDeleteChannel, useNotificationChannels } from '@entities/notification'
import NotificationChannelForm from '@features/notification-channels/NotificationChannelForm.vue'
import { SBadge, SButton, SCard, SConfirmDialog, SEmptyState, SSkeleton, STable } from '@shared/ui'
import { Bell, Hash, Mail, Pencil, Plus, Send, Trash2 } from 'lucide-vue-next'
import { ref } from 'vue'

const { data: channels, isLoading } = useNotificationChannels()
const deleteMutation = useDeleteChannel()

const formOpen = ref(false)
const editingChannel = ref<NotificationChannel | undefined>()
const deleteDialogOpen = ref(false)
const deletingChannel = ref<NotificationChannel | undefined>()

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
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingChannel.value)
    return
  await deleteMutation.mutateAsync(deletingChannel.value.id)
  deleteDialogOpen.value = false
  deletingChannel.value = undefined
}
</script>

<template>
  <div class="mt-6">
    <SCard title="Notification Channels">
      <template #header>
        <SButton variant="primary" size="sm" @click="openCreate">
          <Plus class="w-4 h-4" />
          Add Channel
        </SButton>
      </template>

      <div v-if="isLoading" class="p-4">
        <SSkeleton variant="rect" height="200px" />
      </div>

      <template v-else-if="!channels || channels.length === 0">
        <SEmptyState
          :icon="Bell"
          title="No notification channels"
          description="Add a channel to receive alerts about sync failures."
        >
          <SButton variant="primary" size="sm" @click="openCreate">
            <Plus class="w-4 h-4" />
            Add Channel
          </SButton>
        </SEmptyState>
      </template>

      <template v-else>
        <STable :columns="columns" :data="channels" :padding="false">
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
              <button
                class="p-1.5 rounded-lg text-text-muted hover:text-heading hover:bg-surface-hover transition-colors"
                @click="openEdit(row)"
              >
                <Pencil class="w-4 h-4" />
              </button>
              <button
                class="p-1.5 rounded-lg text-text-muted hover:text-danger hover:bg-danger-bg transition-colors"
                @click="openDelete(row)"
              >
                <Trash2 class="w-4 h-4" />
              </button>
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
      @confirm="confirmDelete"
      @cancel="deleteDialogOpen = false"
    />
  </div>
</template>
