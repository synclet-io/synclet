<script setup lang="ts">
import type { ChartPalette } from './chartSetup'
import { computed } from 'vue'
import { Doughnut } from 'vue-chartjs'
import { useChartTheme } from './chartSetup'

const props = defineProps<{
  labels: string[]
  counts: number[]
}>()

const { palette, version } = useChartTheme()

const colorKeys: Record<string, keyof ChartPalette> = {
  connector: 'danger',
  timeout: 'warning',
  oom: 'accent',
  infrastructure: 'neutral',
}

const chartData = computed(() => ({
  labels: props.labels,
  datasets: [
    {
      data: props.counts,
      backgroundColor: props.labels.map(l => palette.value[colorKeys[l.toLowerCase()] ?? 'muted']),
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
    <Doughnut :key="version" :data="chartData" :options="chartOptions" />
  </div>
</template>
