<template>
  <div class="devices-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>设备管理</span>
          <el-button type="primary" @click="handleCreate">新增设备</el-button>
        </div>
      </template>

      <!-- 筛选条件 -->
      <el-form :inline="true" :model="filters" class="filter-form">
        <el-form-item label="设备类型">
          <el-select v-model="filters.type" placeholder="全部" clearable style="width: 150px">
            <el-option label="PLC" value="PLC" />
            <el-option label="传感器" value="Sensor" />
            <el-option label="仪表" value="Instrument" />
            <el-option label="智能设备" value="SmartDevice" />
          </el-select>
        </el-form-item>
        <el-form-item label="设备状态">
          <el-select v-model="filters.status" placeholder="全部" clearable style="width: 150px">
            <el-option label="在线" value="online" />
            <el-option label="离线" value="offline" />
            <el-option label="故障" value="fault" />
            <el-option label="维护" value="maintenance" />
          </el-select>
        </el-form-item>
        <el-form-item label="区域">
          <el-input v-model="filters.area" placeholder="区域名称" clearable style="width: 150px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadDevices">查询</el-button>
          <el-button @click="resetFilters">重置</el-button>
        </el-form-item>
      </el-form>

      <!-- 设备列表 -->
      <el-table :data="devices" v-loading="loading" stripe>
        <el-table-column label="设备名称" width="150">
          <template #default="{ row }">
            {{ row.Name || row.name }}
          </template>
        </el-table-column>
        <el-table-column label="设备类型" width="120">
          <template #default="{ row }">
            <el-tag :type="getTypeTagType(row.Type || row.type)">{{ getTypeLabel(row.Type || row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.Status || row.status)">{{ getStatusLabel(row.Status || row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="健康度" width="100">
          <template #default="{ row }">
            <el-progress
              :percentage="row.HealthScore || row.health_score"
              :color="getHealthColor(row.HealthScore || row.health_score)"
              :stroke-width="8"
            />
          </template>
        </el-table-column>
        <el-table-column label="区域" width="120">
          <template #default="{ row }">
            {{ row.Area || row.area }}
          </template>
        </el-table-column>
        <el-table-column label="描述" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.Description || row.description }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="handleView(row)">查看</el-button>
            <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadDevices"
          @current-change="loadDevices"
        />
      </div>
    </el-card>

    <!-- 创建设备对话框 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="600px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="设备名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入设备名称" />
        </el-form-item>
        <el-form-item label="设备类型" prop="type">
          <el-select v-model="form.type" placeholder="请选择设备类型" style="width: 100%">
            <el-option label="PLC" value="PLC" />
            <el-option label="传感器" value="Sensor" />
            <el-option label="仪表" value="Instrument" />
            <el-option label="智能设备" value="SmartDevice" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="请输入设备描述" />
        </el-form-item>
        <el-form-item label="位置">
          <el-row :gutter="10">
            <el-col :span="8">
              <el-input-number v-model="form.location.x" placeholder="X" style="width: 100%" />
            </el-col>
            <el-col :span="8">
              <el-input-number v-model="form.location.y" placeholder="Y" style="width: 100%" />
            </el-col>
            <el-col :span="8">
              <el-input-number v-model="form.location.z" placeholder="Z" style="width: 100%" />
            </el-col>
          </el-row>
        </el-form-item>
        <el-form-item label="区域">
          <el-input v-model="form.area" placeholder="请输入区域名称" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { deviceApi, type Device, type DeviceFilters } from '@/services/deviceApi'

const loading = ref(false)
const devices = ref<Device[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)

const filters = reactive<DeviceFilters>({
  type: undefined,
  status: undefined,
  area: undefined,
})

const dialogVisible = ref(false)
const dialogTitle = ref('新增设备')
const formRef = ref()
const form = reactive({
  name: '',
  type: '',
  description: '',
  location: {
    x: 0,
    y: 0,
    z: 0,
  },
  area: '',
})
const editingId = ref<string | null>(null)

const rules = {
  name: [{ required: true, message: '请输入设备名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择设备类型', trigger: 'change' }],
}

const loadDevices = async () => {
  loading.value = true
  try {
    const result = await deviceApi.getDevices({
      ...filters,
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value,
    })
    devices.value = result.devices
    total.value = result.count
  } catch (error: any) {
    ElMessage.error('加载设备列表失败: ' + (error.response?.data?.error || error.message))
  } finally {
    loading.value = false
  }
}

const resetFilters = () => {
  filters.type = undefined
  filters.status = undefined
  filters.area = undefined
  loadDevices()
}

const handleCreate = () => {
  dialogTitle.value = '新增设备'
  editingId.value = null
  form.name = ''
  form.type = ''
  form.description = ''
  form.location = { x: 0, y: 0, z: 0 }
  form.area = ''
  dialogVisible.value = true
}

const handleEdit = async (device: Device) => {
  dialogTitle.value = '编辑设备'
  editingId.value = device.ID || device.id || ''
  form.name = device.Name || device.name || ''
  form.type = device.Type || device.type || ''
  form.description = device.Description || device.description || ''
  form.location = device.Location ? { x: device.Location.X, y: device.Location.Y, z: device.Location.Z || 0 } : (device.location || { x: 0, y: 0, z: 0 })
  form.area = device.Area || device.area || ''
  dialogVisible.value = true
}

const handleView = async (device: Device) => {
  // TODO: 跳转到设备详情页面
  ElMessage.info('查看设备详情功能待实现')
}

const handleDelete = async (device: Device) => {
  try {
    const deviceName = device.Name || device.name || ''
    const deviceId = device.ID || device.id || ''
    await ElMessageBox.confirm(`确定要删除设备 "${deviceName}" 吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deviceApi.deleteDevice(deviceId)
    ElMessage.success('删除成功')
    loadDevices()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + (error.response?.data?.error || error.message))
    }
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid: boolean) => {
    if (!valid) return

    try {
      if (editingId.value) {
        await deviceApi.updateDevice(editingId.value, form)
        ElMessage.success('更新成功')
      } else {
        await deviceApi.createDevice(form)
        ElMessage.success('创建成功')
      }
      dialogVisible.value = false
      loadDevices()
    } catch (error: any) {
      ElMessage.error((editingId.value ? '更新' : '创建') + '失败: ' + (error.response?.data?.error || error.message))
    }
  })
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

const getHealthColor = (score: number) => {
  if (score >= 80) return '#67c23a'
  if (score >= 60) return '#e6a23c'
  return '#f56c6c'
}

onMounted(() => {
  loadDevices()
})
</script>

<style scoped>
.devices-page {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.filter-form {
  margin-bottom: 20px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
