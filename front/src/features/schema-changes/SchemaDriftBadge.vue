<script setup lang="ts">
import { useSchemaChanges } from '@entities/connection'
import { SBadge } from '@shared/ui'
import { computed, ref } from 'vue'

const props = defineProps<{
  connectionId: string
}>()

const enabled = ref(true)
const { data: changes } = useSchemaChanges(computed(() => props.connectionId), { enabled })

const count = computed(() => changes.value?.length ?? 0)
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
