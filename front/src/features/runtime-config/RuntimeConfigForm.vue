<script setup lang="ts">
import type { Tab } from '@shared/ui'
import { useRuntimeDefaults } from '@entities/system'
import { SAlert, SButton, SCard, SCheckbox, SConfirmDialog, SInput, STabs, STextarea } from '@shared/ui'
import { Trash2 } from 'lucide-vue-next'
import { computed, reactive, ref, watch } from 'vue'

interface DockerVolume {
  hostPath: string
  containerPath: string
  readOnly: boolean
}

const props = defineProps<{
  runtimeConfig: string | null
  entityType: 'source' | 'destination'
  saving: boolean
}>()

const emit = defineEmits<{
  save: [configJson: string | null]
  reset: []
}>()

// Common
const cpuRequest = ref('')
const cpuLimit = ref('')
const memoryRequest = ref('')
const memoryLimit = ref('')

// Docker
const dockerNetwork = ref('')
const dockerVolumes = ref<DockerVolume[]>([])

// K8s
const tolerationsJson = ref('')
const nodeSelectorJson = ref('')
const affinityJson = ref('')

const runtimeDefaults = useRuntimeDefaults()
const defaults = computed(() => runtimeDefaults.data.value)

function defaultPlaceholder(value: string | undefined, example: string): string {
  if (!defaults.value)
    return example
  return value ? `Default: ${value}` : `No default (${example})`
}

const placeholders = computed(() => ({
  cpuRequest: defaultPlaceholder(defaults.value?.cpuRequest, 'e.g. 250m'),
  cpuLimit: defaultPlaceholder(defaults.value?.cpuLimit, 'e.g. 1000m'),
  memoryRequest: defaultPlaceholder(defaults.value?.memoryRequest, 'e.g. 256Mi'),
  memoryLimit: defaultPlaceholder(defaults.value?.memoryLimit, 'e.g. 512Mi'),
}))

const validationErrors = reactive<Record<string, string>>({})
const confirmResetOpen = ref(false)

type TabValue = 'common' | 'docker' | 'k8s'

const activeTab = ref<TabValue>('common')
const tabs: Tab[] = [
  { name: 'Common', value: 'common' },
  { name: 'Docker', value: 'docker' },
  { name: 'Kubernetes', value: 'k8s' },
]

function parseConfig(json: string | null) {
  cpuRequest.value = ''
  cpuLimit.value = ''
  memoryRequest.value = ''
  memoryLimit.value = ''
  dockerNetwork.value = ''
  dockerVolumes.value = []
  tolerationsJson.value = ''
  nodeSelectorJson.value = ''
  affinityJson.value = ''
  if (!json)
    return
  try {
    const config = JSON.parse(json)
    cpuRequest.value = config.cpu_request ?? config.cpuRequest ?? ''
    cpuLimit.value = config.cpu_limit ?? config.cpuLimit ?? ''
    memoryRequest.value = config.memory_request ?? config.memoryRequest ?? ''
    memoryLimit.value = config.memory_limit ?? config.memoryLimit ?? ''
    dockerNetwork.value = config.docker_network ?? config.dockerNetwork ?? ''
    const volumes = config.docker_volumes ?? config.dockerVolumes ?? []
    dockerVolumes.value = Array.isArray(volumes)
      ? volumes.map((v: Record<string, unknown>) => ({
          hostPath: (v.host_path as string) ?? (v.hostPath as string) ?? '',
          containerPath: (v.container_path as string) ?? (v.containerPath as string) ?? '',
          readOnly: Boolean(v.read_only ?? v.readOnly),
        }))
      : []
    tolerationsJson.value = config.tolerations ? JSON.stringify(config.tolerations, null, 2) : ''
    nodeSelectorJson.value = config.node_selector
      ? JSON.stringify(config.node_selector, null, 2)
      : config.nodeSelector ? JSON.stringify(config.nodeSelector, null, 2) : ''
    affinityJson.value = config.affinity ? JSON.stringify(config.affinity, null, 2) : ''
  }
  catch {
    // Invalid JSON, leave fields empty.
  }
}

watch(() => props.runtimeConfig, val => parseConfig(val), { immediate: true })

const cpuRegex = /^(?:\d+m|\d+(?:\.\d+)?|\.\d+)$/
const memoryRegex = /^(?:\d+(?:\.\d*)?|\.\d+)(?:[KMGTPE]i|[kMGTPE]|[eE][+-]?\d+)?$/

function errorTab(key: string): TabValue {
  if (key === 'cpuLimit' || key === 'memoryLimit')
    return 'common'
  if (key.startsWith('vol'))
    return 'docker'
  return 'k8s'
}

