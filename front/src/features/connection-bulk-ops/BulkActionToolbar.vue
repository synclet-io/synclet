<script setup lang="ts">
import { SButton } from '@shared/ui'
import { CheckSquare, Pause, Play, Trash2, X } from 'lucide-vue-next'

defineProps<{
  count: number
  running?: boolean
}>()

const emit = defineEmits<{
  enable: []
  disable: []
  delete: []
  clear: []
}>()
</script>

<template>
  <div class="flex items-center justify-between gap-3 px-4 py-2 mb-3 bg-primary/10 border border-primary/30 rounded-lg">
    <div class="flex items-center gap-2 text-sm font-medium text-heading">
      <CheckSquare class="w-4 h-4 text-primary" />
      {{ count }} selected
    </div>
    <div class="flex items-center gap-2">
      <SButton variant="secondary" size="sm" :disabled="running" @click="emit('enable')">
        <Play class="w-4 h-4" /> Enable
      </SButton>
      <SButton variant="secondary" size="sm" :disabled="running" @click="emit('disable')">
        <Pause class="w-4 h-4" /> Disable
      </SButton>
      <SButton variant="danger" size="sm" :disabled="running" @click="emit('delete')">
        <Trash2 class="w-4 h-4" /> Delete
      </SButton>
      <SButton variant="ghost" size="icon" title="Clear selection" aria-label="Clear selection" :disabled="running" @click="emit('clear')">
        <X class="w-4 h-4" />
      </SButton>
    </div>
  </div>
</template>
