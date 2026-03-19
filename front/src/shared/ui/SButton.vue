<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'

const props = withDefaults(defineProps<{
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
  size?: 'sm' | 'md' | 'lg'
  loading?: boolean
  disabled?: boolean
  to?: string | object
}>(), {
  variant: 'primary',
  size: 'md',
  loading: false,
  disabled: false,
})

const classes = computed(() => {
  const base = 'inline-flex items-center justify-center gap-2 font-medium transition-all duration-150 focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-primary/50 disabled:opacity-50 disabled:pointer-events-none'

  const variants: Record<string, string> = {
    primary: 'bg-primary text-on-primary shadow-xs hover:bg-primary-hover hover:shadow-soft active:scale-[0.98]',
    secondary: 'border border-border bg-surface text-heading hover:bg-surface-hover active:scale-[0.98]',
    ghost: 'text-text-secondary hover:bg-surface-hover hover:text-heading',
    danger: 'bg-danger text-white shadow-xs hover:bg-danger-700 active:scale-[0.98]',
  }

  const sizes: Record<string, string> = {
    sm: 'px-3 py-1.5 text-xs rounded-md',
    md: 'px-4 py-2 text-sm rounded-lg',
    lg: 'px-5 py-2.5 text-sm rounded-lg',
  }

  return [base, variants[props.variant], sizes[props.size]].join(' ')
})

const component = computed(() => (props.to ? RouterLink : 'button'))
</script>

<template>
  <component
    :is="component"
    :to="to"
    :disabled="disabled || loading"
    :class="classes"
  >
    <svg v-if="loading" class="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
    </svg>
    <slot />
  </component>
</template>
