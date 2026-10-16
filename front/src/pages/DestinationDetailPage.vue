<script setup lang="ts">
import { useConnectorVersions, useManagedConnector } from '@entities/connector'
import { useDeleteDestination, useDestination, useUpdateDestination } from '@entities/destination'
import DeleteInstanceDialog from '@features/connector-instance/DeleteInstanceDialog.vue'
import InstanceConnectionsCard from '@features/connector-instance/InstanceConnectionsCard.vue'
import RuntimeConfigForm from '@features/runtime-config/RuntimeConfigForm.vue'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SBadge, SButton, SCard, SSkeleton, useToast } from '@shared/ui'
import { Pencil, Trash2 } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const destinationId = computed(() => route.params.id as string)
const { data: destination, isLoading, error } = useDestination(destinationId)
const updateDestination = useUpdateDestination()
const deleteDestination = useDeleteDestination()
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

const deleteOpen = ref(false)
const deleteError = ref('')

function requestDelete() {
  deleteError.value = ''
  deleteOpen.value = true
}

async function handleDelete() {
  if (!destination.value)
    return
  deleteError.value = ''
  try {
    await deleteDestination.mutateAsync(destination.value.id)
  }
  catch (e: unknown) {
    deleteError.value = getErrorMessage(e) || 'Failed to delete destination'
    return
  }
  deleteOpen.value = false
  toast.success('Destination deleted')
  router.push({ name: 'destinations' })
}
</script>

<template>
  <div v-if="isLoading">
    <SSkeleton class="h-8 w-48 mb-4" />
    <SSkeleton class="h-64" />
  </div>
  <div v-else-if="error">
    <SAlert variant="danger">
      Failed to load destination: {{ getErrorMessage(error) }}
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
        <SButton variant="danger" @click="requestDelete">
          <Trash2 class="w-4 h-4" /> Delete
        </SButton>
      </template>
    </PageHeader>

    <SCard class="mb-6">
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
        <div>
          <span class="text-text-muted">Connector</span>
          <p class="text-text-primary font-medium">
            {{ connectorLabel }}
          </p>
          <RouterLink
            v-if="hasUpdate"
            :to="{ name: 'settings-connectors' }"
            class="mt-1 inline-flex items-center gap-2 hover:underline"
            title="Manage connector versions"
          >
            <SBadge variant="info">
              Update available
            </SBadge>
            <span class="text-xs text-primary">Latest: v{{ versionInfo!.latestVersion }}</span>
          </RouterLink>
        </div>
        <div>
          <span class="text-text-muted">Created</span>
          <p class="text-text-primary">
            {{ destination.createdAt?.toLocaleDateString() ?? '-' }}
          </p>
        </div>
      </div>
    </SCard>

    <InstanceConnectionsCard kind="destination" :instance-id="destination.id" />

    <RuntimeConfigForm
      :runtime-config="destination.runtimeConfig"
      entity-type="destination"
      :saving="updateDestination.isPending.value"
      @save="handleSaveRuntimeConfig"
      @reset="handleResetRuntimeConfig"
    />

    <DeleteInstanceDialog
      :open="deleteOpen"
      kind="destination"
      :name="destination.name"
      :loading="deleteDestination.isPending.value"
      :error="deleteError"
      @confirm="handleDelete"
      @cancel="deleteOpen = false"
    />
  </div>
</template>
