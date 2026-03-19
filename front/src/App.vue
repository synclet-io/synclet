<script setup lang="ts">
import SToast from '@shared/ui/SToast.vue'
import { defineAsyncComponent } from 'vue'
import { RouterView } from 'vue-router'

const VueQueryDevtools = import.meta.env.DEV
  ? defineAsyncComponent(() =>
      import('@tanstack/vue-query-devtools').then(m => ({ default: m.VueQueryDevtools })),
    )
  : null
</script>

<template>
  <RouterView />
  <SToast />
  <VueQueryDevtools v-if="VueQueryDevtools" />
</template>
