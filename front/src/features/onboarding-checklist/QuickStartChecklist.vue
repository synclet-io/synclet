<script setup lang="ts">
import { SButton, SCard } from '@shared/ui'
import { ArrowRight, CheckCircle2, Circle, PartyPopper, X } from 'lucide-vue-next'
import { useOnboardingState } from './useOnboardingState'

const state = useOnboardingState()
</script>

<template>
  <SCard v-if="state.shouldShow.value" class="mb-6">
    <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3 sm:gap-4 mb-4">
      <div class="flex-1">
        <h2 class="text-base font-semibold text-heading flex items-center gap-2">
          <PartyPopper v-if="state.allComplete.value" class="w-5 h-5 text-success" />
          Quick start
        </h2>
        <p v-if="state.allComplete.value" class="text-sm text-text-secondary mt-1">
          You're all set. Hide this widget when you're ready.
        </p>
        <p v-else class="text-sm text-text-secondary mt-1">
          {{ state.completedCount.value }} of {{ state.totalSteps.value }} complete — finish the steps to land your first sync.
        </p>
      </div>
      <SButton
        v-if="state.allComplete.value"
        variant="ghost"
        size="sm"
        @click="state.dismiss()"
      >
        <X class="w-4 h-4" /> Dismiss
      </SButton>
    </div>

    <!-- Progress bar -->
    <div class="h-1.5 w-full bg-surface-raised rounded-full overflow-hidden mb-4">
      <div
        class="h-full bg-primary transition-all duration-300"
        :style="{ width: `${(state.completedCount.value / state.totalSteps.value) * 100}%` }"
      />
    </div>

    <ul class="space-y-2">
      <li
        v-for="(step, index) in state.steps.value"
        :key="step.id"
        class="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-3 p-2 rounded-lg transition-colors"
        :class="step.done ? 'opacity-60' : 'hover:bg-surface-hover'"
      >
        <div class="flex items-start gap-3 flex-1 min-w-0">
          <CheckCircle2 v-if="step.done" class="w-5 h-5 text-success shrink-0 mt-0.5" />
          <Circle v-else class="w-5 h-5 text-text-muted shrink-0 mt-0.5" />

          <div class="flex-1 min-w-0">
            <div class="flex items-baseline gap-2 flex-wrap">
              <span class="text-xs text-text-muted">{{ index + 1 }}.</span>
              <span
                class="text-sm font-medium"
                :class="step.done ? 'line-through text-text-secondary' : 'text-text-primary'"
              >
                {{ step.title }}
              </span>
            </div>
            <p v-if="!step.done" class="text-xs text-text-secondary mt-0.5">
              {{ step.description }}
            </p>
          </div>
        </div>

        <SButton
          v-if="!step.done"
          :to="step.cta.to"
          variant="ghost"
          size="sm"
          class="self-start sm:self-auto ml-8 sm:ml-0 shrink-0"
        >
          {{ step.cta.label }}
          <ArrowRight class="w-3.5 h-3.5" />
        </SButton>
      </li>
    </ul>
  </SCard>
</template>
