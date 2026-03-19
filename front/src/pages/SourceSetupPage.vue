<script setup lang="ts">
import type { ExternalDocumentationUrl, ManagedConnector } from '@entities/connector'
import { useConnectorTaskResult } from '@entities/connector-task'
import { getConnectorSpec, listManagedConnectors } from '@entities/connector/api'
import { createSource, testConnection as testSourceConnection } from '@entities/source/api'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SButton, SCard, SInput } from '@shared/ui'
import JsonSchemaForm from '@shared/ui/JsonSchemaForm.vue'
import { ExternalLink } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const step = ref(1)
const connectors = ref<ManagedConnector[]>([])
const selectedConnector = ref<ManagedConnector | null>(null)
const spec = ref<any>(null)
const name = ref('')
const configValues = ref<Record<string, any>>({})
const showJsonEditor = ref(false)
const jsonEditorText = ref('{}')
const loading = ref(false)
const error = ref('')
const search = ref('')
const testing = ref(false)
const externalDocs = ref<ExternalDocumentationUrl[]>([])

const testTaskId = ref<string | null>(null)
const { data: testTaskResult } = useConnectorTaskResult(testTaskId)

const testResult = computed(() => {
  if (!testTaskResult.value)
    return null
  const task = testTaskResult.value
  if (task.status === 'pending' || task.status === 'running')
    return null
  if (task.status === 'failed')
    return { success: false, message: task.errorMessage || 'Connection test failed' }
  if (task.status === 'completed' && task.checkResult) {
    return { success: task.checkResult.success, message: task.checkResult.message || (task.checkResult.success ? 'Connection successful' : 'Connection failed') }
  }
  return null
})

const testPolling = computed(() => {
  if (!testTaskId.value)
    return false
  if (!testTaskResult.value)
    return true
  return testTaskResult.value.status === 'pending' || testTaskResult.value.status === 'running'
})

const DOC_GROUPS = [
  { key: 'setup', label: 'Setup Guides', types: ['authentication_guide'] },
  { key: 'reference', label: 'Reference', types: ['api_reference', 'rate_limits'] },
  { key: 'status', label: 'Status', types: ['api_status_page', 'api_release_history'] },
] as const

const groupedDocs = computed(() =>
  DOC_GROUPS
    .map(group => ({
      ...group,
      links: externalDocs.value.filter(d => (group.types as readonly string[]).includes(d.type)),
    }))
    .filter(group => group.links.length > 0),
)

const hasExternalDocs = computed(() => externalDocs.value.length > 0)

const connectionSpec = computed(() => spec.value?.connectionSpecification ?? null)

watch(configValues, (val) => {
  if (!showJsonEditor.value) {
    jsonEditorText.value = JSON.stringify(val, null, 2)
  }
}, { deep: true })

function syncJsonToForm() {
  try {
    configValues.value = JSON.parse(jsonEditorText.value)
    error.value = ''
  }
  catch {
    error.value = 'Invalid JSON'
  }
}

onMounted(async () => {
  try {
    const all = await listManagedConnectors()
    connectors.value = all.filter(c => c.connectorType === 'source')
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e)
  }
})

async function selectConnector(c: ManagedConnector) {
  selectedConnector.value = c
  loading.value = true
  try {
    const result = await getConnectorSpec(c.id)
    spec.value = result.spec
    externalDocs.value = result.externalDocumentationUrls
    configValues.value = {}
    step.value = 2
  }
  catch (e: unknown) {
    error.value = `Failed to load spec: ${getErrorMessage(e)}`
  }
  finally {
    loading.value = false
  }
}

function syncJsonIfNeeded() {
  if (showJsonEditor.value) {
    try {
      configValues.value = JSON.parse(jsonEditorText.value)
    }
    catch {
      // leave configValues as-is if JSON is invalid
    }
  }
}

function cleanConfig(obj: Record<string, any>): Record<string, any> {
  const result: Record<string, any> = {}
  for (const [k, v] of Object.entries(obj)) {
    if (k.startsWith('__'))
      continue
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      result[k] = cleanConfig(v)
    }
    else {
      result[k] = v
    }
  }
  return result
}

async function handleCreateSource() {
  if (!selectedConnector.value)
    return
  syncJsonIfNeeded()
  loading.value = true
  error.value = ''
  testTaskId.value = null
  try {
    await createSource({
      name: name.value,
      managedConnectorId: selectedConnector.value.id,
      config: cleanConfig(configValues.value),
    })
    router.push('/sources')
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e)
  }
  finally {
    loading.value = false
  }
}

async function handleTestConnection() {
  if (!selectedConnector.value)
    return
  syncJsonIfNeeded()
  testing.value = true
  testTaskId.value = null
  error.value = ''
  try {
    const { taskId } = await testSourceConnection({
      managedConnectorId: selectedConnector.value.id,
      config: cleanConfig(configValues.value),
    })
    testTaskId.value = taskId
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e)
  }
  finally {
    testing.value = false
  }
}

