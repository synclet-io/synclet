<script setup lang="ts">
import { SButton, SCard, SInput } from '@shared/ui'
import { ref, watch } from 'vue'

const props = defineProps<{
  runtimeConfig: string | null
  entityType: 'source' | 'destination'
  saving: boolean
}>()

const emit = defineEmits<{
  save: [configJson: string | null]
  reset: []
}>()

const cpuRequest = ref('')
const cpuLimit = ref('')
const memoryRequest = ref('')
const memoryLimit = ref('')
const serviceAccountName = ref('')
const tolerationsJson = ref('')
const nodeSelectorJson = ref('')
const affinityJson = ref('')
const validationErrors = ref<Record<string, string>>({})

function parseConfig(json: string | null) {
  if (!json) {
    cpuRequest.value = ''
    cpuLimit.value = ''
    memoryRequest.value = ''
    memoryLimit.value = ''
    serviceAccountName.value = ''
    tolerationsJson.value = ''
    nodeSelectorJson.value = ''
    affinityJson.value = ''
    return
  }
  try {
    const config = JSON.parse(json)
    cpuRequest.value = config.cpuRequest ?? ''
    cpuLimit.value = config.cpuLimit ?? ''
    memoryRequest.value = config.memoryRequest ?? ''
    memoryLimit.value = config.memoryLimit ?? ''
    serviceAccountName.value = config.serviceAccountName ?? ''
    tolerationsJson.value = config.tolerations ? JSON.stringify(config.tolerations, null, 2) : ''
    nodeSelectorJson.value = config.nodeSelector ? JSON.stringify(config.nodeSelector, null, 2) : ''
    affinityJson.value = config.affinity ? JSON.stringify(config.affinity, null, 2) : ''
  }
  catch {
    // If config is invalid JSON, leave fields empty
  }
}

watch(() => props.runtimeConfig, val => parseConfig(val), { immediate: true })

const cpuRegex = /^\d+(?:\.\d+)?m?$/
const memoryRegex = /^\d+(?:\.\d+)?(?:Mi|Gi|Ki|Ti)?$/

function validate(): boolean {
  const errors: Record<string, string> = {}

  if (cpuRequest.value && !cpuRegex.test(cpuRequest.value)) {
    errors.cpuRequest = 'Invalid CPU value. Use Kubernetes format: 250m, 0.5, 1'
  }
  if (cpuLimit.value && !cpuRegex.test(cpuLimit.value)) {
    errors.cpuLimit = 'Invalid CPU value. Use Kubernetes format: 250m, 0.5, 1'
  }
  if (memoryRequest.value && !memoryRegex.test(memoryRequest.value)) {
    errors.memoryRequest = 'Invalid memory value. Use Kubernetes format: 128Mi, 1Gi'
  }
  if (memoryLimit.value && !memoryRegex.test(memoryLimit.value)) {
    errors.memoryLimit = 'Invalid memory value. Use Kubernetes format: 128Mi, 1Gi'
  }

  if (tolerationsJson.value) {
    try {
      JSON.parse(tolerationsJson.value)
    }
    catch {
      errors.tolerations = 'Invalid JSON. Check syntax and try again.'
    }
  }
  if (nodeSelectorJson.value) {
    try {
      JSON.parse(nodeSelectorJson.value)
    }
    catch {
      errors.nodeSelector = 'Invalid JSON. Check syntax and try again.'
    }
  }
  if (affinityJson.value) {
    try {
      JSON.parse(affinityJson.value)
    }
    catch {
      errors.affinity = 'Invalid JSON. Check syntax and try again.'
    }
  }

  validationErrors.value = errors
  return Object.keys(errors).length === 0
}

function buildConfigJson(): string | null {
  const config: Record<string, unknown> = {}
  if (cpuRequest.value)
    config.cpuRequest = cpuRequest.value
  if (cpuLimit.value)
    config.cpuLimit = cpuLimit.value
  if (memoryRequest.value)
    config.memoryRequest = memoryRequest.value
  if (memoryLimit.value)
    config.memoryLimit = memoryLimit.value
  if (serviceAccountName.value)
    config.serviceAccountName = serviceAccountName.value
  if (tolerationsJson.value)
    config.tolerations = JSON.parse(tolerationsJson.value)
  if (nodeSelectorJson.value)
    config.nodeSelector = JSON.parse(nodeSelectorJson.value)
  if (affinityJson.value)
    config.affinity = JSON.parse(affinityJson.value)

  return Object.keys(config).length > 0 ? JSON.stringify(config) : null
}

