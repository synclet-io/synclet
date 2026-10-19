<script setup lang="ts">
import type { Column } from './columns'
import type { CastOptions } from './model'
import { computed, useId } from 'vue'
import CelInput from './CelInput.vue'
import { inputClass, labelClass } from './styles'

const props = defineProps<{
  spec: CastOptions
  columns: Column[]
}>()

const emit = defineEmits<{
  update: [spec: CastOptions]
}>()

const formatPresets = ['iso8601', 'unix', 'unix_ms', 'dd.MM.yyyy HH:mm', 'dd.MM.yyyy', 'yyyy-MM-dd HH:mm:ss', 'MM/dd/yyyy']
const formatListId = useId()

const isTime = computed(() => props.spec.to === 'timestamp' || props.spec.to === 'date')
const isNumber = computed(() => props.spec.to === 'integer' || props.spec.to === 'number')
const isJson = computed(() => props.spec.to === 'object' || props.spec.to === 'array')
const hasFormat = computed(() => isTime.value || props.spec.to === 'string')

function set(patch: Partial<CastOptions>) {
  emit('update', { ...props.spec, ...patch })
}

function list(values: string[] | undefined): string {
  return (values ?? []).join(', ')
}

function parseList(text: string): string[] {
  return text.split(',').map(v => v.trim()).filter(Boolean)
}
</script>

<template>
  <div class="grid gap-2 sm:grid-cols-2">
    <template v-if="hasFormat">
      <label class="block">
        <span :class="labelClass">Format</span>
        <input
          data-testid="cast-format"
          :class="inputClass"
          :list="formatListId"
          :value="spec.format ?? ''"
          placeholder="iso8601"
          @input="set({ format: ($event.target as HTMLInputElement).value })"
        >
        <datalist :id="formatListId">
          <option v-for="f in formatPresets" :key="f" :value="f" />
        </datalist>
      </label>
      <label v-if="isTime" class="block">
        <span :class="labelClass">Timezone for values without one</span>
        <input
          data-testid="cast-timezone"
          :class="inputClass"
          :value="spec.timezone ?? ''"
          placeholder="UTC"
          @input="set({ timezone: ($event.target as HTMLInputElement).value })"
        >
      </label>
    </template>

    <template v-if="isNumber">
      <label class="block">
        <span :class="labelClass">Decimal separator</span>
        <select
          data-testid="cast-decimal-separator"
          :class="inputClass"
          :value="spec.decimalSeparator ?? ''"
          @change="set({ decimalSeparator: ($event.target as HTMLSelectElement).value })"
        >
          <option value="">. (default)</option>
          <option value=",">,</option>
        </select>
      </label>
      <label class="block">
        <span :class="labelClass">Thousands separator</span>
        <input
          data-testid="cast-thousands-separator"
          :class="inputClass"
          :value="spec.thousandsSeparator ?? ''"
          placeholder="none"
          @input="set({ thousandsSeparator: ($event.target as HTMLInputElement).value })"
        >
      </label>
      <label class="block">
        <span :class="labelClass">Strip prefix</span>
        <input
          data-testid="cast-strip-prefix"
          :class="inputClass"
          :value="spec.stripPrefix ?? ''"
          placeholder="$"
          @input="set({ stripPrefix: ($event.target as HTMLInputElement).value })"
        >
      </label>
      <label class="block">
        <span :class="labelClass">Strip suffix</span>
        <input
          data-testid="cast-strip-suffix"
          :class="inputClass"
          :value="spec.stripSuffix ?? ''"
          placeholder="%"
          @input="set({ stripSuffix: ($event.target as HTMLInputElement).value })"
        >
      </label>
      <label class="flex items-center gap-2 text-sm text-heading">
        <input
          data-testid="cast-trim"
          type="checkbox"
          :checked="!!spec.trim"
          @change="set({ trim: ($event.target as HTMLInputElement).checked })"
        >
        Trim spaces
      </label>
    </template>

    <template v-if="spec.to === 'boolean'">
      <label class="block">
        <span :class="labelClass">True values</span>
        <input
          data-testid="cast-true-values"
          :class="inputClass"
          :value="list(spec.trueValues)"
          placeholder="yes, 1, Y"
          @change="set({ trueValues: parseList(($event.target as HTMLInputElement).value) })"
        >
      </label>
      <label class="block">
        <span :class="labelClass">False values</span>
        <input
          data-testid="cast-false-values"
          :class="inputClass"
          :value="list(spec.falseValues)"
          placeholder="no, 0, N"
          @change="set({ falseValues: parseList(($event.target as HTMLInputElement).value) })"
        >
      </label>
    </template>

    <p v-if="isJson" class="text-xs text-text-muted sm:col-span-2">
      Parses a JSON string.
    </p>

    <label class="block">
      <span :class="labelClass">On error</span>
      <select
        data-testid="cast-on-error"
        :class="inputClass"
        :value="spec.onError ?? ''"
        @change="set({ onError: ($event.target as HTMLSelectElement).value })"
      >
        <option value="">As the step</option>
        <option value="fail">Fail the sync</option>
        <option value="null">Write null</option>
        <option value="skip">Skip the record</option>
      </select>
    </label>
    <label class="block">
      <span :class="labelClass">Source expression</span>
      <CelInput
        data-testid="cast-source"
        class="w-full"
        :model-value="spec.source ?? ''"
        :columns="columns"
        placeholder="the field itself"
        aria-label="Source expression"
        @update:model-value="set({ source: $event })"
      />
    </label>
  </div>
</template>