const tabsWithErrors = computed(() => {
  const withErrors = new Set(Object.keys(validationErrors).map(errorTab))
  return tabs.filter(t => withErrors.has(t.value as TabValue))
})

function clearErrors() {
  for (const k of Object.keys(validationErrors))
    delete validationErrors[k]
}

function validate(): boolean {
  clearErrors()

  if (cpuRequest.value && !cpuRegex.test(cpuRequest.value))
    validationErrors.cpuRequest = 'Invalid CPU value. Use Kubernetes format: 250m, 0.5, 1'
  if (cpuLimit.value && !cpuRegex.test(cpuLimit.value))
    validationErrors.cpuLimit = 'Invalid CPU value. Use Kubernetes format: 250m, 0.5, 1'
  if (memoryRequest.value && !memoryRegex.test(memoryRequest.value))
    validationErrors.memoryRequest = 'Invalid memory value. Use Kubernetes format: 128Mi, 1Gi, 512M'
  if (memoryLimit.value && !memoryRegex.test(memoryLimit.value))
    validationErrors.memoryLimit = 'Invalid memory value. Use Kubernetes format: 128Mi, 1Gi, 512M'

  dockerVolumes.value.forEach((vol, i) => {
    if (vol.hostPath || vol.containerPath) {
      if (!vol.hostPath)
        validationErrors[`vol${i}Host`] = 'Host path is required'
      if (!vol.containerPath)
        validationErrors[`vol${i}Container`] = 'Container path is required'
      if (vol.containerPath === '/tmp' || vol.containerPath.startsWith('/tmp/'))
        validationErrors[`vol${i}Container`] = 'Cannot mount to /tmp (reserved by synclet)'
    }
  })

  for (const [key, value] of [['tolerations', tolerationsJson.value], ['nodeSelector', nodeSelectorJson.value], ['affinity', affinityJson.value]] as const) {
    if (value) {
      try {
        JSON.parse(value)
      }
      catch {
        validationErrors[key] = 'Invalid JSON. Check syntax and try again.'
      }
    }
  }

  return Object.keys(validationErrors).length === 0
}

function buildConfigJson(): string | null {
  const config: Record<string, unknown> = {}
  if (cpuRequest.value)
    config.cpu_request = cpuRequest.value
  if (cpuLimit.value)
    config.cpu_limit = cpuLimit.value
  if (memoryRequest.value)
    config.memory_request = memoryRequest.value
  if (memoryLimit.value)
    config.memory_limit = memoryLimit.value
  if (dockerNetwork.value)
    config.docker_network = dockerNetwork.value

  const filteredVolumes = dockerVolumes.value
    .filter(v => v.hostPath && v.containerPath)
    .map(v => ({ host_path: v.hostPath, container_path: v.containerPath, read_only: v.readOnly }))
  if (filteredVolumes.length > 0)
    config.docker_volumes = filteredVolumes

  if (tolerationsJson.value)
    config.tolerations = JSON.parse(tolerationsJson.value)
  if (nodeSelectorJson.value)
    config.node_selector = JSON.parse(nodeSelectorJson.value)
  if (affinityJson.value)
    config.affinity = JSON.parse(affinityJson.value)

  return Object.keys(config).length > 0 ? JSON.stringify(config) : null
}

function handleSave() {
  if (!validate()) {
    const first = tabsWithErrors.value[0]
    if (first && !tabsWithErrors.value.some(t => t.value === activeTab.value))
      activeTab.value = first.value as TabValue
    return
  }
  emit('save', buildConfigJson())
}

function handleReset() {
  confirmResetOpen.value = true
}

function confirmReset() {
  confirmResetOpen.value = false
  emit('reset')
}

function addVolume() {
  dockerVolumes.value.push({ hostPath: '', containerPath: '', readOnly: false })
}

