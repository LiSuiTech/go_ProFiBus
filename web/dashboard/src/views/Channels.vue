<template>
  <div class="channels-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span><el-icon><Connection /></el-icon> 采集通道管理</span>
          <el-button type="primary" @click="showAddChannelDialog">
            <el-icon><Plus /></el-icon> 新增通道
          </el-button>
        </div>
      </template>

      <!-- 通道统计 -->
      <el-row :gutter="20" class="stats-row">
        <el-col :span="6">
          <el-statistic title="总通道数" :value="channels.length" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="运行中" :value="runningCount">
            <template #suffix>
              <el-icon color="#67C23A"><CircleCheck /></el-icon>
            </template>
          </el-statistic>
        </el-col>
        <el-col :span="6">
          <el-statistic title="已停止" :value="stoppedCount">
            <template #suffix>
              <el-icon color="#909399"><CircleClose /></el-icon>
            </template>
          </el-statistic>
        </el-col>
        <el-col :span="6">
          <el-statistic title="错误" :value="errorCount">
            <template #suffix>
              <el-icon color="#F56C6C"><Warning /></el-icon>
            </template>
          </el-statistic>
        </el-col>
      </el-row>

      <!-- 通道列表 -->
      <el-table :data="channels" style="width: 100%; margin-top: 20px" v-loading="loading">
        <el-table-column prop="name" label="通道名称" min-width="150" />
        <el-table-column prop="protocol" label="协议类型" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.protocol }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="device_name" label="设备名称" width="150" />
        <el-table-column prop="device_port" label="设备端口" width="150" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag
              :type="getStatusType(row.status)"
              size="small"
            >
              {{ getStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="点位数量" width="100">
          <template #default="{ row }">
            <el-button link type="primary" @click="showPointsDialog(row)">
              {{ row.points?.length || 0 }} 个
            </el-button>
          </template>
        </el-table-column>
        <el-table-column prop="enabled" label="启用" width="80">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
              {{ row.enabled ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="300">
          <template #default="{ row }">
            <el-button-group>
              <el-button
                v-if="row.status !== 'running'"
                type="success"
                size="small"
                @click="handleStart(row)"
              >
                <el-icon><VideoPlay /></el-icon> 启动
              </el-button>
              <el-button
                v-else
                type="warning"
                size="small"
                @click="handleStop(row)"
              >
                <el-icon><VideoPause /></el-icon> 停止
              </el-button>
              <el-button
                type="primary"
                size="small"
                @click="showEditChannelDialog(row)"
              >
                <el-icon><Edit /></el-icon> 编辑
              </el-button>
              <el-button
                type="danger"
                size="small"
                @click="handleDelete(row)"
              >
                <el-icon><Delete /></el-icon> 删除
              </el-button>
            </el-button-group>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新增/编辑通道对话框 -->
    <el-dialog
      v-model="channelDialogVisible"
      :title="isEditing ? '编辑通道' : '新增通道'"
      width="600px"
    >
      <el-form
        ref="channelFormRef"
        :model="channelForm"
        :rules="channelRules"
        label-width="120px"
      >
        <el-form-item label="通道名称" prop="name">
          <el-input v-model="channelForm.name" placeholder="请输入通道名称" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input
            v-model="channelForm.description"
            type="textarea"
            placeholder="请输入描述信息"
          />
        </el-form-item>
        <el-form-item label="协议类型" prop="protocol">
          <el-select
            v-model="channelForm.protocol"
            placeholder="请选择协议"
            @change="handleProtocolChange"
          >
            <el-option
              v-for="protocol in protocols"
              :key="protocol.type"
              :label="`${protocol.name} - ${protocol.description}`"
              :value="protocol.type"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="设备名称" prop="device_name">
          <el-input v-model="channelForm.device_name" placeholder="请输入设备名称" />
        </el-form-item>
        <el-form-item label="设备端口" prop="device_port">
          <el-input
            v-model="channelForm.device_port"
            placeholder="如 /dev/ttyUSB0, COM1"
          />
        </el-form-item>

        <!-- 动态协议配置字段 -->
        <el-divider>协议配置</el-divider>
        <template v-if="selectedProtocol">
          <el-form-item
            v-for="field in selectedProtocol.config_fields"
            :key="field.key"
            :label="field.label"
            :prop="`config.${field.key}`"
          >
            <el-input
              v-if="field.type === 'text' || field.type === 'number'"
              v-model="channelForm.config[field.key]"
              :type="field.type"
              :placeholder="`请输入${field.label}`"
            />
            <el-select
              v-else-if="field.type === 'select'"
              v-model="channelForm.config[field.key]"
              :placeholder="`请选择${field.label}`"
            >
              <el-option
                v-for="option in field.options"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </el-select>
          </el-form-item>
        </template>

        <el-form-item label="启用">
          <el-switch v-model="channelForm.enabled" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="channelDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveChannel">确定</el-button>
      </template>
    </el-dialog>

    <!-- 点位管理对话框 -->
    <el-dialog
      v-model="pointsDialogVisible"
      :title="`${currentChannel?.name} - 点位管理`"
      width="900px"
    >
      <el-button
        type="primary"
        style="margin-bottom: 15px"
        @click="showAddPointDialog"
      >
        <el-icon><Plus /></el-icon> 新增点位
      </el-button>

      <el-table :data="currentChannel?.points" style="width: 100%">
        <el-table-column prop="name" label="点位名称" width="150" />
        <el-table-column prop="address" label="地址" width="100" />
        <el-table-column prop="data_type" label="数据类型" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.data_type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="unit" label="单位" width="80" />
        <el-table-column prop="scale" label="缩放系数" width="100" />
        <el-table-column prop="offset" label="偏移量" width="100" />
        <el-table-column prop="enabled" label="启用" width="80">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
              {{ row.enabled ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="150">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="showEditPointDialog(row)">
              编辑
            </el-button>
            <el-button type="danger" size="small" @click="handleDeletePoint(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 新增/编辑点位对话框 -->
    <el-dialog
      v-model="pointDialogVisible"
      :title="isEditingPoint ? '编辑点位' : '新增点位'"
      width="500px"
    >
      <el-form
        ref="pointFormRef"
        :model="pointForm"
        :rules="pointRules"
        label-width="100px"
      >
        <el-form-item label="点位名称" prop="name">
          <el-input v-model="pointForm.name" placeholder="请输入点位名称" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="pointForm.description" placeholder="请输入描述" />
        </el-form-item>
        <el-form-item label="地址" prop="address">
          <el-input v-model="pointForm.address" placeholder="如 40001, 0x10" />
        </el-form-item>
        <el-form-item label="数据类型" prop="data_type">
          <el-select v-model="pointForm.data_type" placeholder="请选择数据类型">
            <el-option label="整数 (int)" value="int" />
            <el-option label="浮点数 (float)" value="float" />
            <el-option label="布尔 (bool)" value="bool" />
            <el-option label="字符串 (string)" value="string" />
            <el-option label="字节 (bytes)" value="bytes" />
          </el-select>
        </el-form-item>
        <el-form-item label="单位" prop="unit">
          <el-input v-model="pointForm.unit" placeholder="如 ℃, Pa, m/s" />
        </el-form-item>
        <el-form-item label="缩放系数" prop="scale">
          <el-input-number v-model="pointForm.scale" :precision="2" :step="0.1" />
        </el-form-item>
        <el-form-item label="偏移量" prop="offset">
          <el-input-number v-model="pointForm.offset" :precision="2" :step="0.1" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="pointForm.enabled" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="pointDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSavePoint">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import {
  Connection,
  Plus,
  Edit,
  Delete,
  VideoPlay,
  VideoPause,
  CircleCheck,
  CircleClose,
  Warning,
} from '@element-plus/icons-vue'
import type {
  Channel,
  Point,
  ProtocolInfo,
  ProtocolType,
  CreateChannelRequest,
  UpdateChannelRequest,
  CreatePointRequest,
  UpdatePointRequest,
} from '../types'
import * as channelApi from '../services/channelApi'

// 数据
const channels = ref<Channel[]>([])
const protocols = ref<ProtocolInfo[]>([])
const loading = ref(false)

// 对话框控制
const channelDialogVisible = ref(false)
const pointsDialogVisible = ref(false)
const pointDialogVisible = ref(false)
const isEditing = ref(false)
const isEditingPoint = ref(false)

// 当前操作对象
const currentChannel = ref<Channel | null>(null)
const selectedProtocol = ref<ProtocolInfo | null>(null)

// 表单引用
const channelFormRef = ref<FormInstance>()
const pointFormRef = ref<FormInstance>()

// 通道表单
const channelForm = ref<CreateChannelRequest>({
  name: '',
  description: '',
  protocol: 'UART' as ProtocolType,
  device_name: '',
  device_port: '',
  config: {},
  enabled: true,
})

// 点位表单
const pointForm = ref<CreatePointRequest>({
  name: '',
  description: '',
  address: '',
  data_type: 'float',
  unit: '',
  scale: 1.0,
  offset: 0.0,
  enabled: true,
})

// 表单验证规则
const channelRules: FormRules = {
  name: [{ required: true, message: '请输入通道名称', trigger: 'blur' }],
  protocol: [{ required: true, message: '请选择协议类型', trigger: 'change' }],
  device_name: [{ required: true, message: '请输入设备名称', trigger: 'blur' }],
  device_port: [{ required: true, message: '请输入设备端口', trigger: 'blur' }],
}

const pointRules: FormRules = {
  name: [{ required: true, message: '请输入点位名称', trigger: 'blur' }],
  address: [{ required: true, message: '请输入地址', trigger: 'blur' }],
  data_type: [{ required: true, message: '请选择数据类型', trigger: 'change' }],
}

// 计算属性
const runningCount = computed(() =>
  channels.value.filter((c) => c.status === 'running').length
)
const stoppedCount = computed(() =>
  channels.value.filter((c) => c.status === 'stopped').length
)
const errorCount = computed(() => channels.value.filter((c) => c.status === 'error').length)

// 初始化
onMounted(() => {
  fetchProtocols()
  fetchChannels()
})

// 获取协议列表
const fetchProtocols = async () => {
  try {
    const data = await channelApi.getProtocols()
    protocols.value = data.protocols
  } catch (error: any) {
    ElMessage.error(`获取协议列表失败: ${error.message}`)
  }
}

// 获取通道列表
const fetchChannels = async () => {
  loading.value = true
  try {
    const data = await channelApi.getChannels()
    channels.value = data.channels
  } catch (error: any) {
    ElMessage.error(`获取通道列表失败: ${error.message}`)
  } finally {
    loading.value = false
  }
}

// 协议变更处理
const handleProtocolChange = (protocol: ProtocolType) => {
  selectedProtocol.value =
    protocols.value.find((p) => p.type === protocol) || null
  // 重置配置
  channelForm.value.config = {}
  // 设置默认值
  selectedProtocol.value?.config_fields.forEach((field) => {
    if (field.default_value !== undefined) {
      channelForm.value.config[field.key] = field.default_value
    }
  })
}

// 显示新增通道对话框
const showAddChannelDialog = () => {
  isEditing.value = false
  channelForm.value = {
    name: '',
    description: '',
    protocol: 'UART',
    device_name: '',
    device_port: '',
    config: {},
    enabled: true,
  }
  handleProtocolChange('UART')
  channelDialogVisible.value = true
}

// 显示编辑通道对话框
const showEditChannelDialog = (channel: Channel) => {
  isEditing.value = true
  currentChannel.value = channel
  channelForm.value = {
    name: channel.name,
    description: channel.description,
    protocol: channel.protocol,
    device_name: channel.device_name,
    device_port: channel.device_port,
    config: { ...channel.config },
    enabled: channel.enabled,
  }
  selectedProtocol.value =
    protocols.value.find((p) => p.type === channel.protocol) || null
  channelDialogVisible.value = true
}

// 保存通道
const handleSaveChannel = async () => {
  await channelFormRef.value?.validate(async (valid) => {
    if (!valid) return

    try {
      if (isEditing.value && currentChannel.value) {
        await channelApi.updateChannel(currentChannel.value.id, channelForm.value)
        ElMessage.success('通道更新成功')
      } else {
        await channelApi.createChannel(channelForm.value)
        ElMessage.success('通道创建成功')
      }
      channelDialogVisible.value = false
      fetchChannels()
    } catch (error: any) {
      ElMessage.error(`操作失败: ${error.message}`)
    }
  })
}

// 删除通道
const handleDelete = async (channel: Channel) => {
  try {
    await ElMessageBox.confirm(`确定要删除通道 "${channel.name}" 吗？`, '警告', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    await channelApi.deleteChannel(channel.id)
    ElMessage.success('通道删除成功')
    fetchChannels()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`删除失败: ${error.message}`)
    }
  }
}

// 启动通道
const handleStart = async (channel: Channel) => {
  try {
    await channelApi.startChannel(channel.id)
    ElMessage.success('通道启动成功')
    fetchChannels()
  } catch (error: any) {
    ElMessage.error(`启动失败: ${error.message}`)
  }
}

// 停止通道
const handleStop = async (channel: Channel) => {
  try {
    await channelApi.stopChannel(channel.id)
    ElMessage.success('通道停止成功')
    fetchChannels()
  } catch (error: any) {
    ElMessage.error(`停止失败: ${error.message}`)
  }
}

// 显示点位管理对话框
const showPointsDialog = async (channel: Channel) => {
  try {
    const updatedChannel = await channelApi.getChannel(channel.id)
    currentChannel.value = updatedChannel
    pointsDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error(`获取点位信息失败: ${error.message}`)
  }
}

// 显示新增点位对话框
const showAddPointDialog = () => {
  isEditingPoint.value = false
  pointForm.value = {
    name: '',
    description: '',
    address: '',
    data_type: 'float',
    unit: '',
    scale: 1.0,
    offset: 0.0,
    enabled: true,
  }
  pointDialogVisible.value = true
}

// 显示编辑点位对话框
const showEditPointDialog = (point: Point) => {
  isEditingPoint.value = true
  pointForm.value = {
    name: point.name,
    description: point.description,
    address: point.address,
    data_type: point.data_type,
    unit: point.unit,
    scale: point.scale,
    offset: point.offset,
    enabled: point.enabled,
  }
  pointDialogVisible.value = true
}

// 保存点位
const handleSavePoint = async () => {
  await pointFormRef.value?.validate(async (valid) => {
    if (!valid || !currentChannel.value) return

    try {
      if (isEditingPoint.value) {
        // TODO: 需要获取当前编辑的点位 ID
        const point = currentChannel.value.points?.find(
          (p) => p.name === pointForm.value.name
        )
        if (point) {
          await channelApi.updatePoint(
            currentChannel.value.id,
            point.id,
            pointForm.value
          )
          ElMessage.success('点位更新成功')
        }
      } else {
        await channelApi.createPoint(currentChannel.value.id, pointForm.value)
        ElMessage.success('点位创建成功')
      }
      pointDialogVisible.value = false
      // 刷新点位列表
      const updatedChannel = await channelApi.getChannel(currentChannel.value.id)
      currentChannel.value = updatedChannel
    } catch (error: any) {
      ElMessage.error(`操作失败: ${error.message}`)
    }
  })
}

// 删除点位
const handleDeletePoint = async (point: Point) => {
  if (!currentChannel.value) return

  try {
    await ElMessageBox.confirm(`确定要删除点位 "${point.name}" 吗？`, '警告', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    await channelApi.deletePoint(currentChannel.value.id, point.id)
    ElMessage.success('点位删除成功')
    // 刷新点位列表
    const updatedChannel = await channelApi.getChannel(currentChannel.value.id)
    currentChannel.value = updatedChannel
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`删除失败: ${error.message}`)
    }
  }
}

// 辅助函数
const getStatusType = (status: string) => {
  switch (status) {
    case 'running':
      return 'success'
    case 'stopped':
      return 'info'
    case 'error':
      return 'danger'
    default:
      return 'info'
  }
}

const getStatusLabel = (status: string) => {
  switch (status) {
    case 'running':
      return '运行中'
    case 'stopped':
      return '已停止'
    case 'error':
      return '错误'
    default:
      return status
  }
}
</script>

<style scoped>
.channels-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header span {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 18px;
  font-weight: bold;
}

.stats-row {
  margin-bottom: 20px;
}
</style>
