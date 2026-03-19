<script setup lang="ts">
defineProps<{
  label?: string
  error?: string
  hint?: string
  type?: string
  modelValue?: string | number
  placeholder?: string
  required?: boolean
  disabled?: boolean
}>()

defineEmits<{
  'update:modelValue': [value: string | number]
}>()
</script>

<template>
  <div>
    <label v-if="label" class="block text-sm font-medium text-heading mb-1.5">{{ label }}</label>
    <input
      :type="type || 'text'"
      :value="modelValue"
      :placeholder="placeholder"
      :required="required"
      :disabled="disabled"
      class="block w-full h-10 px-3.5 border rounded-lg text-sm bg-surface text-heading placeholder:text-text-muted transition-all duration-150 focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary disabled:opacity-50 disabled:bg-surface-raised"
      :class="error ? 'border-danger' : 'border-border'"
      @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    >
    <p v-if="error" class="mt-1.5 text-xs text-danger">
      {{ error }}
    </p>
    <p v-else-if="hint" class="mt-1.5 text-xs text-text-muted">
      {{ hint }}
    </p>
  </div>
</template>
