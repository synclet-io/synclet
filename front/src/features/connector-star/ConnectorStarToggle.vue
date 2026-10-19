<script setup lang="ts">
import type { ManagedConnector } from '@entities/connector'
import { useAuth } from '@entities/auth'
import { useSetConnectorStar } from '@entities/connector'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { useToast } from '@shared/ui'
import { Star } from 'lucide-vue-next'
import { computed } from 'vue'

const props = defineProps<{
  connector: Pick<ManagedConnector, 'id' | 'name' | 'starred'>
}>()

const auth = useAuth()
const toast = useToast()
const setStar = useSetConnectorStar()

const canEdit = computed(() => {
  const role = auth.workspaces.value.find(w => w.workspaceId === auth.currentWorkspaceId.value)?.role
  return role === 'editor' || role === 'admin'
})

const label = computed(() => `${props.connector.starred ? 'Unstar' : 'Star'} ${props.connector.name}`)

function toggle() {
  setStar.mutate(
    { id: props.connector.id, starred: !props.connector.starred },
    { onError: e => toast.error(getErrorMessage(e) || 'Failed to update the star') },
  )
}
</script>

<template>
  <button
    type="button"
    class="inline-flex items-center justify-center w-8 h-8 rounded-lg transition-colors duration-150 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 disabled:opacity-50 disabled:cursor-not-allowed"
    :class="connector.starred ? 'text-warning hover:bg-surface-hover' : 'text-text-muted hover:text-heading hover:bg-surface-hover'"
    :aria-label="label"
    :aria-pressed="connector.starred"
    :title="canEdit ? label : 'Only editors can star connectors'"
    :disabled="!canEdit"
    @click.stop="toggle"
  >
    <Star class="w-4 h-4" :fill="connector.starred ? 'currentColor' : 'none'" />
  </button>
</template>
