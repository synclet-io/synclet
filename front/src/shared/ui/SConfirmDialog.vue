<script setup lang="ts">
import SAlert from './SAlert.vue'
import SButton from './SButton.vue'
import SModal from './SModal.vue'

defineOptions({ inheritAttrs: false })

withDefaults(defineProps<{
  open: boolean
  title?: string
  message?: string
  confirmText?: string
  cancelText?: string
  variant?: 'primary' | 'danger'
  loading?: boolean
  error?: string
}>(), {
  title: 'Confirm',
  confirmText: 'Confirm',
  cancelText: 'Cancel',
  variant: 'danger',
  loading: false,
})

defineEmits<{
  confirm: []
  cancel: []
}>()
</script>

<template>
  <SModal :open="open" :title="title" size="sm" @close="$emit('cancel')">
    <p class="text-sm text-text-secondary">
      {{ message }}
    </p>
    <SAlert v-if="error" variant="danger" class="mt-4">
      {{ error }}
    </SAlert>
    <template #footer>
      <SButton variant="secondary" @click="$emit('cancel')">
        {{ cancelText }}
      </SButton>
      <SButton :variant="variant" :loading="loading" @click="$emit('confirm')">
        {{ confirmText }}
      </SButton>
    </template>
  </SModal>
</template>