function removeVolume(index: number) {
  dockerVolumes.value.splice(index, 1)
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
            Configure container resources and executor-specific options for this {{ entityType }}. Empty fields use the global default shown as their placeholder (see Settings &rsaquo; General &rsaquo; Runtime defaults).
          </p>
        </div>
      </div>
    </template>

    <div class="p-6">
      <STabs v-model="activeTab" :tabs="tabs" variant="pills" />

      <!-- Common tab -->
      <div v-if="activeTab === 'common'" class="space-y-4">
        <p class="text-xs text-text-muted">
          Resource limits apply to both Docker and Kubernetes executors. Requests (lower bounds) are Kubernetes-only — set them in the Kubernetes tab.
        </p>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <SInput v-model="cpuLimit" label="CPU Limit" :placeholder="placeholders.cpuLimit" :error="validationErrors.cpuLimit" />
          <SInput v-model="memoryLimit" label="Memory Limit" :placeholder="placeholders.memoryLimit" :error="validationErrors.memoryLimit" />
        </div>
      </div>

      <!-- Docker tab -->
      <div v-else-if="activeTab === 'docker'" class="space-y-6">
        <p class="text-xs text-text-muted">
          Docker-specific options. Ignored when the executor is Kubernetes.
        </p>

        <div>
          <SInput v-model="dockerNetwork" label="Network mode" placeholder="bridge (default), host, none, or a custom network name" />
          <p class="text-xs text-text-muted mt-1">
            Defaults to <code class="text-text-primary">bridge</code> for sync containers. Use <code class="text-text-primary">host</code> to share the host network or specify a Docker network name to attach to.
          </p>
        </div>

        <div>
          <div class="flex items-center justify-between mb-2">
            <label class="text-sm font-medium text-text-primary">Volume mounts</label>
            <SButton variant="secondary" size="sm" @click="addVolume">
              Add volume
            </SButton>
          </div>
          <p class="text-xs text-text-muted mb-3">
            Bind host directories into the container — useful for syncing CSV files or SQLite databases. The container path cannot start with <code class="text-text-primary">/tmp</code>.
          </p>
          <div v-if="dockerVolumes.length === 0" class="text-xs text-text-muted italic py-3 text-center border border-dashed border-border rounded-lg">
            No volume mounts configured.
          </div>
          <div v-for="(vol, idx) in dockerVolumes" :key="idx" class="grid grid-cols-1 md:grid-cols-12 gap-2 mb-2 items-start">
            <div class="md:col-span-5">
              <SInput v-model="vol.hostPath" placeholder="Host path (e.g. /data/csv)" :error="validationErrors[`vol${idx}Host`]" />
            </div>
            <div class="md:col-span-5">
              <SInput v-model="vol.containerPath" placeholder="Container path (e.g. /data)" :error="validationErrors[`vol${idx}Container`]" />
            </div>
            <SCheckbox v-model="vol.readOnly" label="RO" title="Mount read-only" class="md:col-span-1 mt-2.5" />
            <button
              type="button"
              class="md:col-span-1 mt-2 p-1.5 text-text-muted hover:text-danger transition-colors"
              title="Remove"
              @click="removeVolume(idx)"
            >
              <Trash2 class="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>

      <!-- K8s tab -->
      <div v-else class="space-y-4">
        <p class="text-xs text-text-muted">
          Kubernetes-specific scheduling and requests. Ignored when the executor is Docker.
        </p>

        <div>
          <h4 class="text-sm font-medium text-heading mb-3">
            Resource requests
          </h4>
          <p class="text-xs text-text-muted mb-3">
            Lower bounds the scheduler reserves on the node. Limits set in the Common tab are the upper bounds.
          </p>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <SInput v-model="cpuRequest" label="CPU Request" :placeholder="placeholders.cpuRequest" :error="validationErrors.cpuRequest" />
            <SInput v-model="memoryRequest" label="Memory Request" :placeholder="placeholders.memoryRequest" :error="validationErrors.memoryRequest" />
          </div>
        </div>

        <STextarea
          v-model="tolerationsJson"
          label="Tolerations"
          :rows="6"
          placeholder="[{&quot;key&quot;: &quot;dedicated&quot;, &quot;operator&quot;: &quot;Equal&quot;, &quot;value&quot;: &quot;sync&quot;, &quot;effect&quot;: &quot;NoSchedule&quot;}]"
          monospace
          :error="validationErrors.tolerations"
        />

        <STextarea
          v-model="nodeSelectorJson"
          label="Node Selector"
          :rows="4"
          placeholder="{&quot;disktype&quot;: &quot;ssd&quot;}"
          monospace
          :error="validationErrors.nodeSelector"
        />

        <STextarea
          v-model="affinityJson"
          label="Pod Affinity"
          :rows="6"
          placeholder="{&quot;nodeAffinity&quot;: {...}}"
          monospace
          :error="validationErrors.affinity"
        />
      </div>

      <SAlert v-if="tabsWithErrors.length > 0" variant="danger" class="mt-6">
        Fix the highlighted fields on the {{ tabsWithErrors.map(t => t.name).join(', ') }} {{ tabsWithErrors.length === 1 ? 'tab' : 'tabs' }} before saving.
      </SAlert>

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

  <SConfirmDialog
    :open="confirmResetOpen"
    title="Reset runtime config"
    message="This will clear all overrides and revert to global defaults. Continue?"
    confirm-text="Reset"
    @confirm="confirmReset"
    @cancel="confirmResetOpen = false"
  />
</template>
