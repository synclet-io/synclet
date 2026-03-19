<script setup lang="ts">
import { resetStreamState, useUpdateStreamState } from '@entities/connection'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SButton, SConfirmDialog, useToast } from '@shared/ui'
import { computed, ref } from 'vue'

const props = defineProps<{
  connectionId: string
  streamName: string
  streamNamespace: string
  stateData: string | undefined
  updatedAt: Date | undefined
}>()

const toast = useToast()
const updateStreamStateMutation = useUpdateStreamState()

const editedState = ref(props.stateData ?? '')
const confirmReset = ref(false)
const saving = ref(false)
const resetting = ref(false)

const isEdited = computed(() => editedState.value !== (props.stateData ?? ''))

const resetMessage = computed(() => {
  const displayName = props.streamNamespace
    ? `${props.streamNamespace}.${props.streamName}`
    : props.streamName
  return `Reset state for "${displayName}"? The next sync will perform a full refresh.`
})

async function handleSave() {
  try {
    JSON.parse(editedState.value)
  }
  catch {
    toast.error('Invalid JSON in state data')
    return
  }

  saving.value = true
  try {
    await updateStreamStateMutation.mutateAsync({
      connectionId: props.connectionId,
      streamName: props.streamName,
      streamNamespace: props.streamNamespace,
      stateData: editedState.value,
    })
    const displayName = props.streamNamespace
      ? `${props.streamNamespace}.${props.streamName}`
      : props.streamName
    toast.success(`State updated for "${displayName}"`)
  }
  catch (e: unknown) {
    toast.error(getErrorMessage(e) || 'Failed to update state')
  }
  finally {
    saving.value = false
  }
}

async function handleReset() {
  confirmReset.value = false
  resetting.value = true
  try {
    await resetStreamState(props.connectionId, props.streamNamespace, props.streamName)
    editedState.value = ''
    const displayName = props.streamNamespace
      ? `${props.streamNamespace}.${props.streamName}`
      : props.streamName
    toast.success(`State reset for "${displayName}"`)
  }
  catch (e: unknown) {
    toast.error(getErrorMessage(e) || 'Failed to reset state')
  }
  finally {
    resetting.value = false
  }
}
</script>

<template>
  <div class="p-4 space-y-3">
    <textarea
      v-model="editedState"
      rows="6"
      class="w-full font-mono text-xs bg-surface-raised border border-border rounded-lg px-3 py-2 resize-y focus:outline-none focus:ring-2 focus:ring-primary/50"
      placeholder="No state data"
    />
    <div class="flex items-center gap-2">
      <SButton size="sm" :disabled="!isEdited" :loading="saving" @click="handleSave">
        Save State
      </SButton>
      <SButton size="sm" variant="ghost" class="text-danger" :loading="resetting" @click="confirmReset = true">
        Reset State
      </SButton>
      <span v-if="updatedAt" class="text-xs text-text-secondary ml-auto">
        Last updated: {{ updatedAt.toLocaleString() }}
      </span>
    </div>

    <SConfirmDialog
      :open="confirmReset"
      title="Reset stream state"
      :message="resetMessage"
      confirm-text="Reset"
      @confirm="handleReset"
      @cancel="confirmReset = false"
    />
  </div>
</template>
