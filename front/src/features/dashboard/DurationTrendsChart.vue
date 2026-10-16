<script setup lang="ts">
import { computed } from 'vue'
import { Bar } from 'vue-chartjs'
import { useChartTheme } from './chartSetup'

const props = defineProps<{
  labels: string[]
  durations: number[]
}>()

const { palette, version } = useChartTheme()

const chartData = computed(() => ({
  labels: props.labels,
  datasets: [
    {
      label: 'Avg Duration (s)',
      data: props.durations.map(d => +(d / 1000).toFixed(1)),
      backgroundColor: palette.value.primary,
      borderRadius: 4,
    },
  ],
}))

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    y: {
      beginAtZero: true,
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
    <Bar :key="version" :data="chartData" :options="chartOptions" />
  </div>
</template>
