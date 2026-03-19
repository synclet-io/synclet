<script setup lang="ts">
import { useConnectorVersions, useManagedConnector } from '@entities/connector'
import { useDestination, useUpdateDestination } from '@entities/destination'
import RuntimeConfigForm from '@features/runtime-config/RuntimeConfigForm.vue'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SBadge, SButton, SCard, SSkeleton, useToast } from '@shared/ui'
import { Pencil } from 'lucide-vue-next'
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const destinationId = computed(() => route.params.id as string)
const { data: destination, isLoading, error } = useDestination(destinationId)
const updateDestination = useUpdateDestination()
const connectorId = computed(() => destination.value?.managedConnectorId ?? '')
const { data: connector } = useManagedConnector(connectorId)
const connectorImage = computed(() => connector.value?.dockerImage ?? '')
const connectorLabel = computed(() => connector.value ? `${connector.value.dockerImage}:${connector.value.dockerTag}` : '')
const { data: versionInfo } = useConnectorVersions(connectorImage)
const hasUpdate = computed(() => versionInfo.value?.latestVersion && connector.value?.dockerTag && versionInfo.value.latestVersion !== connector.value.dockerTag)
const toast = useToast()

async function handleSaveRuntimeConfig(configJson: string | null) {
  if (!destination.value)
    return
  try {
    await updateDestination.mutateAsync({
      id: destination.value.id,
      runtimeConfig: configJson,
    })
    toast.success('Runtime configuration saved')
  }
  catch (e: unknown) {
    toast.error(`Failed to save runtime configuration: ${getErrorMessage(e)}`)
  }
}

async function handleResetRuntimeConfig() {
  if (!destination.value)
    return
  try {
    await updateDestination.mutateAsync({
      id: destination.value.id,
      runtimeConfig: null,
    })
    toast.success('Runtime configuration reset to defaults')
  }
  catch (e: unknown) {
    toast.error(`Failed to reset runtime configuration: ${getErrorMessage(e)}`)
  }
}
</script>

<template>
  <div v-if="isLoading">
    <SSkeleton class="h-8 w-48 mb-4" />
    <SSkeleton class="h-64" />
  </div>
  <div v-else-if="error">
    <SAlert variant="danger">
      Failed to load destination: {{ error.message }}
    </SAlert>
  </div>
  <div v-else-if="destination">
    <PageHeader
      :title="destination.name"
      :description="connectorLabel"
      back-label="Destinations"
      :back-to="{ name: 'destinations' }"
    >
      <template #actions>
        <SButton variant="secondary" :to="`/destinations/${destination.id}/edit`">
          <Pencil class="w-4 h-4" /> Edit
        </SButton>
      </template>
    </PageHeader>

    <SCard class="mb-6">
      <div class="grid grid-cols-2 gap-4 text-sm p-6">
        <div>
          <span class="text-text-muted">Connector</span>
          <p class="text-text-primary font-medium">
            {{ connectorLabel }}
          </p>
          <div v-if="hasUpdate" class="mt-1 flex items-center gap-2">
            <SBadge variant="info">
              Update available
            </SBadge>
            <span class="text-xs text-primary">Latest: v{{ versionInfo!.latestVersion }}</span>
          </div>
        </div>
        <div>
          <span class="text-text-muted">Created</span>
          <p class="text-text-primary">
            {{ destination.createdAt?.toLocaleDateString() ?? '-' }}
          </p>
        </div>
      </div>
    </SCard>

    <RuntimeConfigForm
      :runtime-config="destination.runtimeConfig"
      entity-type="destination"
      :saving="updateDestination.isPending.value"
      @save="handleSaveRuntimeConfig"
      @reset="handleResetRuntimeConfig"
    />
  </div>
</template>
