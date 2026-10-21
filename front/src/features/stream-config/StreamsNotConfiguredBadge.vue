<script setup lang="ts">
import { SBadge } from '@shared/ui'
import { CONFIGURE_STREAMS_FIRST } from './streamsConfigured'

defineProps<{
  connectionId: string
}>()
</script>

<template>
  <RouterLink
    :to="`/connections/${connectionId}/streams`"
    class="inline-flex"
    :title="`${CONFIGURE_STREAMS_FIRST}: this connection has no configured stream, so it cannot sync.`"
  >
    <SBadge variant="warning" dot>
      Streams not configured
    </SBadge>
  </RouterLink>
</template>
