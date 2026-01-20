<template>
  <div class="throughput-chart">
    <div ref="chartRef" :style="{ width: '100%', height: height }"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import * as echarts from 'echarts'
import type { ECharts } from 'echarts'

interface DataPoint {
  timestamp: string
  value: number
}

interface Props {
  data: DataPoint[]
  title?: string
  height?: string
  unit?: string
}

const props = withDefaults(defineProps<Props>(), {
  title: 'Throughput',
  height: '300px',
  unit: 'samples/sec',
})

const chartRef = ref<HTMLElement>()
let chartInstance: ECharts | null = null

function initChart() {
  if (!chartRef.value) return

  chartInstance = echarts.init(chartRef.value)

  const option: echarts.EChartsOption = {
    title: {
      text: props.title,
      left: 'center',
    },
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        const point = params[0]
        return `${point.name}<br/>${point.seriesName}: ${point.value} ${props.unit}`
      },
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: props.data.map((d) => new Date(d.timestamp).toLocaleTimeString()),
    },
    yAxis: {
      type: 'value',
      name: props.unit,
    },
    series: [
      {
        name: props.title,
        type: 'line',
        smooth: true,
        data: props.data.map((d) => d.value),
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(64, 158, 255, 0.5)' },
            { offset: 1, color: 'rgba(64, 158, 255, 0.1)' },
          ]),
        },
        lineStyle: {
          color: '#409eff',
          width: 2,
        },
        itemStyle: {
          color: '#409eff',
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
      data: props.data.map((d) => new Date(d.timestamp).toLocaleTimeString()),
    },
    series: [
      {
        data: props.data.map((d) => d.value),
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
  () => props.data,
  () => {
    updateChart()
  },
  { deep: true }
)
</script>

<style scoped>
.throughput-chart {
  width: 100%;
}
</style>
