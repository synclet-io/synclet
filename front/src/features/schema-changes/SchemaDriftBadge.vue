<script setup lang="ts">
import { SBadge } from '@shared/ui'

defineProps<{
  connectionId: string
  count: number
}>()
</script>

<template>
  <RouterLink
    v-if="count > 0"
    :to="{ name: 'connection-schema-changes', params: { id: connectionId } }"
    class="inline-flex"
    :title="`${count} pending schema ${count === 1 ? 'change' : 'changes'}`"
  >
    <SBadge variant="warning" dot>
      Schema drift
    </SBadge>
  </RouterLink>
</template>
