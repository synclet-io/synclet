<script setup lang="ts">
import type { InstanceKind } from './instanceKind'
import { useConnectorSpec } from '@entities/connector'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SButton, SCard, SInput, SSkeleton, useToast } from '@shared/ui'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import InstanceConfigField from './InstanceConfigField.vue'
import { INSTANCE_KINDS, useInstance, useUpdateInstance } from './instanceKind'
import { useConfigEditor } from './useConfigEditor'

const props = defineProps<{
  kind: InstanceKind
}>()

const meta = INSTANCE_KINDS[props.kind]

const route = useRoute()
const router = useRouter()
const instanceId = computed(() => route.params.id as string)
const { data: instance, isLoading: instanceLoading } = useInstance(props.kind, instanceId)
const updateMutation = useUpdateInstance(props.kind)
const toast = useToast()

const managedConnectorId = computed(() => instance.value?.managedConnectorId)
const { data: spec, isLoading: specLoading, error: specLoadError } = useConnectorSpec(managedConnectorId)

const name = ref('')
const config = useConfigEditor()
const saving = ref(false)
const error = ref('')

const connectionSpec = computed(() => (spec.value?.spec as any)?.connectionSpecification ?? null)

const specWarning = computed(() => {
  if (!instance.value)
    return ''
  if (!instance.value.managedConnectorId)
    return 'No managed connector linked. You can edit as JSON.'
  if (specLoadError.value)
    return `Failed to load connector spec: ${getErrorMessage(specLoadError.value)}`
  if (spec.value && !spec.value.spec)
    return 'Connector spec has no connectionSpecification. You can edit as JSON.'
  return ''
})

watch(() => instance.value?.id, () => {
  if (!instance.value)
    return
  name.value = instance.value.name
  config.reset(instance.value.config)
}, { immediate: true })

watch(specWarning, (warning) => {
  if (warning)
    config.jsonMode.value = true
}, { immediate: true })

function toggleJson() {
  error.value = config.toggleJson() ? '' : 'Invalid JSON'
}

async function handleSave() {
  if (!instance.value)
    return
  saving.value = true
  error.value = ''
  try {
    await updateMutation.mutateAsync({
      id: instance.value.id,
      name: name.value,
      config: config.resolve(),
    })
    toast.success(`${meta.title} updated`)
    router.push({ name: meta.routes.detail, params: { id: instance.value.id } })
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e)
  }
  finally {
    saving.value = false
  }
}

function handleDiscard() {
  if (instance.value)
    router.push({ name: meta.routes.detail, params: { id: instance.value.id } })
}
</script>

<template>
  <div v-if="instanceLoading || specLoading">
    <SSkeleton class="h-8 w-48 mb-4" />
    <SSkeleton variant="rect" class="h-64" />
  </div>
  <div v-else-if="instance">
    <PageHeader
      :title="`Edit ${meta.title}`"
      :description="instance.name"
      :back-label="instance.name"
      :back-to="{ name: meta.routes.detail, params: { id: instance.id } }"
    />

    <SAlert v-if="error" variant="danger" class="mb-4" dismissible @dismiss="error = ''">
      {{ error }}
    </SAlert>
    <SAlert v-if="specWarning" variant="warning" class="mb-4">
      {{ specWarning }}
    </SAlert>

    <SCard class="max-w-2xl">
      <form class="space-y-4" @submit.prevent="handleSave">
        <SInput v-model="name" :label="`${meta.title} Name`" :placeholder="`My ${meta.title}`" required />

        <InstanceConfigField
          v-model:values="config.values.value"
          v-model:json-text="config.jsonText.value"
          :json-mode="config.jsonMode.value"
          :schema="connectionSpec"
          @toggle="toggleJson"
        />

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
