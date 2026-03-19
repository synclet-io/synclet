<script setup lang="ts">
import { computed } from 'vue'
import { Bar } from 'vue-chartjs'
import './chartSetup'

const props = defineProps<{
  labels: string[]
  durations: number[]
}>()

const chartData = computed(() => ({
  labels: props.labels,
  datasets: [
    {
      label: 'Avg Duration (s)',
      data: props.durations.map(d => +(d / 1000).toFixed(1)),
      backgroundColor: '#3b82f6',
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
