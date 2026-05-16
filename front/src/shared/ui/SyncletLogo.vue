<script setup lang="ts">
withDefaults(defineProps<{
  /** Tailwind size class for the square mark. Defaults to "w-7 h-7". */
  size?: string
}>(), {
  size: 'w-7 h-7',
})
</script>

<template>
  <img
    src="/favicon.svg"
    alt="Synclet"
    class="rounded-lg"
    :class="size"
  >
</template>
