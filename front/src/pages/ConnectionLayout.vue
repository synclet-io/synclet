<script setup lang="ts">
import type { Tab } from '@shared/ui'
import { useConnection, useDisableConnection, useEnableConnection } from '@entities/connection'
import { useCancelJob, useJobs, useTriggerSync } from '@entities/job'
import { useConnectionStats } from '@entities/stats'
import HealthBadge from '@features/connection-stats/HealthBadge.vue'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { statusVariant } from '@shared/lib/format'
import { PageHeader, SAlert, SBadge, SButton, SSkeleton, STabs, useToast } from '@shared/ui'
import { Pause, Play, RefreshCw, Settings, XCircle } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { RouterView, useRoute } from 'vue-router'

const route = useRoute()
const id = computed(() => route.params.id as string)
const toast = useToast()

const { data: connection, isLoading: loading, error } = useConnection(id)
const { data: jobs } = useJobs(id)
const { data: connStats } = useConnectionStats(id, ref('24h'))
const triggerSyncMutation = useTriggerSync()
const enableConnectionMutation = useEnableConnection()
const disableConnectionMutation = useDisableConnection()
const cancelJobMutation = useCancelJob()

const syncing = computed(() => triggerSyncMutation.isPending.value)
const cancelling = computed(() => cancelJobMutation.isPending.value)
const activeJob = computed(() => jobs.value?.items.find(j => j.status === 'running' || j.status === 'scheduled' || j.status === 'starting'))

const tabs = computed<Tab[]>(() => [
  { name: 'Overview', to: { name: 'connection-detail', params: { id: id.value } } },
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
  if (!activeJob.value)
    return
  try {
    await cancelJobMutation.mutateAsync(activeJob.value.id)
    toast.success('Sync cancelled')
  }
  catch (e: unknown) {
    toast.error(`Error: ${getErrorMessage(e)}`)
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
      {{ error.message }}
    </SAlert>
  </div>

  <template v-else-if="connection">
    <PageHeader :title="connection.name" description="Connection details and sync history" back-label="Connections" :back-to="{ name: 'connections' }">
      <template #actions>
        <HealthBadge v-if="connStats" :health="connStats.health" />
        <SBadge :variant="statusVariant(connection.status)" dot>
          {{ connection.status }}
        </SBadge>
        <SButton variant="secondary" @click="toggleEnabled">
          <component :is="connection.status === 'active' ? Pause : Play" class="w-4 h-4" />
          {{ connection.status === 'active' ? 'Disable' : 'Enable' }}
        </SButton>
        <SButton variant="secondary" :to="`/connections/${id}/streams`">
          <Settings class="w-4 h-4" /> Streams
        </SButton>
        <SButton v-if="activeJob" variant="danger" :loading="cancelling" @click="cancelSync">
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
    <RouterView />
  </template>
</template>
