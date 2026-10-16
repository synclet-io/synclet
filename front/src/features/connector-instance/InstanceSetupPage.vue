<script setup lang="ts">
import type { ExternalDocumentationUrl, ManagedConnector } from '@entities/connector'
import type { InstanceKind } from './instanceKind'
import { useConnectorSpec, useManagedConnectors } from '@entities/connector'
import { useConnectorTaskResult } from '@entities/connector-task'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SButton, SCard, SConfirmDialog, SEmptyState, SInput, SSkeleton } from '@shared/ui'
import { ExternalLink, PackageOpen } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { onBeforeRouteLeave, useRouter } from 'vue-router'
import InstanceConfigField from './InstanceConfigField.vue'
import { INSTANCE_KINDS, useCreateInstance, useTestInstanceConnection } from './instanceKind'
import { useConfigEditor } from './useConfigEditor'

const props = defineProps<{
  kind: InstanceKind
}>()

const meta = INSTANCE_KINDS[props.kind]

const router = useRouter()
const step = ref(1)
const selectedConnector = ref<ManagedConnector | null>(null)
const name = ref('')
const config = useConfigEditor()
const error = ref('')
const search = ref('')
const created = ref(false)
const resolveLeave = ref<((leave: boolean) => void) | null>(null)

const { data: allConnectors, isLoading: connectorsLoading, error: connectorsError } = useManagedConnectors()
const connectors = computed(() => (allConnectors.value ?? []).filter(c => c.connectorType === props.kind))

const selectedConnectorId = computed(() => selectedConnector.value?.id)
const { data: specResult, isLoading: specLoading, error: specError } = useConnectorSpec(selectedConnectorId)
const externalDocs = computed<ExternalDocumentationUrl[]>(() => specResult.value?.externalDocumentationUrls ?? [])

const createMutation = useCreateInstance(props.kind)
const testMutation = useTestInstanceConnection(props.kind)

const testTaskId = ref<string | null>(null)
const { data: testTaskResult, polling: testPolling, error: testPollError, timedOut: testTimedOut } = useConnectorTaskResult(testTaskId)

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

const testError = computed(() => {
  if (testPollError.value)
    return getErrorMessage(testPollError.value)
  if (testTimedOut.value)
    return 'The connection test is taking longer than expected. Try again later.'
  return ''
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

const connectionSpec = computed(() => (specResult.value?.spec as any)?.connectionSpecification ?? null)

const hasUnsavedChanges = computed(() =>
  step.value === 2 && (!!name.value || Object.keys(config.values.value).length > 0),
)

onBeforeRouteLeave(() => {
  if (created.value || !hasUnsavedChanges.value)
    return true
  return new Promise<boolean>((resolve) => {
    resolveLeave.value = resolve
  })
})

function settleLeave(leave: boolean) {
  resolveLeave.value?.(leave)
  resolveLeave.value = null
}

function toggleJson() {
  error.value = config.toggleJson() ? '' : 'Invalid JSON'
}

function selectConnector(c: ManagedConnector) {
  selectedConnector.value = c
  config.values.value = {}
  testTaskId.value = null
  error.value = ''
  step.value = 2
}

async function handleCreate() {
  if (!selectedConnector.value)
    return
  error.value = ''
  testTaskId.value = null
  try {
    const instance = await createMutation.create({
      name: name.value,
      managedConnectorId: selectedConnector.value.id,
      config: config.resolve(),
    })
    created.value = true
    if (instance)
      router.push({ name: meta.routes.detail, params: { id: instance.id } })
    else
      router.push({ name: meta.routes.list })
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e)
  }
}

async function handleTestConnection() {
  if (!selectedConnector.value)
    return
  testTaskId.value = null
  error.value = ''
  try {
    const { taskId } = await testMutation.mutateAsync({
      managedConnectorId: selectedConnector.value.id,
      config: config.resolve(),
    })
    testTaskId.value = taskId
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e)
  }
}

const filteredConnectors = computed(() => {
  if (!search.value)
    return connectors.value
  const q = search.value.toLowerCase()
  return connectors.value.filter(c => c.name.toLowerCase().includes(q))
})
</script>

<template>
  <PageHeader :title="`Add ${meta.title}`" :description="meta.setupDescription" />

  <SAlert v-if="error" variant="danger" class="mb-4" dismissible @dismiss="error = ''">
    {{ error }}
  </SAlert>

  <div v-if="step === 1">
    <SAlert v-if="connectorsError" variant="danger" class="mb-4">
      Failed to load connectors: {{ getErrorMessage(connectorsError) }}
    </SAlert>
    <div v-else-if="connectorsLoading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
      <SSkeleton v-for="i in 6" :key="i" variant="rect" height="72px" />
    </div>
    <SEmptyState
      v-else-if="connectors.length === 0"
      :icon="PackageOpen"
      :title="`No ${meta.noun} connectors installed yet`"
      description="Connectors arrive from your repositories. If this workspace was just created, the first repository sync may still be running."
    >
      <SButton :to="{ name: 'settings-connectors' }" variant="secondary" size="sm">
        Manage connectors
      </SButton>
    </SEmptyState>
    <template v-else>
      <SInput v-model="search" :placeholder="`Search ${meta.noun} connectors...`" class="mb-4" />
      <p v-if="filteredConnectors.length === 0" class="text-sm text-text-muted py-8 text-center">
        No {{ meta.noun }} connectors match "{{ search }}".
      </p>
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
        <button
          v-for="c in filteredConnectors" :key="c.id" class="text-left bg-surface rounded-xl border border-border p-4 hover:border-primary hover:shadow-raised transition-all duration-200"
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
    </template>
  </div>

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

        <form class="space-y-4" @submit.prevent="handleCreate">
          <SInput
            v-model="name"
            :label="`${meta.title} Name`"
            :placeholder="`My ${meta.title}`"
            :tooltip="meta.nameTooltip"
            required
          />

          <InstanceConfigField
            v-model:values="config.values.value"
            v-model:json-text="config.jsonText.value"
            :json-mode="config.jsonMode.value"
            :schema="connectionSpec"
            :loading="specLoading"
            :load-error="specError"
            @toggle="toggleJson"
          />

          <SAlert v-if="testPolling" variant="info">
            Testing connection...
          </SAlert>
          <SAlert v-else-if="testError" variant="danger">
            {{ testError }}
          </SAlert>
          <SAlert v-else-if="testResult" :variant="testResult.success ? 'success' : 'danger'">
            {{ testResult.message }}
          </SAlert>

          <div class="flex gap-3 pt-2">
            <SButton variant="secondary" type="button" @click="step = 1">
              Back
            </SButton>
            <SButton variant="secondary" type="button" :loading="testMutation.isPending.value || testPolling" :disabled="specLoading" @click="handleTestConnection">
              Test Connection
            </SButton>
            <SButton type="submit" :loading="createMutation.isPending.value" :disabled="!name || specLoading">
              {{ createMutation.isPending.value ? `Creating ${meta.noun}...` : `Create ${meta.title}` }}
            </SButton>
          </div>
        </form>
      </SCard>
    </div>

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

  <SConfirmDialog
    :open="!!resolveLeave"
    title="Leave without creating?"
    :message="`The ${meta.noun} has not been created yet. Leaving discards the name and configuration you entered.`"
    confirm-text="Leave"
    cancel-text="Stay"
    @confirm="settleLeave(true)"
    @cancel="settleLeave(false)"
  />
</template>
