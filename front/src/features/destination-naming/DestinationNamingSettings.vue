<script setup lang="ts">
import type { NamespaceDefinition } from '@entities/connection'
import type { NamespaceMode, StreamName } from './destinationName'
import { SInput, SSelect } from '@shared/ui'
import { computed, ref, watch } from 'vue'
import { destinationName, formatStreamName, namespaceModeFromSettings, namespaceSettingsFromMode } from './destinationName'

const props = withDefaults(defineProps<{
  namespaceDefinition: NamespaceDefinition
  customNamespaceFormat: string
  streamPrefix: string
  sampleStream?: StreamName
}>(), {
  sampleStream: () => ({ namespace: 'public', name: 'users' }),
})

const emit = defineEmits<{
  'update:namespaceDefinition': [value: NamespaceDefinition]
  'update:customNamespaceFormat': [value: string]
  'update:streamPrefix': [value: string]
}>()

const modeOptions = [
  { label: 'Source namespace', value: 'source' },
  { label: 'Source namespace with a prefix', value: 'source_prefix' },
  { label: 'Destination default', value: 'destination' },
  { label: 'Custom format', value: 'custom' },
]

const mode = ref<NamespaceMode>('source')
const namespacePrefix = ref('')

watch(() => [props.namespaceDefinition, props.customNamespaceFormat] as const, ([def, format]) => {
  const parsed = namespaceModeFromSettings(def, format)
  if (mode.value === 'custom' && def === 'custom')
    return
  mode.value = parsed.mode
  namespacePrefix.value = parsed.namespacePrefix
}, { immediate: true })

function emitNamespace() {
  const next = namespaceSettingsFromMode(mode.value, namespacePrefix.value, props.customNamespaceFormat)
  if (next.namespaceDefinition !== props.namespaceDefinition)
    emit('update:namespaceDefinition', next.namespaceDefinition)
  if (next.customNamespaceFormat !== props.customNamespaceFormat)
    emit('update:customNamespaceFormat', next.customNamespaceFormat)
}

function onMode(v: string | number) {
  mode.value = v as NamespaceMode
  emitNamespace()
}

function onNamespacePrefix(v: string | number) {
  namespacePrefix.value = String(v).split('$').join('')
  emitNamespace()
}

const example = computed(() => {
  const target = destinationName(props.sampleStream, {
    namespaceDefinition: props.namespaceDefinition,
    customNamespaceFormat: props.customNamespaceFormat,
    streamPrefix: props.streamPrefix,
  })
  return { from: formatStreamName(props.sampleStream), to: formatStreamName(target), defaultNamespace: !target.namespace }
})
</script>

<template>
  <div class="space-y-5">
    <fieldset class="space-y-4">
      <legend class="text-sm font-semibold text-heading mb-3">
        Destination namespace
      </legend>

      <SSelect
        :model-value="mode"
        label="Namespace mode"
        :options="modeOptions"
        tooltip="The schema or dataset streams land in: the source's own namespace, the source namespace with a prefix, the destination's default, or a custom template."
        @update:model-value="onMode"
      />

      <SInput
        v-if="mode === 'source_prefix'"
        :model-value="namespacePrefix"
        label="Namespace prefix"
        placeholder="e.g. prod_"
        hint="Prepended to the source namespace, e.g. public becomes prod_public"
        @update:model-value="onNamespacePrefix"
      />

      <SInput
        v-if="mode === 'custom'"
        :model-value="customNamespaceFormat"
        label="Custom namespace format"
        placeholder="e.g. staging_${SOURCE_NAMESPACE}"
        hint="Use ${SOURCE_NAMESPACE} to include the source namespace"
        @update:model-value="emit('update:customNamespaceFormat', String($event))"
      />
    </fieldset>

    <SInput
      :model-value="streamPrefix"
      label="Table name prefix"
      placeholder="e.g. production_"
      hint="Prepended to every destination table or stream name. It does not change the namespace."
      @update:model-value="emit('update:streamPrefix', String($event))"
    />

    <div data-testid="destination-name-example" class="rounded-lg border border-border bg-surface-raised px-3 py-2 text-sm">
      <span class="text-text-muted">Example:</span>
      <code class="ml-1 font-mono text-heading">{{ example.from }} → {{ example.to }}</code>
      <span v-if="example.defaultNamespace" class="ml-1 text-text-muted">(in the destination's default namespace)</span>
    </div>
  </div>
</template>
