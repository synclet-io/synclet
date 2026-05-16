<script setup lang="ts">
import type { Connector } from '@entities/connector'
import { SBadge, SButton, SCard } from '@shared/ui'
import { Check, Download } from 'lucide-vue-next'
import { useRouter } from 'vue-router'

defineProps<{
  connector: Connector
  installed: boolean
  installedId: string | null
  installing: boolean
}>()

const emit = defineEmits<{
  install: [connector: Connector]
}>()

const router = useRouter()

function releaseVariant(stage: string): 'success' | 'info' | 'warning' | 'gray' {
  switch (stage) {
    case 'generally_available': return 'success'
    case 'beta': return 'info'
    case 'alpha': return 'warning'
    default: return 'gray'
  }
}

function releaseLabel(stage: string): string {
  switch (stage) {
    case 'generally_available': return 'GA'
    case 'beta': return 'Beta'
    case 'alpha': return 'Alpha'
    case 'custom': return 'Custom'
    default: return stage
  }
}

function configureInstalled() {
  // Phase 1: route back to the managed-connectors browser; once a dedicated
  // managed-connector detail route exists, target it here directly.
  router.push({ name: 'settings-connectors' })
}
</script>

<template>
  <SCard class="flex flex-col gap-3 p-4">
    <div class="flex items-start gap-3">
      <img
        v-if="connector.iconUrl"
        :src="connector.iconUrl"
        :alt="connector.name"
        class="w-10 h-10 rounded object-contain bg-surface-raised"
        loading="lazy"
      >
      <div class="flex-1 min-w-0">
        <h3 class="text-sm font-semibold text-heading truncate" :title="connector.name">
          {{ connector.name }}
        </h3>
        <code class="text-xs text-text-muted truncate block" :title="connector.dockerImage">
          {{ connector.dockerImage }}
        </code>
      </div>
    </div>

    <div class="flex flex-wrap gap-1">
      <SBadge variant="gray">
        {{ connector.type }}
      </SBadge>
      <SBadge :variant="releaseVariant(connector.releaseStage)">
        {{ releaseLabel(connector.releaseStage) }}
      </SBadge>
      <SBadge v-if="connector.supportLevel === 'certified'" variant="info">
        Certified
      </SBadge>
    </div>

    <div class="flex items-center justify-between gap-2 mt-auto">
      <SBadge v-if="installed" variant="success" dot>
        <Check class="w-3 h-3 mr-1 inline-block" />
        Installed
      </SBadge>
      <span v-else class="text-xs text-text-muted">{{ connector.latestVersion }}</span>
      <SButton
        v-if="installed"
        size="sm"
        variant="ghost"
        :disabled="!installedId"
        @click="configureInstalled"
      >
        Configure
      </SButton>
      <SButton
        v-else
        size="sm"
        :loading="installing"
        @click="emit('install', connector)"
      >
        <Download class="w-3.5 h-3.5" />
        Install
      </SButton>
    </div>
  </SCard>
</template>
