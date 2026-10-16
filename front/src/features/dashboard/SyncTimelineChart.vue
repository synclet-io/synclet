<script setup lang="ts">
import { computed } from 'vue'
import { Line } from 'vue-chartjs'
import { useChartTheme, withAlpha } from './chartSetup'

const props = defineProps<{
  labels: string[]
  succeeded: number[]
  failed: number[]
}>()

const { palette, version } = useChartTheme()

const chartData = computed(() => ({
  labels: props.labels,
  datasets: [
    {
      label: 'Succeeded',
      data: props.succeeded,
      borderColor: palette.value.success,
      backgroundColor: withAlpha(palette.value.success, 0.1),
      fill: true,
      tension: 0.3,
      pointRadius: 2,
    },
    {
      label: 'Failed',
      data: props.failed,
      borderColor: palette.value.danger,
      backgroundColor: withAlpha(palette.value.danger, 0.1),
      fill: true,
      tension: 0.3,
      pointRadius: 2,
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
      position: 'bottom' as const,
    },
  },
}
</script>

<template>
  <div class="h-64">
    <Line :key="version" :data="chartData" :options="chartOptions" />
  </div>
</template>
