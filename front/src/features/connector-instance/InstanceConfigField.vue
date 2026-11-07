<script setup lang="ts">
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SSkeleton, STextarea } from '@shared/ui'
import JsonSchemaForm from '@shared/ui/JsonSchemaForm.vue'
import { ref } from 'vue'

defineProps<{
  schema: Record<string, any> | null
  jsonMode: boolean
  loading?: boolean
  loadError?: unknown
}>()

defineEmits<{
  toggle: []
}>()

const values = defineModel<Record<string, any>>('values', { required: true })
const jsonText = defineModel<string>('jsonText', { required: true })

const form = ref<InstanceType<typeof JsonSchemaForm> | null>(null)

/** Validates the form, marking every invalid field; true when there is no form to validate. */
function validate(): boolean {
  return !form.value || form.value.validate().length === 0
}

defineExpose({ validate })
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-2">
      <label class="text-sm font-medium text-heading">Configuration</label>
      <button type="button" class="text-xs text-primary hover:text-primary-hover" @click="$emit('toggle')">
        {{ jsonMode ? 'Switch to Form' : 'Edit as JSON' }}
      </button>
    </div>
    <SSkeleton v-if="loading" :count="4" height="36px" />
    <SAlert v-else-if="loadError" variant="danger">
      Failed to load the connector spec: {{ getErrorMessage(loadError) }}
    </SAlert>
    <JsonSchemaForm v-else-if="!jsonMode && schema" ref="form" v-model="values" :schema="schema" />
    <STextarea v-else v-model="jsonText" :rows="10" placeholder="{}" monospace aria-label="Configuration JSON" />
  </div>
</template>
