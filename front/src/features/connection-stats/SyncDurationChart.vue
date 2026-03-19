<script setup lang="ts">
import { computed } from 'vue'
import { Bar } from 'vue-chartjs'
import '@features/dashboard/chartSetup'

const props = defineProps<{
  syncs: { label: string, durationMs: number, status: string }[]
}>()

const chartData = computed(() => ({
  labels: props.syncs.map(s => s.label),
  datasets: [
    {
      label: 'Duration (s)',
      data: props.syncs.map(s => s.durationMs / 1000),
      backgroundColor: props.syncs.map(s =>
        s.status === 'completed' ? '#22c55e' : '#ef4444',
      ),
      borderRadius: 4,
    },
  ],
}))

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    x: {
      grid: { color: 'rgba(0,0,0,0.05)' },
    },
    y: {
      beginAtZero: true,
      grid: { color: 'rgba(0,0,0,0.05)' },
      title: {
        display: true,
        text: 'Duration (s)',
      },
    },
  },
  plugins: {
    legend: {
      display: false,
    },
  },
}
</script>

<template>
  <div class="h-64">
    <Bar :data="chartData" :options="chartOptions" />
  </div>
</template>
