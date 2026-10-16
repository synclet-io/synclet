<script setup lang="ts">
import type { HealthStatus } from '@entities/stats'
import type { Tab } from '@shared/ui'
import { useConnection, useDisableConnection, useEnableConnection } from '@entities/connection'
import { useActiveJob, useCancelJob, useTriggerSync } from '@entities/job'
import { useConnectionStats } from '@entities/stats'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { statusVariant } from '@shared/lib/format'
import { PageHeader, SAlert, SBadge, SButton, SConfirmDialog, SSkeleton, STabs, useToast } from '@shared/ui'
import { Copy, Pause, Play, RefreshCw, Settings, XCircle } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { RouterView, useRoute } from 'vue-router'

const route = useRoute()
const id = computed(() => route.params.id as string)
const toast = useToast()

const { data: connection, isLoading: loading, error } = useConnection(id)
const { data: activeJob } = useActiveJob(id)
const { data: connStats } = useConnectionStats(id, ref('24h'))
const triggerSyncMutation = useTriggerSync()
const enableConnectionMutation = useEnableConnection()
const disableConnectionMutation = useDisableConnection()
const cancelJobMutation = useCancelJob()

const healthBadge: Record<HealthStatus, { variant: 'success' | 'warning' | 'danger' | 'gray', label: string }> = {
  healthy: { variant: 'success', label: 'Healthy' },
  warning: { variant: 'warning', label: 'Warning' },
  failing: { variant: 'danger', label: 'Failing' },
  disabled: { variant: 'gray', label: 'Disabled' },
  unknown: { variant: 'gray', label: 'Unknown' },
}

const syncing = computed(() => triggerSyncMutation.isPending.value)
const cancelling = computed(() => cancelJobMutation.isPending.value)
const confirmCancelOpen = ref(false)

const tabs = computed<Tab[]>(() => [
  { name: 'Overview', to: { name: 'connection-detail', params: { id: id.value } } },
  { name: 'Schema changes', to: { name: 'connection-schema-changes', params: { id: id.value } } },
  { name: 'Settings', to: { name: 'connection-settings', params: { id: id.value } } },
  { name: 'Notifications', to: { name: 'connection-notifications', params: { id: id.value } } },
])

async function triggerSync() {
  try {
    await triggerSyncMutation.mutateAsync(id.value)
    toast.success('Sync triggered')
  }
  catch (e: unknown) {
    toast.error(`Error: ${getErrorMessage(e)}`)
  }
}

async function cancelSync() {
  if (!activeJob.value) {
    confirmCancelOpen.value = false
    return
  }
  try {
    await cancelJobMutation.mutateAsync(activeJob.value.id)
    toast.success('Sync cancelled')
  }
  catch (e: unknown) {
    toast.error(`Error: ${getErrorMessage(e)}`)
  }
  finally {
    confirmCancelOpen.value = false
  }
}

async function toggleEnabled() {
  if (!connection.value)
    return
  try {
    if (connection.value.status === 'active') {
      await disableConnectionMutation.mutateAsync(id.value)
      toast.success('Connection disabled')
    }
    else {
      await enableConnectionMutation.mutateAsync(id.value)
      toast.success('Connection enabled')
    }
  }
  catch (e: unknown) {
    toast.error(`Error: ${getErrorMessage(e)}`)
  }
}
</script>

<template>
  <div v-if="loading" class="space-y-4">
    <SSkeleton variant="rect" height="40px" width="300px" />
    <div class="grid grid-cols-3 gap-4">
      <SSkeleton v-for="i in 3" :key="i" variant="rect" height="80px" />
    </div>
  </div>
  <div v-else-if="error" class="p-4">
    <SAlert variant="danger">
      {{ getErrorMessage(error) }}
    </SAlert>
  </div>

  <template v-else-if="connection">
    <PageHeader :title="connection.name" description="Connection details and sync history" back-label="Connections" :back-to="{ name: 'connections' }">
      <template #actions>
        <SBadge v-if="connStats" :variant="healthBadge[connStats.health].variant" dot>
          {{ healthBadge[connStats.health].label }}
        </SBadge>
        <SBadge :variant="statusVariant(connection.status)" dot>
          {{ connection.status }}
        </SBadge>
        <SBadge v-if="activeJob" :variant="statusVariant(activeJob.status)" dot>
          Sync {{ activeJob.status }}
        </SBadge>
        <SButton variant="secondary" @click="toggleEnabled">
          <component :is="connection.status === 'active' ? Pause : Play" class="w-4 h-4" />
          {{ connection.status === 'active' ? 'Disable' : 'Enable' }}
        </SButton>
        <SButton variant="secondary" :to="`/connections/${id}/streams`">
          <Settings class="w-4 h-4" /> Streams
        </SButton>
        <SButton variant="secondary" :to="`/connections/new?cloneFrom=${id}`" title="Duplicate this connection">
          <Copy class="w-4 h-4" /> Duplicate
        </SButton>
        <SButton v-if="activeJob" variant="danger" :loading="cancelling" @click="confirmCancelOpen = true">
          <XCircle class="w-4 h-4" />
          Cancel Sync
        </SButton>
        <SButton v-else :loading="syncing" @click="triggerSync">
          <RefreshCw class="w-4 h-4" :class="{ 'animate-spin': syncing }" />
          Sync Now
        </SButton>
      </template>
    </PageHeader>

    <STabs :tabs="tabs" />
    <RouterView :key="id" />

    <SConfirmDialog
      :open="confirmCancelOpen"
      title="Cancel sync"
      :message="`Cancel the running sync of ${connection.name}${activeJob ? ` (job ${activeJob.id})` : ''}?`"
      confirm-text="Cancel Sync"
      cancel-text="Keep Running"
      :loading="cancelling"
      @confirm="cancelSync"
      @cancel="confirmCancelOpen = false"
    />
  </template>
</template>
