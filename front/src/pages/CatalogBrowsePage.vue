<script setup lang="ts">
import type { Connector, ConnectorType, ReleaseStage } from '@entities/connector'
import { useAddConnector, useAvailableConnectors, useManagedConnectors } from '@entities/connector'
import { useRepositories } from '@entities/repository'
import CatalogConnectorCard from '@features/catalog-browse/CatalogConnectorCard.vue'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SBadge, SButton, SEmptyState, SInput, SSkeleton, useToast } from '@shared/ui'
import { useDebounce } from '@vueuse/core'
import { Database, FolderPlus, RefreshCw } from 'lucide-vue-next'
import { computed, ref } from 'vue'

const toast = useToast()

const { data: catalog, isLoading: catalogLoading, error: catalogError } = useAvailableConnectors()
const { data: managed } = useManagedConnectors()
const { data: repositories, isLoading: reposLoading } = useRepositories()
const installMutation = useAddConnector()

const search = ref('')
const debouncedSearch = useDebounce(search, 250)
const typeFilter = ref<'' | ConnectorType>('')
const stageFilter = ref<'' | ReleaseStage>('')
const installingImage = ref<string | null>(null)

const installedIndex = computed(() => {
  const map = new Map<string, string>() // key: "image:tag" -> managed-connector id
  for (const conn of managed.value ?? [])
    map.set(`${conn.dockerImage}:${conn.dockerTag}`, conn.id)
  return map
})

function isInstalled(c: Connector): boolean {
  return installedIndex.value.has(`${c.dockerImage}:${c.latestVersion}`)
}

function installedId(c: Connector): string | null {
  return installedIndex.value.get(`${c.dockerImage}:${c.latestVersion}`) ?? null
}

const filtered = computed<Connector[]>(() => {
  const all = catalog.value ?? []
  const query = debouncedSearch.value.trim().toLowerCase()

  return all.filter((c) => {
    if (typeFilter.value && c.type !== typeFilter.value)
      return false
    if (stageFilter.value && c.releaseStage !== stageFilter.value)
      return false
    if (query
      && !c.name.toLowerCase().includes(query)
      && !c.dockerImage.toLowerCase().includes(query)) {
      return false
    }
    return true
  })
})

const hasRepositories = computed(() => (repositories.value?.length ?? 0) > 0)
const hasFilters = computed(() => !!debouncedSearch.value || !!typeFilter.value || !!stageFilter.value)

const stageOptions: { label: string, value: '' | ReleaseStage }[] = [
  { label: 'All stages', value: '' },
  { label: 'GA', value: 'generally_available' },
  { label: 'Beta', value: 'beta' },
  { label: 'Alpha', value: 'alpha' },
]

const typeOptions: { label: string, value: '' | ConnectorType }[] = [
  { label: 'All', value: '' },
  { label: 'Source', value: 'source' },
  { label: 'Destination', value: 'destination' },
]

async function handleInstall(connector: Connector) {
  installingImage.value = connector.dockerImage
  try {
    await installMutation.mutateAsync({
      dockerImage: connector.dockerImage,
      dockerTag: connector.latestVersion,
      name: connector.name,
      connectorType: connector.type,
    })
    toast.success(`Installed ${connector.name}`)
  }
  catch (e: unknown) {
    toast.error(getErrorMessage(e) || `Failed to install ${connector.name}`)
  }
  finally {
    installingImage.value = null
  }
}

function clearSearch() {
  search.value = ''
  typeFilter.value = ''
  stageFilter.value = ''
}
</script>

<template>
  <PageHeader
    title="Connector catalog"
    description="Browse and install connectors from your synced repositories"
    back-label="Connectors"
    :back-to="{ name: 'settings-connectors' }"
  />

  <SAlert v-if="catalogError" variant="danger" class="mb-4">
    Failed to load catalog: {{ catalogError.message }}
  </SAlert>

  <template v-if="catalogLoading || reposLoading">
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      <SSkeleton v-for="i in 6" :key="i" variant="rect" height="140px" />
    </div>
  </template>

  <template v-else-if="(catalog?.length ?? 0) === 0 && !hasRepositories">
    <SEmptyState
      title="No repositories synced"
      description="Add a connector repository and sync it to browse connectors."
      :icon="FolderPlus"
    >
      <template #actions>
        <SButton :to="{ name: 'settings-connectors' }">
          Manage repositories
        </SButton>
      </template>
    </SEmptyState>
  </template>

  <template v-else-if="(catalog?.length ?? 0) === 0">
    <SEmptyState
      title="Catalog is empty"
      description="Your repositories synced but returned no connectors. Try Sync now from the repositories tab."
      :icon="RefreshCw"
    >
      <template #actions>
        <SButton variant="secondary" :to="{ name: 'settings-connectors' }">
          Open repositories
        </SButton>
      </template>
    </SEmptyState>
  </template>

  <template v-else>
    <div class="flex flex-wrap items-center gap-3 mb-6">
      <SInput v-model="search" class="w-72" placeholder="Search connectors..." />

      <div class="flex gap-1">
        <button
          v-for="opt in typeOptions"
          :key="opt.value"
          class="px-3 py-1 text-xs rounded-full border transition-colors"
          :class="typeFilter === opt.value
            ? 'bg-primary text-white border-primary'
            : 'border-border text-text-primary hover:bg-surface-hover'"
          @click="typeFilter = opt.value"
        >
          {{ opt.label }}
        </button>
      </div>

      <div class="flex gap-1">
        <button
          v-for="opt in stageOptions"
          :key="opt.value"
          class="px-3 py-1 text-xs rounded-full border transition-colors"
          :class="stageFilter === opt.value
            ? 'bg-primary text-white border-primary'
            : 'border-border text-text-primary hover:bg-surface-hover'"
          @click="stageFilter = opt.value"
        >
          {{ opt.label }}
        </button>
      </div>

      <div class="ml-auto text-sm text-text-muted">
        <SBadge variant="gray">
          {{ filtered.length }} of {{ catalog?.length ?? 0 }}
        </SBadge>
      </div>
    </div>

    <SEmptyState
      v-if="filtered.length === 0"
      :icon="Database"
      :title="`No connectors match '${debouncedSearch || 'your filters'}'`"
      description="Try clearing the search or removing filters."
    >
      <template #actions>
        <SButton v-if="hasFilters" variant="secondary" @click="clearSearch">
          Clear search
        </SButton>
      </template>
    </SEmptyState>

    <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      <CatalogConnectorCard
        v-for="connector in filtered"
        :key="connector.dockerImage"
        :connector="connector"
        :installed="isInstalled(connector)"
        :installed-id="installedId(connector)"
        :installing="installingImage === connector.dockerImage"
        @install="handleInstall"
      />
    </div>
  </template>
</template>
