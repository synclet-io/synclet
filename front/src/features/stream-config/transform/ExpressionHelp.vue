<script setup lang="ts">
import { docsUrl } from '@shared/lib/docs'
import { HelpCircle } from 'lucide-vue-next'
</script>

<template>
  <a
    data-testid="expression-help"
    :href="docsUrl('guides/transforms')"
    target="_blank"
    rel="noopener noreferrer"
    title="Expressions and functions reference"
    aria-label="Expressions and functions reference"
    class="inline-flex text-text-muted hover:text-primary"
  >
    <HelpCircle class="w-3.5 h-3.5" />
  </a>
</template>
