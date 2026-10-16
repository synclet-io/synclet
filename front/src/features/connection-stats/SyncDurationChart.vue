<script setup lang="ts">
import { useChartTheme } from '@features/dashboard/chartSetup'
import { computed } from 'vue'
import { Bar } from 'vue-chartjs'

const props = defineProps<{
  syncs: { label: string, durationMs: number, status: string }[]
}>()

const { palette, version } = useChartTheme()

const chartData = computed(() => ({
  labels: props.syncs.map(s => s.label),
  datasets: [
    {
      label: 'Duration (s)',
      data: props.syncs.map(s => s.durationMs / 1000),
      backgroundColor: props.syncs.map(s =>
        s.status === 'completed' ? palette.value.success : palette.value.danger,
      ),
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
    <Bar :key="version" :data="chartData" :options="chartOptions" />
  </div>
</template>
