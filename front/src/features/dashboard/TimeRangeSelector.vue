<script setup lang="ts">
import type { TimeRange } from '@entities/stats'
import type { Tab } from '@shared/ui'
import { STabs } from '@shared/ui'

defineProps<{
  modelValue: TimeRange
}>()

const emit = defineEmits<{
  'update:modelValue': [value: TimeRange]
}>()

const tabs: Tab[] = (['24h', '7d', '30d'] as const).map(range => ({ name: range, value: range }))
</script>

<template>
  <STabs
    :model-value="modelValue"
    :tabs="tabs"
    variant="pills"
    aria-label="Time range"
    class="!mb-0"
    @update:model-value="emit('update:modelValue', $event as TimeRange)"
  />
</template>
