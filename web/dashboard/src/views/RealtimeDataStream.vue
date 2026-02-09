<template>
  <div class="realtime-data-stream-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>实时数据流</span>
          <div class="header-actions">
            <el-tag :type="connectionStatusType" size="large">
              <el-icon><Connection /></el-icon>
              {{ connectionStatusText }}
            </el-tag>
            <el-button
              :type="isConnected ? 'danger' : 'primary'"
              @click="toggleConnection"
              style="margin-left: 10px"
            >
              {{ isConnected ? '断开连接' : '连接' }}
            </el-button>
          </div>
        </div>
      </template>

      <!-- 过滤器配置 -->
      <el-form :inline="true" :model="filter" class="filter-form">
        <el-form-item label="设备ID">
          <el-select
            v-model="filter.device_ids"
            multiple
            placeholder="选择设备（留空表示全部）"
            clearable
            filterable
            style="width: 300px"
            @change="updateFilter"
          >
            <el-option
              v-for="device in devices"
              :key="device.id"
              :label="device.name"
              :value="device.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="数据字段">
          <el-select
            v-model="filter.fields"
            multiple
            placeholder="选择字段（留空表示全部）"
            clearable
            filterable
            style="width: 300px"
            @change="updateFilter"
          >
            <el-option
              v-for="field in availableFields"
              :key="field"
              :label="field"
              :value="field"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="最小质量">
          <el-input-number
            v-model="filter.min_quality"
            :min="0"
            :max="1"
            :step="0.1"
            :precision="2"
            style="width: 150px"
            @change="updateFilter"
          />
        </el-form-item>
        <el-form-item>
          <el-button @click="clearData">清空数据</el-button>
          <el-button @click="exportData">导出数据</el-button>
        </el-form-item>
      </el-form>

      <!-- 数据统计 -->
      <el-row :gutter="20" style="margin-bottom: 20px">
        <el-col :span="6">
          <el-statistic title="接收数据点" :value="dataPoints.length" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="设备数量" :value="uniqueDevices.size" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="数据字段数" :value="uniqueFields.size" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="平均质量" :value="averageQuality" :precision="2" />
        </el-col>
      </el-row>

      <!-- 设备数据图表 -->
      <el-tabs v-model="activeTab" type="border-card">
        <el-tab-pane
          v-for="deviceId in Array.from(uniqueDevices)"
          :key="deviceId"
          :label="getDeviceName(deviceId)"
          :name="deviceId"
        >
          <div class="charts-container">
            <el-row :gutter="20">
              <el-col
                v-for="field in getDeviceFields(deviceId)"
                :key="field"
                :span="12"
                style="margin-bottom: 20px"
              >
                <RealtimeDataChart
                  :device-id="deviceId"
                  :field="field"
                  :data="getDeviceData(deviceId)"
                  :max-data-points="maxDataPoints"
                  height="300px"
                />
              </el-col>
            </el-row>
          </div>
        </el-tab-pane>
      </el-tabs>

      <!-- 数据表格 -->
      <el-card shadow="hover" style="margin-top: 20px">
        <template #header>
          <span>最新数据</span>
          <el-button
            link
            type="primary"
            style="float: right"
            @click="showDataTable = !showDataTable"
          >
            {{ showDataTable ? '隐藏' : '显示' }}
          </el-button>
        </template>
        <el-table
          v-if="showDataTable"
          :data="recentData"
          stripe
          max-height="400px"
          style="width: 100%"
        >
          <el-table-column prop="device_id" label="设备ID" width="150" />
          <el-table-column prop="timestamp" label="时间戳" width="180">
            <template #default="{ row }">
              {{ new Date(row.timestamp).toLocaleString() }}
            </template>
          </el-table-column>
          <el-table-column label="数据" min-width="300">
            <template #default="{ row }">
              <pre style="margin: 0; font-size: 12px">{{ JSON.stringify(row.data, null, 2) }}</pre>
            </template>
          </el-table-column>
          <el-table-column prop="quality" label="质量" width="100">
            <template #default="{ row }">
              <el-progress
                :percentage="(row.quality || 0) * 100"
                :color="getQualityColor(row.quality || 0)"
                :stroke-width="8"
              />
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection } from '@element-plus/icons-vue'
import { getDataStreamWebSocketService, type DeviceDataEvent } from '@/services/dataStreamWebSocket'
import { deviceApi, type Device } from '@/services/deviceApi'
import RealtimeDataChart from '@/components/RealtimeDataChart.vue'

