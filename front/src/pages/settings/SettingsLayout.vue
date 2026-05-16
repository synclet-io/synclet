<script setup lang="ts">
import type { Tab } from '@shared/ui'
import { useAuth } from '@entities/auth'
import { useSystemInfo } from '@entities/system'
import { STabs } from '@shared/ui'
import { computed } from 'vue'
import { RouterView } from 'vue-router'

const { data: systemInfo } = useSystemInfo()
const isSingleWorkspace = computed(() => systemInfo.value?.workspacesMode === 'single')

const auth = useAuth()
const isAdmin = computed(() => {
  const ws = auth.workspaces.value.find(w => w.workspaceId === auth.currentWorkspaceId.value)
  return ws?.role === 'admin'
})

const tabs = computed<Tab[]>(() => {
  const allTabs: Tab[] = [
    { name: 'General', to: { name: 'settings-general' } },
    { name: 'Members', to: { name: 'settings-members' } },
    { name: 'Connectors', to: { name: 'settings-connectors' } },
    { name: 'Notifications', to: { name: 'settings-notifications' } },
    { name: 'Webhooks', to: { name: 'settings-webhooks' } },
    { name: 'API Keys', to: { name: 'settings-api-keys' } },
    { name: 'Account', to: { name: 'settings-account' } },
  ]
  let visible = allTabs
  if (isSingleWorkspace.value)
    visible = visible.filter(t => t.name !== 'Members')
  if (!isAdmin.value)
    visible = visible.filter(t => t.name !== 'Webhooks')
  return visible
})
</script>

<template>
  <div>
    <h1 class="text-2xl font-bold text-heading mb-6">
      Settings
    </h1>
    <STabs :tabs="tabs" />
    <RouterView />
  </div>
</template>
