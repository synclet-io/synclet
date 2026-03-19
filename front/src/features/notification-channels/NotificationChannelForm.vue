<script setup lang="ts">
import type { ChannelType, NotificationChannel } from '@entities/notification'
import { useCreateChannel, useTestChannel, useUpdateChannel } from '@entities/notification'
import SAlert from '@shared/ui/SAlert.vue'
import SButton from '@shared/ui/SButton.vue'
import SInput from '@shared/ui/SInput.vue'
import SModal from '@shared/ui/SModal.vue'
import { useToast } from '@shared/ui/useToast'
import { Hash, Mail, Send } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  channel?: NotificationChannel
  open: boolean
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const toast = useToast()
const createMutation = useCreateChannel()
const updateMutation = useUpdateChannel()
const testMutation = useTestChannel()

const channelType = ref<ChannelType>('slack')
const name = ref('')
const enabled = ref(true)
const testError = ref('')

// Type-specific config
const slackWebhookUrl = ref('')
const emailRecipients = ref('')
const telegramBotToken = ref('')
const telegramChatId = ref('')

const isEdit = computed(() => !!props.channel)

const channelTypes: { type: ChannelType, label: string, icon: typeof Hash }[] = [
  { type: 'slack', label: 'Slack', icon: Hash },
  { type: 'email', label: 'Email', icon: Mail },
  { type: 'telegram', label: 'Telegram', icon: Send },
]

watch(() => props.open, (val) => {
  if (val && props.channel) {
    channelType.value = props.channel.channelType
    name.value = props.channel.name
    enabled.value = props.channel.enabled
    testError.value = ''
    const cfg = props.channel.config
    if (props.channel.channelType === 'slack') {
      slackWebhookUrl.value = cfg.webhook_url || ''
    }
    else if (props.channel.channelType === 'email') {
      emailRecipients.value = cfg.recipients || ''
    }
    else if (props.channel.channelType === 'telegram') {
      telegramBotToken.value = cfg.bot_token || ''
      telegramChatId.value = cfg.chat_id || ''
    }
  }
  else if (val) {
    channelType.value = 'slack'
    name.value = ''
    enabled.value = true
    testError.value = ''
    slackWebhookUrl.value = ''
    emailRecipients.value = ''
    telegramBotToken.value = ''
    telegramChatId.value = ''
  }
})

function getConfig(): Record<string, string> {
  switch (channelType.value) {
    case 'slack':
      return { webhook_url: slackWebhookUrl.value }
    case 'email':
      return { recipients: emailRecipients.value }
    case 'telegram':
      return { bot_token: telegramBotToken.value, chat_id: telegramChatId.value }
  }
}

const saving = computed(() => createMutation.isPending.value || updateMutation.isPending.value)

async function handleSave() {
  testError.value = ''
  const config = getConfig()
  if (isEdit.value && props.channel) {
    await updateMutation.mutateAsync({
      id: props.channel.id,
      name: name.value,
      config,
      enabled: enabled.value,
    })
  }
  else {
    await createMutation.mutateAsync({
      name: name.value,
      channelType: channelType.value,
      config,
      enabled: enabled.value,
    })
  }
  emit('saved')
  emit('close')
}

async function handleTest() {
  testError.value = ''
  if (!isEdit.value || !props.channel)
    return
  try {
    await testMutation.mutateAsync(props.channel.id)
    toast.success('Test notification sent successfully')
  }
  catch (err: unknown) {
    testError.value = err instanceof Error ? err.message : 'Failed to send test notification'
  }
}
</script>

<template>
  <SModal :open="open" :title="isEdit ? 'Edit Channel' : 'Add Notification Channel'" size="md" @close="emit('close')">
    <!-- Channel type selector (only for create) -->
    <div v-if="!isEdit" class="mb-6">
      <label class="block text-sm font-medium text-heading mb-3">Channel Type</label>
      <div class="grid grid-cols-3 gap-3">
        <button
          v-for="ct in channelTypes"
          :key="ct.type"
          type="button"
          class="flex flex-col items-center gap-2 p-4 border rounded-xl transition-all"
          :class="channelType === ct.type ? 'ring-2 ring-primary border-primary bg-primary/5' : 'border-border hover:border-primary/50'"
          @click="channelType = ct.type"
        >
          <component :is="ct.icon" class="w-6 h-6" :class="channelType === ct.type ? 'text-primary' : 'text-text-secondary'" />
          <span class="text-sm font-medium" :class="channelType === ct.type ? 'text-primary' : 'text-heading'">{{ ct.label }}</span>
        </button>
      </div>
    </div>

    <!-- Common fields -->
    <div class="space-y-4">
      <SInput v-model="name" label="Name" placeholder="My notification channel" required />

      <!-- Type-specific config -->
      <template v-if="channelType === 'slack'">
        <SInput
          v-model="slackWebhookUrl"
          label="Webhook URL"
          placeholder="https://hooks.slack.com/services/..."
          required
        />
      </template>

      <template v-if="channelType === 'email'">
        <SInput
          v-model="emailRecipients"
          label="Recipients"
          placeholder="admin@company.com, alerts@company.com"
          required
        />
      </template>

      <template v-if="channelType === 'telegram'">
        <SInput
          v-model="telegramBotToken"
          label="Bot Token"
          placeholder="123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
          required
        />
        <SInput
          v-model="telegramChatId"
          label="Chat ID"
          placeholder="-1001234567890"
          required
        />
      </template>

      <!-- Enabled checkbox -->
      <label class="flex items-center gap-2 cursor-pointer">
        <input v-model="enabled" type="checkbox" class="w-4 h-4 rounded border-border text-primary focus:ring-primary/20">
        <span class="text-sm text-heading">Enabled</span>
      </label>

      <!-- Test error -->
      <SAlert v-if="testError" variant="danger" dismissible @dismiss="testError = ''">
        {{ testError }}
      </SAlert>
    </div>

    <template #footer>
      <SButton
        v-if="isEdit"
        variant="secondary"
        :loading="testMutation.isPending.value"
        @click="handleTest"
      >
        Send Test
      </SButton>
      <SButton
        variant="primary"
        :loading="saving"
        @click="handleSave"
      >
        {{ isEdit ? 'Save Changes' : 'Save Channel' }}
      </SButton>
    </template>
  </SModal>
</template>
