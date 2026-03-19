<script setup lang="ts">
import { getConnectorSpec } from '@entities/connector/api'
import { useDestination, useUpdateDestination } from '@entities/destination'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SButton, SCard, SInput, SSkeleton, useToast } from '@shared/ui'
import JsonSchemaForm from '@shared/ui/JsonSchemaForm.vue'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const destinationId = computed(() => route.params.id as string)
const { data: destination, isLoading: destinationLoading } = useDestination(destinationId)
const updateDestinationMutation = useUpdateDestination()
const toast = useToast()

const spec = ref<any>(null)
const specLoading = ref(false)
const specError = ref('')
const configValues = ref<Record<string, any>>({})
const name = ref('')
const showJsonEditor = ref(false)
const jsonEditorText = ref('{}')
const saving = ref(false)
const error = ref('')

const connectionSpec = computed(() => spec.value?.spec?.connectionSpecification ?? null)

// When destination loads, look up managed connector by dockerImage to get spec
watch(() => destination.value, async (d) => {
  if (!d)
    return
  name.value = d.name
  configValues.value = { ...d.config }
  jsonEditorText.value = JSON.stringify(d.config, null, 2)

  // Find managed connector matching this destination's image
  specLoading.value = true
  specError.value = ''
  try {
    if (d.managedConnectorId) {
      const loadedSpec = await getConnectorSpec(d.managedConnectorId)
      spec.value = loadedSpec
      // Guard: if connectionSpecification key is absent, fall back to JSON editor
      if (!loadedSpec?.spec) {
        specError.value = 'Connector spec has no connectionSpecification. You can edit as JSON.'
        showJsonEditor.value = true
      }
    }
    else {
      specError.value = 'No managed connector linked. You can edit as JSON.'
      showJsonEditor.value = true
    }
  }
  catch (e: unknown) {
    specError.value = `Failed to load connector spec: ${getErrorMessage(e)}`
    showJsonEditor.value = true
  }
  finally {
    specLoading.value = false
  }
}, { immediate: true })

// Sync form values to JSON editor
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

async function handleSave() {
  if (!destination.value)
    return
  saving.value = true
  error.value = ''
  try {
    const config = showJsonEditor.value ? JSON.parse(jsonEditorText.value) : configValues.value
    await updateDestinationMutation.mutateAsync({
      id: destination.value.id,
      name: name.value,
      config: cleanConfig(config),
    })
    toast.success('Destination updated')
    router.push(`/destinations/${destination.value.id}`)
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e)
  }
  finally {
    saving.value = false
  }
}

function handleDiscard() {
  if (destination.value) {
    router.push(`/destinations/${destination.value.id}`)
  }
}
</script>

<template>
  <div v-if="destinationLoading || specLoading">
    <SSkeleton class="h-8 w-48 mb-4" />
    <SSkeleton class="h-64" />
  </div>
  <div v-else-if="destination">
    <PageHeader
      title="Edit Destination"
      :description="destination.name"
      :back-label="destination.name"
      :back-to="`/destinations/${destination.id}`"
    />

    <SAlert v-if="error" variant="danger" class="mb-4" dismissible @dismiss="error = ''">
      {{ error }}
    </SAlert>
    <SAlert v-if="specError" variant="warning" class="mb-4">
      {{ specError }}
    </SAlert>

    <SCard class="max-w-2xl">
      <form class="space-y-4" @submit.prevent="handleSave">
        <SInput v-model="name" label="Destination Name" placeholder="My Destination" required />

        <div>
          <div class="flex items-center justify-between mb-2">
            <label class="text-sm font-medium text-heading">Configuration</label>
            <button
              type="button"
              class="text-xs text-primary hover:text-primary-hover"
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

        <div class="flex gap-3 pt-2">
          <SButton variant="secondary" type="button" @click="handleDiscard">
            Discard Changes
          </SButton>
          <SButton type="submit" :loading="saving">
            {{ saving ? 'Testing connection...' : 'Save Changes' }}
          </SButton>
        </div>
      </form>
    </SCard>
  </div>
</template>
