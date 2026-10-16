<script setup lang="ts">
import { Check, Minus } from 'lucide-vue-next'
import { computed, useAttrs } from 'vue'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  modelValue?: boolean
  label?: string
  indeterminate?: boolean
  disabled?: boolean
  size?: 'sm' | 'md'
}>(), {
  size: 'md',
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const attrs = useAttrs()

const controlAttrs = computed(() => {
  const { class: _class, style: _style, ...rest } = attrs
  return rest
})

function onChange(event: Event) {
  const el = event.target as HTMLInputElement
  el.indeterminate = props.indeterminate
  emit('update:modelValue', el.checked)
}
</script>

<template>
  <label
    class="inline-flex items-center gap-2 align-middle"
    :class="[$attrs.class, disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer']"
    :style="$attrs.style"
  >
    <span class="relative inline-flex shrink-0" :class="size === 'sm' ? 'w-3.5 h-3.5' : 'w-4 h-4'">
      <input
        v-bind="controlAttrs"
        type="checkbox"
        :checked="modelValue"
        :indeterminate.prop="indeterminate"
        :disabled="disabled"
        class="absolute inset-0 m-0 w-full h-full appearance-none rounded border border-border bg-surface transition-colors duration-150 cursor-pointer enabled:hover:border-primary checked:bg-primary-solid checked:border-primary-solid indeterminate:bg-primary-solid indeterminate:border-primary-solid focus:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-surface disabled:cursor-not-allowed"
        @change="onChange"
      >
      <Minus v-if="indeterminate" class="pointer-events-none absolute inset-0 w-full h-full p-px text-on-primary" :stroke-width="3" aria-hidden="true" />
      <Check v-else-if="modelValue" class="pointer-events-none absolute inset-0 w-full h-full p-px text-on-primary" :stroke-width="3" aria-hidden="true" />
    </span>
    <span v-if="label || $slots.default" class="text-sm text-heading">
      <slot>{{ label }}</slot>
    </span>
  </label>
</template>
