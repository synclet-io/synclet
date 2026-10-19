<script setup lang="ts">
import type { ManagedConnector } from '@entities/connector'
import { useAuth } from '@entities/auth'
import { useConnectorStarPending, useSetConnectorStar } from '@entities/connector'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { useToast } from '@shared/ui'
import { Star } from 'lucide-vue-next'
import { computed, useId } from 'vue'

const props = defineProps<{
  connector: Pick<ManagedConnector, 'id' | 'name' | 'starred'>
}>()

const VIEWER_HINT = 'Only editors can star connectors'

const auth = useAuth()
const toast = useToast()
const setStar = useSetConnectorStar()
const pending = useConnectorStarPending(computed(() => props.connector.id))
const hintId = useId()

const canEdit = computed(() => {
  const role = auth.workspaces.value.find(w => w.workspaceId === auth.currentWorkspaceId.value)?.role
  return role === 'editor' || role === 'admin'
})

const label = computed(() => `Star ${props.connector.name}`)

function toggle() {
  if (!canEdit.value || pending.value)
    return
  setStar.mutate(
    { id: props.connector.id, starred: !props.connector.starred },
    { onError: e => toast.error(getErrorMessage(e) || 'Failed to update the star') },
  )
}
</script>

<template>
  <button
    type="button"
    class="inline-flex items-center justify-center w-8 h-8 rounded-lg transition-colors duration-150 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 aria-disabled:opacity-50 aria-disabled:cursor-not-allowed"
    :class="connector.starred ? 'text-warning hover:bg-surface-hover' : 'text-text-muted hover:text-heading hover:bg-surface-hover'"
    :aria-label="label"
    :aria-pressed="connector.starred"
    :aria-disabled="canEdit ? undefined : 'true'"
    :aria-describedby="canEdit ? undefined : hintId"
    :title="canEdit ? label : VIEWER_HINT"
    @click.stop="toggle"
  >
    <Star class="w-4 h-4" :fill="connector.starred ? 'currentColor' : 'none'" />
    <span v-if="!canEdit" :id="hintId" class="sr-only">{{ VIEWER_HINT }}</span>
  </button>
</template>
