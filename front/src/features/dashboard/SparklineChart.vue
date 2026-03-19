<script setup lang="ts">
import { computed } from 'vue'
import { Line } from 'vue-chartjs'
import './chartSetup'

const props = withDefaults(defineProps<{
  values: number[]
  color?: string
}>(), {
  color: '#6366f1',
})

const chartData = computed(() => ({
  labels: props.values.map(() => ''),
  datasets: [
    {
      data: props.values,
      borderColor: props.color,
      borderWidth: 1.5,
      pointRadius: 0,
      tension: 0.3,
      fill: false,
    },
  ],
}))

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    x: { display: false },
    y: { display: false },
  },
  plugins: {
    legend: { display: false },
    tooltip: { enabled: false },
  },
  layout: {
    padding: 0,
  },
}
</script>

<template>
  <div class="h-8 w-24">
    <Line :data="chartData" :options="chartOptions" />
  </div>
</template>
