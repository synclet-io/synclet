<script setup lang="ts">
import type { Column } from './types'
import SEmptyState from './SEmptyState.vue'
import SSkeleton from './SSkeleton.vue'

defineProps<{
  columns: Column[]
  data: any[] | undefined | null
  loading?: boolean
  emptyText?: string
  emptyDescription?: string
}>()

defineSlots<{
  [key: `cell-${string}`]: (props: { row: any, index: number }) => any
  empty?: () => any
}>()
</script>

<template>
  <div class="bg-surface border border-border rounded-xl overflow-hidden">
    <!-- Loading skeleton -->
    <div v-if="loading" class="divide-y divide-border">
      <div class="bg-surface-raised px-6 py-3.5 flex gap-8">
        <SSkeleton v-for="col in columns" :key="col.key" variant="text" width="80px" />
      </div>
      <div v-for="i in 5" :key="i" class="px-6 py-3.5 flex gap-8">
        <SSkeleton v-for="col in columns" :key="col.key" variant="text" width="120px" />
      </div>
    </div>

    <!-- Empty state -->
    <div v-else-if="!data || data.length === 0" class="py-12">
      <slot name="empty">
        <SEmptyState :title="emptyText || 'No data'" :description="emptyDescription" />
      </slot>
    </div>

    <!-- Table -->
    <div v-else class="overflow-x-auto">
      <table class="w-full min-w-[600px]">
        <thead>
          <tr class="border-b border-border">
            <th
              v-for="col in columns"
              :key="col.key"
              class="px-6 py-3 text-xs font-medium text-text-muted uppercase tracking-wider bg-surface-raised"
              :class="{
                'text-left': col.align !== 'right' && col.align !== 'center',
                'text-right': col.align === 'right',
                'text-center': col.align === 'center',
              }"
              :style="col.width ? { width: col.width } : {}"
            >
              {{ col.label }}
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-for="(row, index) in data" :key="index" class="hover:bg-surface-hover/50 transition-colors">
            <td
              v-for="col in columns"
              :key="col.key"
              class="px-6 py-3.5 text-sm"
              :class="{
                'text-left': col.align !== 'right' && col.align !== 'center',
                'text-right': col.align === 'right',
                'text-center': col.align === 'center',
              }"
            >
              <slot :name="`cell-${col.key}`" :row="row" :index="index">
                {{ (row as any)[col.key] }}
              </slot>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
