<script setup lang="ts">
import { SButton, SEmptyState } from '@shared/ui'
import { FileQuestionMark } from 'lucide-vue-next'
import { useRoute } from 'vue-router'

const route = useRoute()
</script>

<template>
  <SEmptyState
    :icon="FileQuestionMark"
    title="Page not found"
    :description="`Nothing lives at ${route.path}. The link may be outdated or mistyped.`"
  >
    <SButton to="/">
      Back to dashboard
    </SButton>
  </SEmptyState>
</template>
