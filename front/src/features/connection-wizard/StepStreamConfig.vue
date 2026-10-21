<script setup lang="ts">
import type { ConfiguredStream } from '@entities/connection'
import type { DiscoveredCatalog, StreamRow } from '@features/stream-config/catalogStreams'
import { useConnectorTaskResult } from '@entities/connector-task'
import { discoverSourceSchema, getSourceCatalog } from '@entities/source'
import { configuredCatalogInput, parseStreams, toConfiguredStreams } from '@features/stream-config/catalogStreams'
import StreamConfigEditor from '@features/stream-config/StreamConfigEditor.vue'
import { streamErrorSummary } from '@features/stream-config/useStreamValidation'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SButton, SEmptyState, SSkeleton } from '@shared/ui'
import { RefreshCw } from 'lucide-vue-next'
import { onMounted, ref, watch } from 'vue'

const props = defineProps<{
  sourceId: string
  connectionName: string
  streams: ConfiguredStream[]
  discoveredCatalog: Record<string, unknown> | null
}>()

const emit = defineEmits<{
  'update:streams': [streams: ConfiguredStream[]]
  'update:discoveredCatalog': [catalog: Record<string, unknown> | null]
  'skip': []
}>()

const localStreams = ref<StreamRow[]>([])
const editor = ref<InstanceType<typeof StreamConfigEditor> | null>(null)
const discovering = ref(false)
const error = ref('')

const discoverTaskId = ref<string | null>(null)
const DISCOVERY_FAILED = 'Schema discovery failed. Check that the source connector is running and try again.'
const { data: discoverTaskResult, error: discoverPollError, timedOut: discoverTimedOut } = useConnectorTaskResult(discoverTaskId)

function stopDiscovery(message?: string) {
  if (message)
    error.value = message
  discovering.value = false
  discoverTaskId.value = null
}

watch(discoverTaskResult, (task) => {
  if (!task)
    return
  if (task.status === 'completed') {
    const cat = task.discoverResult?.catalog
    if (cat) {
      loadStreams(cat)
      emit('update:discoveredCatalog', cat)
    }
    stopDiscovery()
  }
  else if (task.status === 'failed') {
    stopDiscovery(task.errorMessage || DISCOVERY_FAILED)
  }
})

watch(discoverPollError, (e) => {
  if (e && discoverTaskId.value)
    stopDiscovery(getErrorMessage(e))
})

watch(discoverTimedOut, (timedOut) => {
  if (timedOut && discoverTaskId.value)
    stopDiscovery('Schema discovery is taking longer than expected. Check the source connector and try again.')
})

// Streams start turned off unless the wizard already holds a configuration for them.
function loadStreams(cat: Record<string, unknown>, configured: ConfiguredStream[] = []) {
  const rows = parseStreams(cat as unknown as DiscoveredCatalog, configuredCatalogInput(configured))
  localStreams.value = configured.length > 0 ? rows : rows.map(r => ({ ...r, enabled: false }))
}

onMounted(async () => {
  if (props.discoveredCatalog) {
    loadStreams(props.discoveredCatalog, props.streams)
    return
  }
  try {
    const cached = await getSourceCatalog(props.sourceId)
    if (cached && cached.catalog && Object.keys(cached.catalog).length > 0) {
      loadStreams(cached.catalog as Record<string, unknown>)
      emit('update:discoveredCatalog', cached.catalog as Record<string, unknown>)
    }
  }
  catch {
    // No cached catalog available, will show empty state / discover button
  }
})

watch(localStreams, (rows) => {
  emit('update:streams', toConfiguredStreams(rows))
}, { deep: true })

/** Validates the enabled streams and reports whether the wizard may go on. */
function validate(): boolean {
  const errors = editor.value?.validate() ?? []
  if (errors.length > 0) {
    error.value = `Fix ${errors.length} issue(s) before continuing: ${streamErrorSummary(errors)}`
    return false
  }
  error.value = ''
  return true
}

defineExpose({ validate })

async function handleDiscoverSchema() {
  discovering.value = true
  error.value = ''
  discoverTaskId.value = null
  try {
    const { taskId } = await discoverSourceSchema(props.sourceId)
    discoverTaskId.value = taskId
  }
  catch (e: unknown) {
    stopDiscovery(getErrorMessage(e) || DISCOVERY_FAILED)
  }
}
</script>

<template>
  <div>
    <SAlert v-if="error" variant="danger" class="mb-4" dismissible @dismiss="error = ''">
      {{ error }}
    </SAlert>

    <div v-if="discovering" class="space-y-3">
      <SSkeleton v-for="i in 3" :key="i" height="40px" />
    </div>

    <div v-else-if="localStreams.length === 0" class="py-8">
      <SEmptyState title="No streams discovered" description="Click Discover Schema to load available streams from the source connector.">
        <SButton :loading="discovering" @click="handleDiscoverSchema">
          <RefreshCw class="w-4 h-4" /> Discover Schema
        </SButton>
      </SEmptyState>
    </div>

    <div v-else>
      <div class="flex items-center justify-between mb-4">
        <p class="text-sm text-text-secondary">
          {{ localStreams.filter(s => s.enabled).length }} of {{ localStreams.length }} streams enabled
        </p>
        <SButton variant="secondary" size="sm" :loading="discovering" @click="handleDiscoverSchema">
          <RefreshCw class="w-4 h-4" /> Discover Schema
        </SButton>
      </div>

      <StreamConfigEditor ref="editor" v-model:streams="localStreams" :source-id="sourceId" :connection-name="connectionName" />
    </div>
  </div>
</template>
