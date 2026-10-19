<script setup lang="ts">
import type { Column, TransformIssue } from './columns'
import type { StepDraft } from './model'
import { AlertCircle, AlertTriangle, ChevronDown, ChevronUp, GripVertical, Trash2 } from 'lucide-vue-next'
import { MAPPER_LABELS } from './model'
import StepForm from './StepForm.vue'

defineProps<{
  step: StepDraft
  index: number
  count: number
  columns: Column[]
  issues: TransformIssue[]
  resultTypes: Record<string, string>
  dragging?: boolean
}>()

const emit = defineEmits<{
  update: [step: StepDraft]
  move: [delta: number]
  remove: []
  grab: [event: PointerEvent]
}>()

function onGrab(e: PointerEvent) {
  if (e.button !== 0)
    return
  e.preventDefault()
  emit('grab', e)
}

const iconButton = 'p-1 rounded text-text-muted hover:text-heading hover:bg-surface-hover disabled:opacity-30 disabled:pointer-events-none'
</script>

<template>
  <article
    data-testid="step-card"
    :data-kind="step.kind"
    class="rounded-lg border bg-surface p-3 space-y-2"
    :class="[
      issues.some(i => i.severity === 'error') ? 'border-danger/60' : 'border-border',
      dragging ? 'shadow-lg ring-1 ring-primary' : '',
    ]"
  >
    <header class="flex items-center gap-2">
      <button
        type="button"
        data-testid="step-drag"
        tabindex="-1"
        aria-label="Drag to reorder"
        title="Drag to reorder"
        class="-m-1 p-1 rounded cursor-grab touch-none text-text-muted hover:text-heading"
        :class="dragging ? 'cursor-grabbing' : ''"
        @pointerdown="onGrab"
      >
        <GripVertical class="w-4 h-4" />
      </button>
      <span class="text-xs font-semibold text-text-muted tabular-nums">{{ index + 1 }}</span>
      <span class="text-sm font-semibold text-heading">{{ MAPPER_LABELS[step.kind] }}</span>
      <div class="ml-auto flex items-center gap-0.5">
        <button type="button" data-testid="step-up" :class="iconButton" :disabled="index === 0" aria-label="Move step up" @click="emit('move', -1)">
          <ChevronUp class="w-4 h-4" />
        </button>
        <button type="button" data-testid="step-down" :class="iconButton" :disabled="index === count - 1" aria-label="Move step down" @click="emit('move', 1)">
          <ChevronDown class="w-4 h-4" />
        </button>
        <button type="button" data-testid="step-remove" class="hover:!text-danger" :class="[iconButton]" aria-label="Remove step" @click="emit('remove')">
          <Trash2 class="w-4 h-4" />
        </button>
      </div>
    </header>

    <StepForm :step="step" :columns="columns" :result-types="resultTypes" @update="emit('update', $event)" />

    <ul v-if="issues.length > 0" class="space-y-1">
      <li
        v-for="(issue, i) in issues"
        :key="i"
        data-testid="step-issue"
        :data-severity="issue.severity"
        class="flex items-start gap-1.5 text-xs"
        :class="issue.severity === 'error' ? 'text-danger' : 'text-warning'"
      >
        <AlertCircle v-if="issue.severity === 'error'" class="w-3.5 h-3.5 mt-px shrink-0" aria-hidden="true" />
        <AlertTriangle v-else class="w-3.5 h-3.5 mt-px shrink-0" aria-hidden="true" />
        <span class="whitespace-pre-wrap break-words">{{ issue.message }}</span>
      </li>
    </ul>
  </article>
</template>
