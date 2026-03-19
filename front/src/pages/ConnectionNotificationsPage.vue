<script setup lang="ts">
import type { ChannelType, NotificationCondition } from '@entities/notification'
import { useCreateRule, useDeleteRule, useNotificationChannels, useNotificationRules, useUpdateRule } from '@entities/notification'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SBadge, useToast } from '@shared/ui'
import { ChevronDown, Hash, Mail, Send } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const id = route.params.id as string
const toast = useToast()

const connectionIdRef = ref(id)
const { data: notifChannels } = useNotificationChannels()
const { data: notifRules } = useNotificationRules(connectionIdRef)
const createRuleMutation = useCreateRule()
const updateRuleMutation = useUpdateRule()
const deleteRuleMutation = useDeleteRule()

const expandedChannels = ref<Set<string>>(new Set())

function toggleExpand(channelId: string) {
  if (expandedChannels.value.has(channelId)) {
    expandedChannels.value.delete(channelId)
  }
  else {
    expandedChannels.value.add(channelId)
  }
}

const channelTypeIcons: Record<ChannelType, typeof Hash> = {
  slack: Hash,
  email: Mail,
  telegram: Send,
}

const conditionOptions: { value: NotificationCondition, label: string, description: string, hasValue: boolean }[] = [
  { value: 'on_failure', label: 'On failure', description: 'Notify when a sync fails', hasValue: false },
  { value: 'on_consecutive_failures', label: 'On consecutive failures', description: 'Notify after N consecutive failures', hasValue: true },
  { value: 'on_zero_records', label: 'On zero records', description: 'Notify when a sync produces no records', hasValue: false },
]

function getChannelRules(channelId: string) {
  return notifRules.value?.filter(r => r.channelId === channelId) ?? []
}

function hasCondition(channelId: string, condition: NotificationCondition): boolean {
  return getChannelRules(channelId).some(r => r.condition === condition)
}

function getRuleForCondition(channelId: string, condition: NotificationCondition) {
  return getChannelRules(channelId).find(r => r.condition === condition)
}

function getActiveConditionsSummary(channelId: string): string {
  const rules = getChannelRules(channelId)
  if (rules.length === 0)
    return 'No rules'
  return rules.map((r) => {
    const opt = conditionOptions.find(o => o.value === r.condition)
    if (r.condition === 'on_consecutive_failures')
      return `${opt?.label} (${r.conditionValue})`
    return opt?.label ?? r.condition
  }).join(', ')
}

async function toggleCondition(channelId: string, condition: NotificationCondition) {
  try {
    const existing = getRuleForCondition(channelId, condition)
    if (existing) {
      await deleteRuleMutation.mutateAsync(existing.id)
    }
    else {
      await createRuleMutation.mutateAsync({
        channelId,
        connectionId: id,
        condition,
        conditionValue: condition === 'on_consecutive_failures' ? 3 : 1,
        enabled: true,
      })
    }
  }
  catch (e: unknown) {
    toast.error(`Error: ${getErrorMessage(e)}`)
  }
}

async function updateConditionValue(channelId: string, condition: NotificationCondition, value: number) {
  const existing = getRuleForCondition(channelId, condition)
  if (existing && value >= 1) {
    try {
      await updateRuleMutation.mutateAsync({
        id: existing.id,
        condition: existing.condition,
        conditionValue: value,
        enabled: existing.enabled,
      })
    }
    catch (e: unknown) {
      toast.error(`Error: ${getErrorMessage(e)}`)
    }
  }
}
</script>

<template>
  <div v-if="!notifChannels || notifChannels.length === 0" class="text-center py-12">
    <p class="text-sm text-text-muted mb-2">
      No notification channels configured.
    </p>
    <router-link
      :to="{ name: 'settings-notifications' }"
      class="text-sm text-primary hover:underline"
    >
      Go to Settings &gt; Notifications to add channels.
    </router-link>
  </div>

  <div v-else class="space-y-2">
    <p class="text-sm text-text-secondary mb-4">
      Configure which notification channels and conditions trigger alerts for this connection.
    </p>

    <div
      v-for="channel in notifChannels"
      :key="channel.id"
      class="border border-border rounded-lg overflow-hidden"
    >
      <!-- Collapsed header — click to expand -->
      <button
        class="w-full flex items-center justify-between px-4 py-3 hover:bg-surface-raised/50 transition-colors"
        @click="toggleExpand(channel.id)"
      >
        <div class="flex items-center gap-3">
          <component :is="channelTypeIcons[channel.channelType]" class="w-4 h-4 text-text-secondary" />
          <span class="text-sm font-medium text-heading">{{ channel.name }}</span>
          <SBadge variant="gray" class="text-[10px]">
            {{ channel.channelType }}
          </SBadge>
          <SBadge v-if="getChannelRules(channel.id).length > 0" variant="info" class="text-[10px]">
            {{ getChannelRules(channel.id).length }} active
          </SBadge>
        </div>
        <div class="flex items-center gap-3">
          <span v-if="!expandedChannels.has(channel.id)" class="text-xs text-text-muted hidden sm:inline">
            {{ getActiveConditionsSummary(channel.id) }}
          </span>
          <ChevronDown
            class="w-4 h-4 text-text-muted transition-transform"
            :class="{ 'rotate-180': expandedChannels.has(channel.id) }"
          />
        </div>
      </button>

      <!-- Expanded conditions -->
      <div v-if="expandedChannels.has(channel.id)" class="border-t border-border">
        <div
          v-for="opt in conditionOptions"
          :key="opt.value"
          class="flex items-center justify-between px-4 py-3 border-b border-border last:border-b-0"
        >
          <label class="flex items-center gap-2.5 cursor-pointer">
            <input
              type="checkbox"
              :checked="hasCondition(channel.id, opt.value)"
              class="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20"
              @change="toggleCondition(channel.id, opt.value)"
            >
            <div>
              <span class="text-sm font-medium text-text-primary">{{ opt.label }}</span>
              <p class="text-xs text-text-muted">{{ opt.description }}</p>
            </div>
          </label>
          <div v-if="opt.hasValue && hasCondition(channel.id, opt.value)" class="flex items-center gap-2">
            <span class="text-xs text-text-muted">after</span>
            <input
              type="number"
              min="1"
              max="100"
              :value="getRuleForCondition(channel.id, opt.value)?.conditionValue ?? 3"
              class="w-16 px-2 py-1 text-sm rounded border border-border bg-surface text-text-primary focus:ring-1 focus:ring-primary/20 focus:border-primary"
              @change="updateConditionValue(channel.id, opt.value, Number(($event.target as HTMLInputElement).value))"
            >
            <span class="text-xs text-text-muted">failures</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
