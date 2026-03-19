<script setup lang="ts">
import { computed } from 'vue'
import { Doughnut } from 'vue-chartjs'
import './chartSetup'

const props = defineProps<{
  labels: string[]
  counts: number[]
}>()

const colorMap: Record<string, string> = {
  connector: '#ef4444',
  timeout: '#f59e0b',
  oom: '#f97316',
  infrastructure: '#64748b',
  unknown: '#cbd5e1',
}

const chartData = computed(() => ({
  labels: props.labels,
  datasets: [
    {
      data: props.counts,
      backgroundColor: props.labels.map(l => colorMap[l.toLowerCase()] || '#cbd5e1'),
      borderWidth: 0,
    },
  ],
}))

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      position: 'right' as const,
    },
  },
}
</script>

<template>
  <div class="h-64">
    <Doughnut :data="chartData" :options="chartOptions" />
  </div>
</template>