function handleSave() {
  if (!validate())
    return
  emit('save', buildConfigJson())
}

function handleReset() {
  // eslint-disable-next-line no-alert
  if (window.confirm('Reset runtime config: This will clear all overrides and revert to global defaults. Continue?')) {
    emit('reset')
  }
}
</script>

<template>
  <SCard>
    <template #header>
      <div class="flex items-center justify-between w-full">
        <div>
          <h3 class="text-base font-semibold text-heading">
            Runtime Configuration
          </h3>
          <p class="text-sm text-text-muted mt-1">
            Configure container resource limits and Kubernetes scheduling for this {{ entityType }}.
          </p>
        </div>
      </div>
    </template>

    <div class="p-6">
      <div>
        <h4 class="text-sm font-medium text-heading mb-3">
          Resource Limits
        </h4>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <SInput
            v-model="cpuRequest" label="CPU Request" placeholder="e.g. 250m"
            :error="validationErrors.cpuRequest"
          />
          <SInput
            v-model="cpuLimit" label="CPU Limit" placeholder="e.g. 1000m"
            :error="validationErrors.cpuLimit"
          />
          <SInput
            v-model="memoryRequest" label="Memory Request" placeholder="e.g. 256Mi"
            :error="validationErrors.memoryRequest"
          />
          <SInput
            v-model="memoryLimit" label="Memory Limit" placeholder="e.g. 512Mi"
            :error="validationErrors.memoryLimit"
          />
        </div>
        <p class="text-xs text-text-muted mt-2">
          Leave blank to use global defaults.
        </p>
      </div>

      <div class="mt-6">
        <h4 class="text-sm font-medium text-heading mb-3">
          Advanced Kubernetes Settings
        </h4>
        <div class="space-y-4">
          <div>
            <SInput v-model="serviceAccountName" label="Service Account Name" placeholder="e.g. my-sync-service-account" />
            <p class="text-xs text-text-muted mt-1 mb-2">
              Kubernetes service account for the replication pod. Leave blank to use the global default.
            </p>
          </div>
          <div>
            <label class="text-sm text-text-primary mb-1 block">Tolerations</label>
            <textarea
              v-model="tolerationsJson" rows="6"
              placeholder="[{&quot;key&quot;: &quot;dedicated&quot;, &quot;operator&quot;: &quot;Equal&quot;, &quot;value&quot;: &quot;sync&quot;, &quot;effect&quot;: &quot;NoSchedule&quot;}]"
              class="w-full px-3 py-2 border border-border rounded-lg bg-surface text-sm font-mono text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
            />
            <p v-if="validationErrors.tolerations" class="text-xs text-danger mt-1.5">
              {{ validationErrors.tolerations }}
            </p>
          </div>
          <div>
            <label class="text-sm text-text-primary mb-1 block">Node Selector</label>
            <textarea
              v-model="nodeSelectorJson" rows="4"
              placeholder="{&quot;disktype&quot;: &quot;ssd&quot;}"
              class="w-full px-3 py-2 border border-border rounded-lg bg-surface text-sm font-mono text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
            />
            <p v-if="validationErrors.nodeSelector" class="text-xs text-danger mt-1.5">
              {{ validationErrors.nodeSelector }}
            </p>
          </div>
          <div>
            <label class="text-sm text-text-primary mb-1 block">Pod Affinity</label>
            <textarea
              v-model="affinityJson" rows="6"
              placeholder="{&quot;nodeAffinity&quot;: {...}}"
              class="w-full px-3 py-2 border border-border rounded-lg bg-surface text-sm font-mono text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
            />
            <p v-if="validationErrors.affinity" class="text-xs text-danger mt-1.5">
              {{ validationErrors.affinity }}
            </p>
          </div>
        </div>
      </div>

      <div class="flex justify-end gap-3 mt-6 pt-4 border-t border-border">
        <SButton variant="secondary" class="text-danger" @click="handleReset">
          Reset to Defaults
        </SButton>
        <SButton :loading="saving" @click="handleSave">
          Save Runtime Config
        </SButton>
      </div>
    </div>
  </SCard>
</template>
