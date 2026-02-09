<template>
  <div class="realtime-data-chart">
    <div ref="chartRef" :style="{ width: '100%', height: height }"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import * as echarts from 'echarts'
import type { ECharts } from 'echarts'
import type { DeviceDataEvent } from '@/services/dataStreamWebSocket'

interface Props {
  deviceId: string
  field: string
  height?: string
  maxDataPoints?: number
  data: DeviceDataEvent[]
}

const props = withDefaults(defineProps<Props>(), {
  height: '300px',
  maxDataPoints: 100,
})

const chartRef = ref<HTMLElement>()
let chartInstance: ECharts | null = null

function initChart() {
  if (!chartRef.value) return

  chartInstance = echarts.init(chartRef.value)

  updateChart()
}

function updateChart() {
  if (!chartInstance) return

  // 提取时间序列数据
  const timestamps: string[] = []
  const values: number[] = []

  props.data.forEach((event) => {
    const value = event.data[props.field]
    if (value !== undefined && value !== null) {
      const numValue = typeof value === 'number' ? value : parseFloat(String(value))
      if (!isNaN(numValue)) {
        timestamps.push(new Date(event.timestamp).toLocaleTimeString())
        values.push(numValue)
      }
    }
  })

  // 限制数据点数量
  if (timestamps.length > props.maxDataPoints) {
    timestamps.splice(0, timestamps.length - props.maxDataPoints)
    values.splice(0, values.length - props.maxDataPoints)
  }

  const option: echarts.EChartsOption = {
    title: {
      text: `${props.field} - ${props.deviceId}`,
      left: 'center',
      textStyle: {
        fontSize: 14,
      },
    },
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        const point = params[0]
        return `${point.name}<br/>${point.seriesName}: ${point.value}`
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
      data: timestamps,
      axisLabel: {
        rotate: 45,
      },
    },
    yAxis: {
      type: 'value',
      name: props.field,
    },
    series: [
      {
        name: props.field,
        type: 'line',
        smooth: true,
        data: values,
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
        animation: false, // 禁用动画以提高性能
      },
    ],
  }

  chartInstance.setOption(option, true) // 使用notMerge=true来完全替换
}

watch(
  () => props.data,
  () => {
    updateChart()
  },
  { deep: true }
)

onMounted(() => {
  initChart()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }
})

function handleResize() {
  if (chartInstance) {
    chartInstance.resize()
  }
}
</script>

<style scoped>
.realtime-data-chart {
  width: 100%;
}
</style>
