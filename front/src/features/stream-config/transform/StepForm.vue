<script setup lang="ts">
import type { Column } from './columns'
import type { CastOptions as CastSpec, StepDraft, StepEntry } from './model'
import { Plus, X } from 'lucide-vue-next'
import { computed, useId } from 'vue'
import CastOptions from './CastOptions.vue'
import CelInput from './CelInput.vue'
import ExpressionHelp from './ExpressionHelp.vue'
import { CAST_TARGETS, newEntry } from './model'
import { fieldClass } from './styles'

const props = defineProps<{
  step: StepDraft
  columns: Column[]
  resultTypes: Record<string, string>
}>()

const emit = defineEmits<{
  update: [step: StepDraft]
}>()

const columnListId = useId()

const onErrorOptions = computed(() => {
  switch (props.step.kind) {
    case 'rename':
      return []
    case 'filter':
      return [{ value: 'skip', label: 'Skip the record' }]
    default:
      return [{ value: 'null', label: 'Write null' }, { value: 'skip', label: 'Skip the record' }]
  }
})

const fieldPlaceholder = computed(() => (props.step.kind === 'set' ? 'new or existing field' : 'field'))

function update(patch: Partial<StepDraft>) {
  emit('update', { ...props.step, ...patch })
}

function updateEntry(index: number, patch: Partial<StepEntry>) {
  update({ entries: props.step.entries.map((e, i) => (i === index ? { ...e, ...patch } : e)) })
}

function updateCast(index: number, patch: Partial<CastSpec>) {
  const entry = props.step.entries[index]
  updateEntry(index, { cast: { ...(entry.cast ?? { to: 'string' }), ...patch } })
}

function addEntry() {
  update({ entries: [...props.step.entries, newEntry(props.step.kind)] })
}

function removeEntry(index: number) {
  update({ entries: props.step.entries.filter((_, i) => i !== index) })
}

function keep(value: string): number | undefined {
  const n = Number.parseInt(value, 10)
  return Number.isNaN(n) ? undefined : n
}

function value(e: Event): string {
  return (e.target as HTMLInputElement).value
}
</script>

<template>
  <div class="space-y-2">
    <datalist :id="columnListId">
      <option v-for="c in columns" :key="c.name" :value="c.name" />
    </datalist>

    <template v-if="step.kind === 'filter'">
      <label class="block">
        <span class="flex items-center gap-1 text-xs font-medium text-text-secondary mb-1">Keep records where <ExpressionHelp /></span>
        <CelInput
          data-testid="filter-expr"
          class="w-full"
          multiline
          :model-value="step.filter ?? ''"
          :columns="columns"
          placeholder="r.status != 'test'"
          aria-label="Keep records where"
          @update:model-value="update({ filter: $event })"
        />
      </label>
      <p v-if="resultTypes['']" data-testid="result-type" class="text-xs text-text-muted font-mono">
        → {{ resultTypes[''] }}
      </p>
    </template>

    <template v-else>
      <div
        v-for="(entry, i) in step.entries"
        :key="i"
        data-testid="step-entry"
        class="rounded-md"
        :class="step.kind === 'cast' ? 'border border-border-light p-2 space-y-2' : ''"
      >
        <div class="flex flex-wrap items-center gap-2">
          <input
            data-testid="entry-field"
            class="flex-1 min-w-[8rem]" :class="[fieldClass]"
            :list="columnListId"
            :value="entry.field"
            :placeholder="fieldPlaceholder"
            aria-label="Field"
            @input="updateEntry(i, { field: value($event) })"
          >

          <template v-if="step.kind === 'rename'">
            <span class="text-text-muted text-sm" aria-hidden="true">→</span>
            <input
              data-testid="entry-target"
              class="flex-1 min-w-[8rem]" :class="[fieldClass]"
              :value="entry.target ?? ''"
              placeholder="new name"
              aria-label="New name"
              @input="updateEntry(i, { target: value($event) })"
            >
          </template>

          <template v-else-if="step.kind === 'cast'">
            <span class="text-text-muted text-sm" aria-hidden="true">to</span>
            <select
              data-testid="entry-cast-to"
              class="w-32 flex-none" :class="[fieldClass]"
              :value="entry.cast?.to ?? 'string'"
              aria-label="Target type"
              @change="updateCast(i, { to: value($event) })"
            >
              <option v-for="t in CAST_TARGETS" :key="t" :value="t">
                {{ t }}
              </option>
            </select>
          </template>

          <template v-else-if="step.kind === 'set'">
            <span class="text-text-muted text-sm" aria-hidden="true">=</span>
            <CelInput
              data-testid="entry-expr"
              class="flex-[2] min-w-[10rem]"
              :model-value="entry.expr ?? ''"
              :columns="columns"
              placeholder="'USD' or r.amount * 100.0"
              aria-label="Expression"
              @update:model-value="updateEntry(i, { expr: $event })"
            />
            <span v-if="resultTypes[entry.field]" data-testid="result-type" class="text-xs text-text-muted font-mono">
              → {{ resultTypes[entry.field] }}
            </span>
          </template>

          <template v-else-if="step.kind === 'mask'">
            <label class="flex items-center gap-1 text-xs text-text-secondary">
              first
              <input
                data-testid="mask-keep-first"
                type="number"
                min="0"
                class="w-16" :class="[fieldClass]"
                :value="entry.keepFirst ?? ''"
                @input="updateEntry(i, { keepFirst: keep(value($event)) })"
              >
            </label>
            <label class="flex items-center gap-1 text-xs text-text-secondary">
              last
              <input
                data-testid="mask-keep-last"
                type="number"
                min="0"
                class="w-16" :class="[fieldClass]"
                :value="entry.keepLast ?? ''"
                @input="updateEntry(i, { keepLast: keep(value($event)) })"
              >
            </label>
          </template>

          <button
            type="button"
            data-testid="entry-remove"
            class="p-1 rounded text-text-muted hover:text-danger hover:bg-danger-bg disabled:opacity-40"
            :disabled="step.entries.length <= 1"
            aria-label="Remove field"
            @click="removeEntry(i)"
          >
            <X class="w-3.5 h-3.5" />
          </button>
        </div>

        <details v-if="step.kind === 'cast'" class="text-sm" :open="!!entry.cast?.source || !!entry.cast?.format">
          <summary class="cursor-pointer text-xs text-text-secondary select-none">
            Cast options
            <span v-if="resultTypes[entry.field]" data-testid="result-type" class="font-mono text-text-muted">→ {{ resultTypes[entry.field] }}</span>
          </summary>
          <div class="pt-2">
            <CastOptions :spec="entry.cast ?? { to: 'string' }" :columns="columns" @update="updateEntry(i, { cast: $event })" />
          </div>
        </details>
      </div>

      <button
        type="button"
        data-testid="entry-add"
        class="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
        @click="addEntry"
      >
        <Plus class="w-3.5 h-3.5" /> Add field
      </button>
    </template>

    <label v-if="onErrorOptions.length > 0" class="flex items-center gap-2 text-xs text-text-secondary whitespace-nowrap">
      On error
      <select
        data-testid="step-on-error"
        class="w-40 h-7 text-xs" :class="[fieldClass]"
        :value="step.onError ?? ''"
        @change="update({ onError: value($event) })"
      >
        <option value="">Fail the sync (default)</option>
        <option v-for="o in onErrorOptions" :key="o.value" :value="o.value">
          {{ o.label }}
        </option>
      </select>
    </label>
  </div>
</template>
