<script setup lang="ts">
import { AnsiUp } from 'ansi_up'
import DOMPurify from 'dompurify'
import { nextTick, ref, watch } from 'vue'

const props = defineProps<{
  lines: string[]
  loading?: boolean
}>()

const ansi = new AnsiUp()

const container = ref<HTMLElement>()
const isAtBottom = ref(true)

function renderLine(line: string): string {
  return DOMPurify.sanitize(ansi.ansi_to_html(line))
}

function onScroll() {
  if (!container.value)
    return
  const { scrollTop, scrollHeight, clientHeight } = container.value
  isAtBottom.value = scrollTop + clientHeight >= scrollHeight - 20
}

function scrollToBottom() {
  if (container.value) {
    container.value.scrollTop = container.value.scrollHeight
  }
  isAtBottom.value = true
}

// Auto-scroll when new lines arrive and user is at bottom
watch(() => props.lines.length, () => {
  if (isAtBottom.value) {
    nextTick(scrollToBottom)
  }
})
</script>

<template>
  <div class="relative">
    <div
      ref="container"
      class="bg-gray-950 text-gray-200 font-mono text-sm p-4 rounded-lg overflow-auto max-h-[600px] min-h-[200px]"
      @scroll="onScroll"
    >
      <div v-if="loading && lines.length === 0" class="text-gray-500">
        Loading logs...
      </div>
      <div v-else-if="lines.length === 0" class="text-gray-500">
        No log output yet.
      </div>
      <div v-else>
        <div v-for="(line, idx) in lines" :key="idx" class="whitespace-pre-wrap break-all leading-5" v-html="renderLine(line)" />
      </div>
    </div>
    <!-- Jump to bottom button -->
    <Transition name="fade">
      <button
        v-if="!isAtBottom && lines.length > 0"
        class="absolute bottom-4 right-4 bg-gray-700 hover:bg-gray-600 text-white text-xs px-3 py-1.5 rounded-full shadow-lg"
        @click="scrollToBottom"
      >
        Jump to bottom
      </button>
    </Transition>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
