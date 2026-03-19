<script setup lang="ts">
import type { Tab } from '@shared/ui'
import { useSystemInfo } from '@entities/system'
import { STabs } from '@shared/ui'
import { computed } from 'vue'
import { RouterView } from 'vue-router'

const { data: systemInfo } = useSystemInfo()
const isSingleWorkspace = computed(() => systemInfo.value?.workspacesMode === 'single')

const tabs = computed<Tab[]>(() => {
  const allTabs: Tab[] = [
    { name: 'General', to: { name: 'settings-general' } },
    { name: 'Members', to: { name: 'settings-members' } },
    { name: 'Connectors', to: { name: 'settings-connectors' } },
    { name: 'Notifications', to: { name: 'settings-notifications' } },
    { name: 'API Keys', to: { name: 'settings-api-keys' } },
    { name: 'Account', to: { name: 'settings-account' } },
  ]
  if (isSingleWorkspace.value) {
    return allTabs.filter(t => t.name !== 'Members')
  }
  return allTabs
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
