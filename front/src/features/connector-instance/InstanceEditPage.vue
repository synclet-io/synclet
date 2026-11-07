<script setup lang="ts">
import type { InstanceKind } from './instanceKind'
import { useConnections } from '@entities/connection'
import { useConnectorSpec } from '@entities/connector'
import { DEFAULT_EXECUTOR_GROUP, ExecutorGroupField, groupOf, useExecutorGroups } from '@features/executor-group'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SButton, SCard, SInput, SModal, SSkeleton, useToast } from '@shared/ui'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import InstanceConfigField from './InstanceConfigField.vue'
import { INSTANCE_KINDS, useDuplicateInstance, useInstance, useUpdateInstance } from './instanceKind'
import { INVALID_CONFIG_MESSAGE, useConfigEditor } from './useConfigEditor'

const props = defineProps<{
  kind: InstanceKind
}>()

const meta = INSTANCE_KINDS[props.kind]

const route = useRoute()
const router = useRouter()
const instanceId = computed(() => route.params.id as string)
const { data: instance, isLoading: instanceLoading } = useInstance(props.kind, instanceId)
const updateMutation = useUpdateInstance(props.kind)
const duplicateMutation = useDuplicateInstance(props.kind)
const toast = useToast()

const managedConnectorId = computed(() => instance.value?.managedConnectorId)
const { data: spec, isLoading: specLoading, error: specLoadError } = useConnectorSpec(managedConnectorId)

const name = ref('')
const group = ref(DEFAULT_EXECUTOR_GROUP)
const config = useConfigEditor()
const configField = ref<InstanceType<typeof InstanceConfigField> | null>(null)
const saving = ref(false)
const error = ref('')

const { data: connections, isLoading: connectionsLoading, error: connectionsError } = useConnections({
  sourceId: computed(() => props.kind === 'source' ? instanceId.value : ''),
  destinationId: computed(() => props.kind === 'destination' ? instanceId.value : ''),
})
const hasConnections = computed(() => (connections.value?.items.length ?? 0) > 0)
const groupLocked = computed(() => hasConnections.value || connectionsLoading.value || !!connectionsError.value)

const { groups } = useExecutorGroups()
const duplicateOpen = ref(false)
const duplicateGroup = ref('')
const duplicateName = ref('')
const duplicateError = ref('')

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
  group.value = groupOf(instance.value)
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
  error.value = ''
  if (configField.value && !configField.value.validate()) {
    error.value = INVALID_CONFIG_MESSAGE
    return
  }
  saving.value = true
  try {
    await updateMutation.mutateAsync({
      id: instance.value.id,
      name: name.value,
      config: config.resolve(),
      ...(group.value !== groupOf(instance.value) ? { executorGroup: group.value } : {}),
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

function openDuplicate() {
  if (!instance.value)
    return
  const current = groupOf(instance.value)
  duplicateGroup.value = groups.value.find(g => g.name !== current)?.name ?? ''
  duplicateName.value = `${instance.value.name} (copy)`
  duplicateError.value = ''
  duplicateOpen.value = true
}

async function handleDuplicate() {
  if (!instance.value || !duplicateGroup.value)
    return
  duplicateError.value = ''
  try {
    const copy = await duplicateMutation.duplicate({
      id: instance.value.id,
      executorGroup: duplicateGroup.value,
      name: duplicateName.value,
    })
    duplicateOpen.value = false
    toast.success(`${meta.title} duplicated`)
    if (copy)
      router.push({ name: meta.routes.detail, params: { id: copy.id } })
  }
  catch (e: unknown) {
    duplicateError.value = getErrorMessage(e)
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

        <ExecutorGroupField v-model="group" :disabled="groupLocked">
          <div v-if="hasConnections" class="flex items-center justify-between gap-3">
            <p class="text-xs text-text-muted">
              Used by connections, so the group cannot change. Duplicate it into another group instead.
            </p>
            <SButton variant="secondary" size="sm" type="button" @click="openDuplicate">
              Duplicate to group
            </SButton>
          </div>
        </ExecutorGroupField>

        <InstanceConfigField
          ref="configField"
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

    <SModal :open="duplicateOpen" title="Duplicate to group" @close="duplicateOpen = false">
      <div class="space-y-4">
        <SAlert v-if="duplicateError" variant="danger">
          {{ duplicateError }}
        </SAlert>
        <ExecutorGroupField v-model="duplicateGroup" :exclude="group" />
        <SInput v-model="duplicateName" :label="`${meta.title} Name`" required />
      </div>
      <template #footer>
        <SButton variant="secondary" type="button" @click="duplicateOpen = false">
          Cancel
        </SButton>
        <SButton type="button" :loading="duplicateMutation.isPending.value" :disabled="!duplicateGroup || !duplicateName" @click="handleDuplicate">
          Duplicate
        </SButton>
      </template>
    </SModal>
  </div>
</template>
