<template>
  <div class="device-layout-map" ref="containerRef">
    <svg ref="svgRef" :width="width" :height="height" class="layout-svg">
      <!-- 背景网格 -->
      <defs>
        <pattern id="grid" width="20" height="20" patternUnits="userSpaceOnUse">
          <path d="M 20 0 L 0 0 0 20" fill="none" stroke="#e0e0e0" stroke-width="1" />
        </pattern>
      </defs>
      <rect width="100%" height="100%" fill="url(#grid)" />

      <!-- 设备节点 -->
      <g v-for="device in devices" :key="device.id" class="device-node">
        <circle
          :cx="device.location.x"
          :cy="device.location.y"
          :r="getDeviceRadius(device)"
          :fill="getDeviceColor(device)"
          :stroke="getDeviceStrokeColor(device)"
          :stroke-width="getDeviceStrokeWidth(device)"
          class="device-circle"
          @click="handleDeviceClick(device)"
          @mouseenter="handleDeviceHover(device, $event)"
          @mouseleave="handleDeviceHover(null, null)"
        />
        <text
          :x="device.location.x"
          :y="device.location.y - getDeviceRadius(device) - 5"
          text-anchor="middle"
          class="device-label"
          font-size="12"
          font-weight="bold"
        >
          {{ device.name }}
        </text>
        <text
          :x="device.location.x"
          :y="device.location.y + getDeviceRadius(device) + 15"
          text-anchor="middle"
          class="device-status"
          font-size="10"
          :fill="getDeviceColor(device)"
        >
          {{ getStatusLabel(device.status) }}
        </text>
      </g>

      <!-- 设备连接线（如果有通道关联） -->
      <g v-if="showConnections">
        <line
          v-for="connection in connections"
          :key="connection.id"
          :x1="connection.x1"
          :y1="connection.y1"
          :x2="connection.x2"
          :y2="connection.y2"
          stroke="#999"
          stroke-width="1"
          stroke-dasharray="3,3"
          opacity="0.5"
        />
      </g>

      <!-- 工具提示 -->
      <foreignObject
        v-if="hoveredDevice"
        :x="tooltipX"
        :y="tooltipY"
        width="200"
        height="150"
        class="tooltip"
      >
        <div class="tooltip-content">
          <h4>{{ hoveredDevice.name }}</h4>
          <p><strong>类型:</strong> {{ getTypeLabel(hoveredDevice.type) }}</p>
          <p><strong>状态:</strong> {{ getStatusLabel(hoveredDevice.status) }}</p>
          <p><strong>健康度:</strong> {{ hoveredDevice.health_score.toFixed(1) }}%</p>
          <p v-if="hoveredDevice.area"><strong>区域:</strong> {{ hoveredDevice.area }}</p>
        </div>
      </foreignObject>
    </svg>

    <!-- 图例 -->
    <div class="legend">
      <div class="legend-item">
        <div class="legend-color" style="background-color: #67c23a"></div>
        <span>在线</span>
      </div>
      <div class="legend-item">
        <div class="legend-color" style="background-color: #909399"></div>
        <span>离线</span>
      </div>
      <div class="legend-item">
        <div class="legend-color" style="background-color: #f56c6c"></div>
        <span>故障</span>
      </div>
      <div class="legend-item">
        <div class="legend-color" style="background-color: #e6a23c"></div>
        <span>维护</span>
      </div>
    </div>

    <!-- 控制按钮 -->
    <div class="controls">
      <el-button size="small" @click="handleZoomIn">放大</el-button>
      <el-button size="small" @click="handleZoomOut">缩小</el-button>
      <el-button size="small" @click="handleReset">重置</el-button>
      <el-button size="small" @click="showConnections = !showConnections">
        {{ showConnections ? '隐藏连接' : '显示连接' }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import type { Device } from '@/services/deviceApi'

interface Props {
  devices: Device[]
  width?: number
  height?: number
}

const props = withDefaults(defineProps<Props>(), {
  width: 1200,
  height: 800,
})

const emit = defineEmits<{
  deviceClick: [device: Device]
}>()

const containerRef = ref<HTMLElement>()
const svgRef = ref<SVGSVGElement>()
const hoveredDevice = ref<Device | null>(null)
const tooltipX = ref(0)
const tooltipY = ref(0)
const showConnections = ref(false)
const zoom = ref(1)
const panX = ref(0)
const panY = ref(0)

const connections = computed(() => {
  // TODO: 根据设备与通道的关联关系生成连接线
  return []
})

const handleDeviceClick = (device: Device) => {
  emit('deviceClick', device)
}

const handleDeviceHover = (device: Device | null, event: MouseEvent | null) => {
  hoveredDevice.value = device
  if (event && device) {
    tooltipX.value = event.offsetX + 10
    tooltipY.value = event.offsetY + 10
  }
}

const getDeviceRadius = (device: Device) => {
  // 根据设备健康度调整大小
  const baseRadius = 20
  const healthMultiplier = device.health_score / 100
  return baseRadius + healthMultiplier * 10
}

const getDeviceColor = (device: Device) => {
  const colorMap: Record<string, string> = {
    online: '#67c23a',
    offline: '#909399',
    fault: '#f56c6c',
    maintenance: '#e6a23c',
  }
  return colorMap[device.status] || '#909399'
}

const getDeviceStrokeColor = (device: Device) => {
  if (device.status === 'fault') return '#f56c6c'
  if (device.status === 'maintenance') return '#e6a23c'
  return '#fff'
}

const getDeviceStrokeWidth = (device: Device) => {
  if (device.status === 'fault') return 3
  return 2
}

const getTypeLabel = (type: string) => {
  const map: Record<string, string> = {
    PLC: 'PLC',
    Sensor: '传感器',
    Instrument: '仪表',
    SmartDevice: '智能设备',
  }
  return map[type] || type
}

const getStatusLabel = (status: string) => {
  const map: Record<string, string> = {
    online: '在线',
    offline: '离线',
    fault: '故障',
    maintenance: '维护',
  }
  return map[status] || status
}

const handleZoomIn = () => {
  zoom.value = Math.min(zoom.value * 1.2, 3)
  updateTransform()
}

const handleZoomOut = () => {
  zoom.value = Math.max(zoom.value / 1.2, 0.5)
  updateTransform()
}

const handleReset = () => {
  zoom.value = 1
  panX.value = 0
  panY.value = 0
  updateTransform()
}

const updateTransform = () => {
  if (svgRef.value) {
    const g = svgRef.value.querySelector('g.device-nodes')
    if (g) {
      ;(g as SVGGElement).setAttribute(
        'transform',
        `translate(${panX.value}, ${panY.value}) scale(${zoom.value})`
      )
    }
  }
}

watch(
  () => props.devices,
  () => {
    // 设备列表更新时，可以重新计算布局
  },
  { deep: true }
)

onMounted(() => {
  // 初始化
})
</script>

<style scoped>
.device-layout-map {
  position: relative;
  width: 100%;
  height: 100%;
  background-color: #f5f7fa;
  border-radius: 8px;
  overflow: hidden;
}

.layout-svg {
  width: 100%;
  height: 100%;
  cursor: move;
}

.device-node {
  cursor: pointer;
  transition: all 0.2s;
}

.device-node:hover {
  opacity: 0.8;
}

.device-circle {
  transition: all 0.2s;
}

.device-label {
  fill: #333;
  pointer-events: none;
}

.device-status {
  pointer-events: none;
}

.tooltip {
  pointer-events: none;
}

.tooltip-content {
  background: white;
  border: 1px solid #ddd;
  border-radius: 4px;
  padding: 10px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.tooltip-content h4 {
  margin: 0 0 8px 0;
  font-size: 14px;
  font-weight: bold;
}

.tooltip-content p {
  margin: 4px 0;
  font-size: 12px;
}

.legend {
  position: absolute;
  top: 10px;
  right: 10px;
  background: white;
  padding: 10px;
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.legend-item {
  display: flex;
  align-items: center;
  margin-bottom: 5px;
}

.legend-color {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  margin-right: 8px;
  border: 2px solid #fff;
}

.controls {
  position: absolute;
  bottom: 10px;
  left: 10px;
  display: flex;
  gap: 10px;
}
</style>
