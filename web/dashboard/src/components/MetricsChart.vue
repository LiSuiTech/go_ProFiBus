<template>
  <div class="metrics-chart">
    <div ref="chartRef" :style="{ width: '100%', height: height }"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import * as echarts from 'echarts'
import type { ECharts } from 'echarts'
import type { ComponentMetrics } from '@/types'

interface Props {
  metrics: ComponentMetrics[]
  height?: string
  type?: 'bar' | 'line'
}

const props = withDefaults(defineProps<Props>(), {
  height: '400px',
  type: 'bar',
})

const chartRef = ref<HTMLElement>()
let chartInstance: ECharts | null = null

function initChart() {
  if (!chartRef.value) return

  chartInstance = echarts.init(chartRef.value)

  const option: echarts.EChartsOption = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow',
      },
    },
    legend: {
      data: ['Avg Duration', 'Max Duration', 'Error Count'],
      bottom: 0,
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '15%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: props.metrics.map((m) => m.component_name),
      axisLabel: {
        rotate: 45,
        interval: 0,
      },
    },
    yAxis: [
      {
        type: 'value',
        name: 'Duration (ms)',
        position: 'left',
      },
      {
        type: 'value',
        name: 'Error Count',
        position: 'right',
      },
    ],
    series: [
      {
        name: 'Avg Duration',
        type: props.type,
        data: props.metrics.map((m) => m.avg_duration_ms),
        yAxisIndex: 0,
        itemStyle: {
          color: '#5470c6',
        },
      },
      {
        name: 'Max Duration',
        type: props.type,
        data: props.metrics.map((m) => m.max_duration_ms),
        yAxisIndex: 0,
        itemStyle: {
          color: '#91cc75',
        },
      },
      {
        name: 'Error Count',
        type: 'bar',
        data: props.metrics.map((m) => m.error_count),
        yAxisIndex: 1,
        itemStyle: {
          color: '#ee6666',
        },
      },
    ],
  }

  chartInstance.setOption(option)
}

function updateChart() {
  if (!chartInstance) return

  const option: echarts.EChartsOption = {
    xAxis: {
      data: props.metrics.map((m) => m.component_name),
    },
    series: [
      {
        data: props.metrics.map((m) => m.avg_duration_ms),
      },
      {
        data: props.metrics.map((m) => m.max_duration_ms),
      },
      {
        data: props.metrics.map((m) => m.error_count),
      },
    ],
  }

  chartInstance.setOption(option)
}

function handleResize() {
  chartInstance?.resize()
}

onMounted(() => {
  initChart()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  chartInstance?.dispose()
})

watch(
  () => props.metrics,
  () => {
    updateChart()
  },
  { deep: true }
)
</script>

<style scoped>
.metrics-chart {
  width: 100%;
}
</style>