function filteredConnectors() {
  if (!search.value)
    return connectors.value
  const q = search.value.toLowerCase()
  return connectors.value.filter(c => c.name.toLowerCase().includes(q))
}
</script>

<template>
  <PageHeader title="Add Source" description="Configure a new data source" />

  <SAlert v-if="error" variant="danger" class="mb-4" dismissible @dismiss="error = ''">
    {{ error }}
  </SAlert>

  <!-- Step 1: Select connector -->
  <div v-if="step === 1">
    <SInput v-model="search" placeholder="Search source connectors..." class="mb-4" />
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
      <button
        v-for="c in filteredConnectors()" :key="c.id" class="text-left bg-surface rounded-xl border border-border p-4 hover:border-primary hover:shadow-raised transition-all duration-200"
        @click="selectConnector(c)"
      >
        <div class="flex items-center gap-3">
          <div>
            <h3 class="text-sm font-semibold text-heading">
              {{ c.name }}
            </h3>
            <p class="text-xs text-text-muted mt-0.5">
              {{ c.dockerImage }}:{{ c.dockerTag }}
            </p>
          </div>
        </div>
      </button>
    </div>
    <div v-if="loading" class="text-center py-8 text-text-muted">
      Loading connector spec...
    </div>
  </div>

  <!-- Step 2: Configure -->
  <div v-if="step === 2" :class="hasExternalDocs ? 'max-w-5xl flex flex-col lg:flex-row gap-6' : 'max-w-2xl'">
    <div :class="hasExternalDocs ? 'flex-1 min-w-0' : ''">
      <SCard>
        <div class="flex items-center gap-3 mb-6 pb-4 border-b border-border">
          <div>
            <h3 class="font-semibold text-heading">
              {{ selectedConnector?.name }}
            </h3>
            <p class="text-xs text-text-muted mt-0.5">
              {{ selectedConnector?.dockerImage }}:{{ selectedConnector?.dockerTag }}
            </p>
          </div>
          <button class="ml-auto text-sm text-primary hover:text-primary-hover" @click="step = 1">
            Change
          </button>
        </div>

        <form class="space-y-4" @submit.prevent="handleCreateSource">
          <SInput v-model="name" label="Source Name" placeholder="My Source" required />

          <div>
            <div class="flex items-center justify-between mb-2">
              <label class="text-sm font-medium text-heading">Configuration</label>
              <button
                type="button" class="text-xs text-primary hover:text-primary-hover"
                @click="showJsonEditor = !showJsonEditor; if (!showJsonEditor) syncJsonToForm()"
              >
                {{ showJsonEditor ? 'Switch to Form' : 'Edit as JSON' }}
              </button>
            </div>
            <div v-if="showJsonEditor">
              <textarea
                v-model="jsonEditorText" rows="10" placeholder="{}"
                class="w-full px-3 py-2 border border-border rounded-lg bg-surface text-sm font-mono text-text-primary"
              />
            </div>
            <div v-else-if="connectionSpec">
              <JsonSchemaForm v-model="configValues" :schema="connectionSpec" />
            </div>
            <div v-else>
              <textarea
                v-model="jsonEditorText" rows="10" placeholder="{}"
                class="w-full px-3 py-2 border border-border rounded-lg bg-surface text-sm font-mono text-text-primary"
              />
            </div>
          </div>

          <SAlert v-if="testPolling" variant="info">
            Testing connection...
          </SAlert>
          <SAlert v-else-if="testResult" :variant="testResult.success ? 'success' : 'danger'">
            {{ testResult.message }}
          </SAlert>

          <div class="flex gap-3 pt-2">
            <SButton variant="secondary" type="button" @click="step = 1">
              Back
            </SButton>
            <SButton variant="secondary" type="button" :loading="testing || testPolling" @click="handleTestConnection">
              Test Connection
            </SButton>
            <SButton type="submit" :loading="loading" :disabled="!name">
              {{ loading ? 'Testing connection...' : 'Create Source' }}
            </SButton>
          </div>
        </form>
      </SCard>
    </div>

    <!-- Documentation sidebar -->
    <div v-if="hasExternalDocs" class="w-full lg:w-72 shrink-0">
      <div class="bg-surface border border-border rounded-xl p-4 space-y-4 sticky top-6">
        <h4 class="text-sm font-semibold text-heading">
          Documentation
        </h4>
        <div v-for="group in groupedDocs" :key="group.key" class="space-y-2">
          <h5 class="text-xs font-semibold text-text-muted uppercase tracking-wide">
            {{ group.label }}
          </h5>
          <a
            v-for="link in group.links"
            :key="link.url"
            :href="link.url"
            target="_blank"
            rel="noopener noreferrer"
            class="flex items-center gap-2 text-sm text-primary hover:text-primary-hover underline-offset-2 hover:underline transition-colors duration-150"
          >
            <ExternalLink class="w-3.5 h-3.5 shrink-0 text-text-muted" />
            {{ link.title }}
          </a>
        </div>
      </div>
    </div>
  </div>
</template>
