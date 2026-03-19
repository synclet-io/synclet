<script setup lang="ts">
import type { BreakingChange } from '@entities/connector'
import SButton from '@shared/ui/SButton.vue'
import SModal from '@shared/ui/SModal.vue'
import DOMPurify from 'dompurify'
import { marked } from 'marked'

withDefaults(defineProps<{
  open: boolean
  connectorName: string
  currentVersion: string
  targetVersion: string
  breakingChanges: BreakingChange[]
  loading?: boolean
}>(), {
  loading: false,
})

defineEmits<{
  confirm: []
  cancel: []
}>()

function renderMarkdown(text: string): string {
  const html = marked.parse(text, { async: false }) as string
  return DOMPurify.sanitize(html)
}
</script>

<template>
  <SModal :open="open" title="Breaking Changes Detected" size="md" @close="$emit('cancel')">
    <div class="space-y-4">
      <p class="text-sm text-text-secondary">
        Updating <strong>{{ connectorName }}</strong> from
        <code class="px-1 py-0.5 bg-gray-100 dark:bg-gray-800 rounded text-xs">{{ currentVersion }}</code>
        to
        <code class="px-1 py-0.5 bg-gray-100 dark:bg-gray-800 rounded text-xs">{{ targetVersion }}</code>
        includes breaking changes:
      </p>
      <ul class="space-y-3">
        <li v-for="bc in breakingChanges" :key="bc.version" class="border-l-2 border-warning pl-3">
          <div class="text-sm font-semibold">
            v{{ bc.version }}
          </div>
          <div class="text-sm text-text-secondary [&_a]:text-primary [&_a]:underline [&_strong]:font-semibold" v-html="renderMarkdown(bc.message)" />
          <a
            v-if="bc.migrationDocumentationUrl"
            :href="bc.migrationDocumentationUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-xs text-primary hover:underline"
          >
            Migration guide
          </a>
        </li>
      </ul>
    </div>
    <template #footer>
      <SButton variant="secondary" @click="$emit('cancel')">
        Keep Current Version
      </SButton>
      <SButton variant="primary" :loading="loading" @click="$emit('confirm')">
        Update Anyway
      </SButton>
    </template>
  </SModal>
</template>
