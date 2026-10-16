<script setup lang="ts">
import { useChartTheme, withAlpha } from '@features/dashboard/chartSetup'
import { computed } from 'vue'
import { Line } from 'vue-chartjs'

const props = defineProps<{
  labels: string[]
  read: number[]
}>()

const { palette, version } = useChartTheme()

const chartData = computed(() => ({
  labels: props.labels,
  datasets: [
    {
      label: 'Records Read',
      data: props.read,
      borderColor: palette.value.primary,
      backgroundColor: withAlpha(palette.value.primary, 0.2),
      fill: 'origin',
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
