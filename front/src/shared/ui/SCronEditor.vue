<script setup lang="ts">
import { describeCron, validateCron } from '@shared/lib/cron'
import { computed, ref, useId } from 'vue'

const props = defineProps<{
  modelValue: string
  label?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const presets = [
  { label: 'Every 5 minutes', value: '*/5 * * * *' },
  { label: 'Every 15 minutes', value: '*/15 * * * *' },
  { label: 'Every 30 minutes', value: '*/30 * * * *' },
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Every 6 hours', value: '0 */6 * * *' },
  { label: 'Every 12 hours', value: '0 */12 * * *' },
  { label: 'Daily (midnight)', value: '0 0 * * *' },
  { label: 'Daily (6 AM)', value: '0 6 * * *' },
  { label: 'Weekly (Sunday midnight)', value: '0 0 * * 0' },
  { label: 'Manual', value: '' },
]

// Determine initial mode from the initial value
const isPreset = presets.some(p => p.value === props.modelValue)
const mode = ref<string>(isPreset ? props.modelValue : 'custom')

const selectId = useId()
const customId = `${selectId}-custom`
const messageId = `${selectId}-message`

const isCustom = computed(() => mode.value === 'custom')
const cronError = computed(() => validateCron(props.modelValue))
const cronDescription = computed(() => describeCron(props.modelValue))
const showError = computed(() => !!props.modelValue && !!cronError.value)

function onPresetChange(e: Event) {
  const val = (e.target as HTMLSelectElement).value
  mode.value = val
  if (val !== 'custom') {
    emit('update:modelValue', val)
  }
}

function onCustomInput(e: Event) {
  emit('update:modelValue', (e.target as HTMLInputElement).value)
}
</script>

<template>
  <div>
    <label v-if="label" :for="selectId" class="block text-sm font-medium text-text-secondary mb-1.5">{{ label }}</label>
    <div class="space-y-2">
      <select
        :id="selectId"
        :value="mode"
        :aria-label="label ? undefined : 'Schedule'"
        class="w-full px-3 py-2 text-sm rounded-lg border border-border bg-surface text-text-primary focus:ring-1 focus:ring-primary/20 focus:border-primary"
        @change="onPresetChange"
      >
        <option v-for="p in presets" :key="p.value" :value="p.value">
          {{ p.label }}
        </option>
        <option value="custom">
          Custom
        </option>
      </select>

      <input
        v-if="isCustom"
        :id="customId"
        type="text"
        aria-label="Cron expression"
        :aria-invalid="showError || undefined"
        :aria-describedby="showError || cronDescription ? messageId : undefined"
        :value="modelValue"
        placeholder="0 */6 * * *"
        class="w-full px-3 py-2 text-sm rounded-lg border border-border bg-surface text-text-primary focus:ring-1 focus:ring-primary/20 focus:border-primary"
        :class="{ 'border-danger': showError }"
        @input="onCustomInput"
      >

      <template v-if="isCustom">
        <p v-if="showError" :id="messageId" class="text-xs text-danger">
          {{ cronError }}
        </p>
        <p v-else-if="cronDescription" :id="messageId" class="text-xs text-text-muted">
          {{ cronDescription }}
        </p>
      </template>
    </div>
  </div>
</template>