const wsService = getDataStreamWebSocketService()

const isConnected = ref(false)
const dataPoints = ref<DeviceDataEvent[]>([])
const devices = ref<Device[]>([])
const showDataTable = ref(false)
const activeTab = ref('')
const maxDataPoints = ref(100)

const filter = ref<{
  device_ids?: string[]
  source_ids?: string[]
  fields?: string[]
  min_quality?: number
}>({})

// 计算属性
const connectionStatusType = computed(() => {
  return isConnected.value ? 'success' : 'danger'
})

const connectionStatusText = computed(() => {
  return isConnected.value ? '已连接' : '未连接'
})

const uniqueDevices = computed(() => {
  const devices = new Set<string>()
  dataPoints.value.forEach((point) => {
    devices.add(point.device_id)
  })
  return devices
})

const uniqueFields = computed(() => {
  const fields = new Set<string>()
  dataPoints.value.forEach((point) => {
    Object.keys(point.data).forEach((field) => fields.add(field))
  })
  return fields
})

const availableFields = computed(() => {
  return Array.from(uniqueFields.value)
})

const averageQuality = computed(() => {
  if (dataPoints.value.length === 0) return 0
  const sum = dataPoints.value.reduce((acc, point) => acc + (point.quality || 0), 0)
  return sum / dataPoints.value.length
})

const recentData = computed(() => {
  return dataPoints.value.slice(-50).reverse()
})

// 方法
function toggleConnection() {
  if (isConnected.value) {
    wsService.disconnect()
    isConnected.value = false
    ElMessage.info('已断开连接')
  } else {
    wsService
      .connect()
      .then(() => {
        isConnected.value = true
        ElMessage.success('连接成功')
        updateFilter()
      })
      .catch((error) => {
        ElMessage.error(`连接失败: ${error.message}`)
      })
  }
}

function updateFilter() {
  if (isConnected.value) {
    wsService.setFilter(filter.value)
  }
}

function handleDataEvent(event: DeviceDataEvent) {
  dataPoints.value.push(event)

  // 限制数据点数量
  if (dataPoints.value.length > maxDataPoints.value * 10) {
    dataPoints.value.splice(0, dataPoints.value.length - maxDataPoints.value * 10)
  }

  // 自动切换到对应的设备标签页
  if (!activeTab.value || !uniqueDevices.value.has(activeTab.value)) {
    activeTab.value = event.device_id
  }
}

function getDeviceName(deviceId: string): string {
  const device = devices.value.find((d) => d.id === deviceId)
  return device ? device.name : deviceId
}

function getDeviceFields(deviceId: string): string[] {
  const fields = new Set<string>()
  dataPoints.value
    .filter((point) => point.device_id === deviceId)
    .forEach((point) => {
      Object.keys(point.data).forEach((field) => fields.add(field))
    })
  return Array.from(fields)
}

function getDeviceData(deviceId: string): DeviceDataEvent[] {
  return dataPoints.value.filter((point) => point.device_id === deviceId)
}

function getQualityColor(quality: number): string {
  if (quality >= 0.8) return '#67c23a'
  if (quality >= 0.5) return '#e6a23c'
  return '#f56c6c'
}

function clearData() {
  dataPoints.value = []
  ElMessage.success('数据已清空')
}

function exportData() {
  const dataStr = JSON.stringify(dataPoints.value, null, 2)
  const blob = new Blob([dataStr], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `realtime-data-${new Date().toISOString()}.json`
  link.click()
  URL.revokeObjectURL(url)
  ElMessage.success('数据已导出')
}

// 生命周期
onMounted(async () => {
  // 加载设备列表
  try {
    const response = await deviceApi.getDevices()
    devices.value = response.devices || []
  } catch (error: any) {
    ElMessage.error(`加载设备列表失败: ${error.message}`)
  }

  // 订阅数据事件
  wsService.subscribe(handleDataEvent)

  // 检查连接状态
  isConnected.value = wsService.isConnected
  if (!isConnected.value) {
    // 自动连接
    toggleConnection()
  }
})

onUnmounted(() => {
  wsService.unsubscribe(handleDataEvent)
  if (isConnected.value) {
    wsService.disconnect()
  }
})
</script>

<style scoped>
.realtime-data-stream-page {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  align-items: center;
}

.filter-form {
  margin-bottom: 20px;
}

.charts-container {
  padding: 20px 0;
}
</style>
