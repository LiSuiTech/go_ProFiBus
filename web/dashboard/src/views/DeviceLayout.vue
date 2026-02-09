<template>
  <div class="device-layout-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>设备布局可视化</span>
          <div>
            <el-select v-model="selectedArea" placeholder="选择区域" clearable style="width: 200px" @change="loadLayout">
              <el-option
                v-for="area in areas"
                :key="area"
                :label="area"
                :value="area"
              />
            </el-select>
            <el-button type="primary" @click="handleEditLayout" style="margin-left: 10px">
              编辑布局
            </el-button>
          </div>
        </div>
      </template>

      <!-- 设备布局图 -->
      <div class="layout-container">
        <DeviceLayoutMap
          :devices="devices"
          :width="layoutWidth"
          :height="layoutHeight"
          @device-click="handleDeviceClick"
        />
      </div>

      <!-- 设备列表侧边栏 -->
      <el-drawer v-model="showDeviceList" title="设备列表" :size="400">
        <el-table :data="devices" stripe>
          <el-table-column prop="name" label="设备名称" width="120" />
          <el-table-column prop="type" label="类型" width="100">
            <template #default="{ row }">
              <el-tag :type="getTypeTagType(row.type)">{{ getTypeLabel(row.type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusTagType(row.status)">{{ getStatusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="health_score" label="健康度" width="100">
            <template #default="{ row }">
              <el-progress
                :percentage="row.health_score"
                :color="getHealthColor(row.health_score)"
                :stroke-width="8"
              />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button link type="primary" @click="handleLocateDevice(row)">定位</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-drawer>

      <!-- 设备详情对话框 -->
      <el-dialog v-model="showDeviceDetail" :title="selectedDevice?.name" width="600px">
        <div v-if="selectedDevice" class="device-detail">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="设备ID">{{ selectedDevice.id }}</el-descriptions-item>
            <el-descriptions-item label="设备类型">
              <el-tag :type="getTypeTagType(selectedDevice.type)">
                {{ getTypeLabel(selectedDevice.type) }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="getStatusTagType(selectedDevice.status)">
                {{ getStatusLabel(selectedDevice.status) }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="健康度">
              <el-progress
                :percentage="selectedDevice.health_score"
                :color="getHealthColor(selectedDevice.health_score)"
              />
            </el-descriptions-item>
            <el-descriptions-item label="位置">
              X: {{ selectedDevice.location.x }}, Y: {{ selectedDevice.location.y }}
              <span v-if="selectedDevice.location.z">, Z: {{ selectedDevice.location.z }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="区域">{{ selectedDevice.area || '-' }}</el-descriptions-item>
            <el-descriptions-item label="描述" :span="2">
              {{ selectedDevice.description || '-' }}
            </el-descriptions-item>
          </el-descriptions>

          <el-divider>关联通道</el-divider>
          <el-tag
            v-for="channelId in deviceChannels[selectedDevice.id] || []"
            :key="channelId"
            style="margin-right: 8px; margin-bottom: 8px"
          >
            {{ channelId }}
          </el-tag>
          <el-empty v-if="!deviceChannels[selectedDevice.id]?.length" description="暂无关联通道" />
        </div>
      </el-dialog>

      <!-- 编辑布局对话框 -->
      <el-dialog v-model="showEditLayout" title="编辑设备布局" width="800px">
        <div class="edit-layout-container">
          <div class="edit-canvas" ref="editCanvasRef">
            <svg :width="layoutWidth" :height="layoutHeight" class="edit-svg">
              <rect width="100%" height="100%" fill="#f5f7fa" />
              <g v-for="device in devices" :key="device.id">
                <circle
                  :cx="device.location.x"
                  :cy="device.location.y"
                  r="15"
                  :fill="getDeviceColor(device)"
                  :stroke="#333"
                  stroke-width="2"
                  class="draggable-device"
                  :data-device-id="device.id"
                  @mousedown="handleDragStart($event, device)"
                />
                <text
                  :x="device.location.x"
                  :y="device.location.y - 20"
                  text-anchor="middle"
                  font-size="12"
                  font-weight="bold"
                >
                  {{ device.name }}
                </text>
              </g>
            </svg>
          </div>
          <div class="edit-controls">
            <el-button @click="showEditLayout = false">取消</el-button>
            <el-button type="primary" @click="handleSaveLayout">保存</el-button>
          </div>
        </div>
      </el-dialog>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { deviceApi, type Device } from '@/services/deviceApi'
import DeviceLayoutMap from '@/components/DeviceLayoutMap.vue'

const loading = ref(false)
const devices = ref<Device[]>([])
const areas = ref<string[]>([])
const selectedArea = ref<string>()
const showDeviceList = ref(false)
const showDeviceDetail = ref(false)
const showEditLayout = ref(false)
const selectedDevice = ref<Device | null>(null)
const deviceChannels = ref<Record<string, string[]>>({})
const layoutWidth = ref(1200)
const layoutHeight = ref(800)
const editCanvasRef = ref<HTMLElement>()

const draggingDevice = ref<Device | null>(null)
const dragOffset = ref({ x: 0, y: 0 })

const loadLayout = async () => {
  loading.value = true
  try {
    const result = await deviceApi.getDeviceLayout(selectedArea.value)
    devices.value = result.devices

    // 加载每个设备的通道关联
    for (const device of devices.value) {
      try {
        const deviceDetail = await deviceApi.getDevice(device.id)
        deviceChannels.value[device.id] = deviceDetail.channel_ids || []
      } catch (error) {
        console.error(`Failed to load channels for device ${device.id}:`, error)
      }
    }

    // 提取区域列表
    const areaSet = new Set<string>()
    devices.value.forEach((d) => {
      if (d.area) areaSet.add(d.area)
    })
    areas.value = Array.from(areaSet).sort()
  } catch (error: any) {
    ElMessage.error('加载设备布局失败: ' + (error.response?.data?.error || error.message))
  } finally {
    loading.value = false
  }
}

const handleDeviceClick = async (device: Device) => {
  selectedDevice.value = device
  showDeviceDetail.value = true

  // 加载设备详情和通道关联
  try {
    const deviceDetail = await deviceApi.getDevice(device.id)
    deviceChannels.value[device.id] = deviceDetail.channel_ids || []
  } catch (error) {
    console.error('Failed to load device detail:', error)
  }
}

const handleLocateDevice = (device: Device) => {
  // TODO: 在布局图上定位设备（滚动到设备位置）
  ElMessage.info(`定位设备: ${device.name}`)
}

const handleEditLayout = () => {
  showEditLayout.value = true
}

const handleDragStart = (event: MouseEvent, device: Device) => {
  draggingDevice.value = device
  const rect = (event.target as SVGCircleElement).getBoundingClientRect()
  dragOffset.value = {
    x: event.clientX - rect.left - device.location.x,
    y: event.clientY - rect.top - device.location.y,
  }

  const handleMouseMove = (e: MouseEvent) => {
    if (draggingDevice.value && editCanvasRef.value) {
      const svgRect = editCanvasRef.value.getBoundingClientRect()
      const newX = e.clientX - svgRect.left - dragOffset.value.x
      const newY = e.clientY - svgRect.top - dragOffset.value.y

      // 更新设备位置
      const deviceIndex = devices.value.findIndex((d) => d.id === draggingDevice.value!.id)
      if (deviceIndex !== -1) {
        devices.value[deviceIndex].location.x = Math.max(0, Math.min(newX, layoutWidth.value))
        devices.value[deviceIndex].location.y = Math.max(0, Math.min(newY, layoutHeight.value))
      }
    }
  }

  const handleMouseUp = () => {
    draggingDevice.value = null
    document.removeEventListener('mousemove', handleMouseMove)
    document.removeEventListener('mouseup', handleMouseUp)
  }

  document.addEventListener('mousemove', handleMouseMove)
  document.addEventListener('mouseup', handleMouseUp)
}

const handleSaveLayout = async () => {
  try {
    // 批量更新设备位置
    for (const device of devices.value) {
      await deviceApi.updateDevice(device.id, {
        location: device.location,
      })
    }
    ElMessage.success('布局保存成功')
    showEditLayout.value = false
    loadLayout() // 重新加载布局
  } catch (error: any) {
    ElMessage.error('保存布局失败: ' + (error.response?.data?.error || error.message))
  }
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

const getTypeTagType = (type: string) => {
  const map: Record<string, string> = {
    PLC: 'primary',
    Sensor: 'success',
    Instrument: 'warning',
    SmartDevice: 'info',
  }
  return map[type] || ''
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

const getStatusTagType = (status: string) => {
  const map: Record<string, string> = {
    online: 'success',
    offline: 'info',
    fault: 'danger',
    maintenance: 'warning',
  }
  return map[status] || ''
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

const getHealthColor = (score: number) => {
  if (score >= 80) return '#67c23a'
  if (score >= 60) return '#e6a23c'
  return '#f56c6c'
}

let refreshInterval: number | null = null

onMounted(() => {
  loadLayout()
  // 定时刷新设备状态
  refreshInterval = window.setInterval(() => {
    loadLayout()
  }, 30000) // 每30秒刷新一次
})

onUnmounted(() => {
  if (refreshInterval !== null) {
    clearInterval(refreshInterval)
  }
})
</script>

<style scoped>
.device-layout-page {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.layout-container {
  width: 100%;
  height: 800px;
  border: 1px solid #ddd;
  border-radius: 4px;
  overflow: auto;
  background-color: #fafafa;
}

.device-detail {
  padding: 10px 0;
}

.edit-layout-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.edit-canvas {
  width: 100%;
  height: 600px;
  border: 1px solid #ddd;
  border-radius: 4px;
  overflow: auto;
  background-color: #fafafa;
}

.edit-svg {
  width: 100%;
  height: 100%;
}

.draggable-device {
  cursor: move;
}

.edit-controls {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
