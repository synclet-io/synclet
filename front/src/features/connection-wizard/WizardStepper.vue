<script setup lang="ts">
defineProps<{
  currentStep: number
  steps: { label: string }[]
}>()

defineEmits<{
  goTo: [step: number]
}>()
</script>

<template>
  <nav class="flex items-center justify-center mb-12">
    <template v-for="(step, i) in steps" :key="i">
      <button
        class="flex items-center gap-2"
        :class="i + 1 < currentStep ? 'cursor-pointer' : 'cursor-default'"
        :disabled="i + 1 > currentStep"
        @click="i + 1 < currentStep ? $emit('goTo', i + 1) : undefined"
      >
        <span
          class="w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium"
          :class="i + 1 === currentStep ? 'bg-primary text-white' : i + 1 < currentStep ? 'bg-primary/20 text-primary' : 'bg-surface-raised text-text-muted'"
        >{{ i + 1 }}</span>
        <span
          class="text-sm hidden md:inline"
          :class="i + 1 === currentStep ? 'font-semibold text-heading' : 'text-text-secondary'"
        >{{ step.label }}</span>
      </button>
      <div v-if="i < steps.length - 1" class="mx-3 h-px w-8 bg-border" />
    </template>
  </nav>
</template>
