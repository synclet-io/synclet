<script setup lang="ts">
defineProps<{
  label?: string
  error?: string
  options: Array<{ label: string, value: string | number }>
  modelValue?: string | number
  placeholder?: string
  required?: boolean
  disabled?: boolean
  size?: 'sm' | 'md'
}>()

defineEmits<{
  'update:modelValue': [value: string | number]
}>()
</script>

<template>
  <div>
    <label v-if="label" class="block text-sm font-medium text-heading mb-1.5">{{ label }}</label>
    <select
      :value="modelValue"
      :required="required"
      :disabled="disabled"
      class="block w-full border rounded-lg bg-surface text-heading transition-all duration-150 focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary disabled:opacity-50"
      :class="[
        error ? 'border-danger' : 'border-border',
        size === 'sm' ? 'h-7 px-2 text-xs' : 'h-10 px-3.5 text-sm',
      ]"
      @change="$emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
    >
      <option v-if="placeholder" value="" disabled>
        {{ placeholder }}
      </option>
      <option v-for="opt in options" :key="opt.value" :value="opt.value">
        {{ opt.label }}
      </option>
    </select>
    <p v-if="error" class="mt-1.5 text-xs text-danger">
      {{ error }}
    </p>
  </div>
</template>
