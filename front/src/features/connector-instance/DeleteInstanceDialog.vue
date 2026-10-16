<script setup lang="ts">
import { SConfirmDialog } from '@shared/ui'
import { computed } from 'vue'

const props = defineProps<{
  open: boolean
  kind: 'source' | 'destination'
  name: string
  loading?: boolean
  error?: string
}>()

defineEmits<{
  confirm: []
  cancel: []
}>()

const title = computed(() => `Delete ${props.kind}`)
const message = computed(() =>
  `Delete "${props.name}"? This cannot be undone. Connections that use this ${props.kind} must be deleted first.`,
)
</script>

<template>
  <SConfirmDialog
    :open="open"
    :title="title"
    :message="message"
    confirm-text="Delete"
    :loading="loading"
    :error="error"
    @confirm="$emit('confirm')"
    @cancel="$emit('cancel')"
  />
</template>
