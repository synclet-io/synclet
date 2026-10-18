<script lang="ts">
import type { Tab } from '@shared/ui'
import type { ComputedRef, InjectionKey } from 'vue'
import { useAuth } from '@entities/auth'
import { PageHeader, STabs } from '@shared/ui'
import { computed, provide } from 'vue'
import { RouterView } from 'vue-router'

export const workspaceAdminKey: InjectionKey<ComputedRef<boolean>> = Symbol('workspaceAdmin')
</script>

<script setup lang="ts">
const auth = useAuth()
const isAdmin = computed(() => {
  const ws = auth.workspaces.value.find(w => w.workspaceId === auth.currentWorkspaceId.value)
  return ws?.role === 'admin'
})
provide(workspaceAdminKey, isAdmin)

const tabs = computed<Tab[]>(() => {
  const allTabs: Tab[] = [
    { name: 'General', to: { name: 'settings-general' } },
    { name: 'Members', to: { name: 'settings-members' } },
    { name: 'Connectors', to: { name: 'settings-connectors' } },
    { name: 'Notifications', to: { name: 'settings-notifications' } },
    { name: 'Webhooks', to: { name: 'settings-webhooks' } },
    { name: 'Audit log', to: { name: 'settings-audit' } },
    { name: 'System', to: { name: 'settings-system' } },
    { name: 'API Keys', to: { name: 'settings-api-keys' } },
    { name: 'Account', to: { name: 'settings-account' } },
  ]
  let visible = allTabs
  if (!isAdmin.value)
    visible = visible.filter(t => t.name !== 'Webhooks' && t.name !== 'Audit log')
  return visible
})
</script>

<template>
  <div>
    <PageHeader title="Settings" />
    <STabs :tabs="tabs" />
    <RouterView />
  </div>
</template>
