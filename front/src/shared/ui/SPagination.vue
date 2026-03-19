<script setup lang="ts">
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-vue-next'
import { computed } from 'vue'

const props = defineProps<{
  total: number
  pageSize: number
  currentPage: number
}>()

const emit = defineEmits<{
  pageChange: [page: number]
}>()

const totalPages = computed(() => Math.ceil(props.total / props.pageSize))

const visiblePages = computed(() => {
  const pages: (number | '...')[] = []
  const tp = totalPages.value
  const cp = props.currentPage

  if (tp <= 7) {
    for (let i = 1; i <= tp; i++) pages.push(i)
    return pages
  }

  pages.push(1)
  if (cp > 3)
    pages.push('...')

  const start = Math.max(2, cp - 1)
  const end = Math.min(tp - 1, cp + 1)
  for (let i = start; i <= end; i++) pages.push(i)

  if (cp < tp - 2)
    pages.push('...')
  pages.push(tp)

  return pages
})

const rangeStart = computed(() => Math.min((props.currentPage - 1) * props.pageSize + 1, props.total))
const rangeEnd = computed(() => Math.min(props.currentPage * props.pageSize, props.total))

const isFirstPage = computed(() => props.currentPage <= 1)
const isLastPage = computed(() => props.currentPage >= totalPages.value)
</script>

<template>
  <div v-if="total > pageSize" class="flex flex-col items-center gap-2">
    <div class="flex items-center gap-1">
      <button
        :disabled="isFirstPage"
        class="w-8 h-8 flex items-center justify-center rounded-lg text-text-secondary hover:bg-surface-hover disabled:opacity-40 disabled:pointer-events-none"
        @click="emit('pageChange', 1)"
      >
        <ChevronsLeft class="w-4 h-4" />
      </button>
      <button
        :disabled="isFirstPage"
        class="w-8 h-8 flex items-center justify-center rounded-lg text-text-secondary hover:bg-surface-hover disabled:opacity-40 disabled:pointer-events-none"
        @click="emit('pageChange', currentPage - 1)"
      >
        <ChevronLeft class="w-4 h-4" />
      </button>

      <template v-for="page in visiblePages" :key="page">
        <span v-if="page === '...'" class="w-8 h-8 flex items-center justify-center text-text-muted">...</span>
        <button
          v-else
          class="min-w-[32px] h-8 px-2 flex items-center justify-center rounded-lg text-sm font-medium" :class="[
            page === currentPage
              ? 'bg-primary text-white'
              : 'text-text-secondary hover:bg-surface-hover',
          ]"
          @click="emit('pageChange', page as number)"
        >
          {{ page }}
        </button>
      </template>

      <button
        :disabled="isLastPage"
        class="w-8 h-8 flex items-center justify-center rounded-lg text-text-secondary hover:bg-surface-hover disabled:opacity-40 disabled:pointer-events-none"
        @click="emit('pageChange', currentPage + 1)"
      >
        <ChevronRight class="w-4 h-4" />
      </button>
      <button
        :disabled="isLastPage"
        class="w-8 h-8 flex items-center justify-center rounded-lg text-text-secondary hover:bg-surface-hover disabled:opacity-40 disabled:pointer-events-none"
        @click="emit('pageChange', totalPages)"
      >
        <ChevronsRight class="w-4 h-4" />
      </button>
    </div>
    <span class="text-xs text-text-muted">{{ rangeStart }}-{{ rangeEnd }} of {{ total }}</span>
  </div>
</template>
