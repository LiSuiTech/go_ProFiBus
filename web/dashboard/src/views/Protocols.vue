<template>
  <div class="protocols-container">
    <el-page-header @back="goBack" content="协议管理">
      <template #extra>
        <el-button type="primary" @click="showCreateDialog">
          <el-icon><Plus /></el-icon>
          创建协议实例
        </el-button>
      </template>
    </el-page-header>

    <!-- 协议类型卡片 -->
    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>支持的协议类型</span>
            </div>
          </template>

          <el-tabs v-model="activeCategory" @tab-change="handleCategoryChange">
            <el-tab-pane label="现场总线 (Fieldbus)" name="fieldbus">
              <el-row :gutter="16">
                <el-col
                  v-for="protocol in filteredProtocols('fieldbus')"
                  :key="protocol.type"
                  :span="6"
                >
                  <el-card class="protocol-card" shadow="hover">
                    <div class="protocol-icon">
                      <el-icon :size="32"><Connection /></el-icon>
                    </div>
                    <h3>{{ protocol.name }}</h3>
                    <p class="description">{{ protocol.description }}</p>
                    <el-tag
                      v-for="feature in protocol.features"
                      :key="feature"
                      size="small"
                      style="margin-right: 5px; margin-bottom: 5px"
                    >
                      {{ feature }}
                    </el-tag>
                  </el-card>
                </el-col>
              </el-row>
            </el-tab-pane>

            <el-tab-pane label="工业以太网 (Industrial Ethernet)" name="industrial_ethernet">
              <el-row :gutter="16">
                <el-col
                  v-for="protocol in filteredProtocols('industrial_ethernet')"
                  :key="protocol.type"
                  :span="6"
                >
                  <el-card class="protocol-card" shadow="hover">
                    <div class="protocol-icon">
                      <el-icon :size="32"><Share /></el-icon>
                    </div>
                    <h3>{{ protocol.name }}</h3>
                    <p class="description">{{ protocol.description }}</p>
                    <el-tag
                      v-for="feature in protocol.features"
                      :key="feature"
                      size="small"
                      style="margin-right: 5px; margin-bottom: 5px"
                    >
                      {{ feature }}
                    </el-tag>
                  </el-card>
                </el-col>
              </el-row>
            </el-tab-pane>

            <el-tab-pane label="IT层协议 (IT Layer)" name="it_layer">
              <el-row :gutter="16">
                <el-col
                  v-for="protocol in filteredProtocols('it_layer')"
                  :key="protocol.type"
                  :span="6"
                >
                    <el-card class="protocol-card" shadow="hover">
                    <div class="protocol-icon">
                      <el-icon :size="32"><Share /></el-icon>
                    </div>
                    <h3>{{ protocol.name }}</h3>
                    <p class="description">{{ protocol.description }}</p>
                    <el-tag
                      v-for="feature in protocol.features"
                      :key="feature"
                      size="small"
                      style="margin-right: 5px; margin-bottom: 5px"
                    >
                      {{ feature }}
                    </el-tag>
                  </el-card>
                </el-col>
              </el-row>
            </el-tab-pane>
          </el-tabs>
        </el-card>
      </el-col>
    </el-row>

    <!-- 协议实例列表 -->
    <el-row style="margin-top: 20px">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>协议实例</span>
              <el-button text @click="loadInstances">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
            </div>
          </template>

          <el-table :data="instances" style="width: 100%" v-loading="loading">
            <el-table-column prop="id" label="实例ID" width="200" />
            <el-table-column prop="type" label="协议类型" width="150" />
            <el-table-column label="连接状态" width="120">
              <template #default="scope">
                <el-tag :type="scope.row.connected ? 'success' : 'info'">
                  {{ scope.row.connected ? '已连接' : '未连接' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="运行状态" width="120">
              <template #default="scope">
                <el-tag :type="scope.row.running ? 'success' : 'info'">
                  {{ scope.row.running ? '运行中' : '已停止' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="180">
              <template #default="scope">
                {{ formatDate(scope.row.created_at) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" fixed="right" width="350">
              <template #default="scope">
                <el-button-group>
                  <el-button
                    v-if="!scope.row.connected"
                    type="primary"
                    size="small"
                    @click="connectInstance(scope.row.id)"
                  >
                    连接
                  </el-button>
                  <el-button
                    v-else
                    type="warning"
                    size="small"
                    @click="disconnectInstance(scope.row.id)"
                  >
                    断开
                  </el-button>

                  <el-button
                    v-if="scope.row.connected && !scope.row.running"
                    type="success"
                    size="small"
                    @click="startMonitoring(scope.row.id)"
                  >
                    启动监听
                  </el-button>
                  <el-button
                    v-else-if="scope.row.running"
                    type="info"
                    size="small"
                    @click="stopMonitoring(scope.row.id)"
                  >
                    停止监听
                  </el-button>

                  <el-button
                    size="small"
                    @click="viewStatus(scope.row.id)"
                  >
                    状态
                  </el-button>

                  <el-button
                    type="danger"
                    size="small"
                    @click="deleteInstance(scope.row.id)"
                  >
                    删除
                  </el-button>
                </el-button-group>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <!-- 创建实例对话框 -->
    <el-dialog
      v-model="createDialogVisible"
      title="创建协议实例"
      width="600px"
    >
      <el-form :model="createForm" label-width="120px">
        <el-form-item label="实例ID">
          <el-input v-model="createForm.id" placeholder="例如: modbus-001" />
        </el-form-item>

        <el-form-item label="协议类型">
          <el-select v-model="createForm.type" placeholder="选择协议类型" @change="handleTypeChange">
            <el-option-group
              v-for="category in ['fieldbus', 'industrial_ethernet', 'it_layer']"
              :key="category"
              :label="getCategoryName(category)"
            >
              <el-option
                v-for="protocol in filteredProtocols(category)"
                :key="protocol.type"
                :label="protocol.name"
                :value="protocol.type"
              />
            </el-option-group>
          </el-select>
        </el-form-item>

        <el-form-item label="配置">
          <el-input
            v-model="createForm.config"
            type="textarea"
            :rows="12"
            placeholder="输入JSON格式的配置"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="createInstance">创建</el-button>
      </template>
    </el-dialog>

    <!-- 状态详情对话框 -->
    <el-dialog
      v-model="statusDialogVisible"
      title="协议状态详情"
      width="500px"
    >
      <el-descriptions :column="1" border v-if="currentStatus">
        <el-descriptions-item label="连接状态">
          <el-tag :type="currentStatus.connected ? 'success' : 'info'">
            {{ currentStatus.connected ? '已连接' : '未连接' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="运行状态">
          <el-tag :type="currentStatus.running ? 'success' : 'info'">
            {{ currentStatus.running ? '运行中' : '已停止' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="请求次数">
          {{ currentStatus.status?.request_count || 0 }}
        </el-descriptions-item>
        <el-descriptions-item label="错误次数">
          {{ currentStatus.status?.error_count || 0 }}
        </el-descriptions-item>
        <el-descriptions-item label="发送字节">
          {{ formatBytes(currentStatus.status?.bytes_sent || 0) }}
        </el-descriptions-item>
        <el-descriptions-item label="接收字节">
          {{ formatBytes(currentStatus.status?.bytes_received || 0) }}
        </el-descriptions-item>
        <el-descriptions-item label="当前延迟">
          {{ formatLatency(currentStatus.status?.current_latency || 0) }}
        </el-descriptions-item>
      </el-descriptions>

      <template #footer>
        <el-button @click="statusDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Connection, Share } from '@element-plus/icons-vue'
import axios from 'axios'

// 状态
const protocols = ref<any[]>([])
const instances = ref<any[]>([])
const loading = ref(false)
const activeCategory = ref('fieldbus')
const createDialogVisible = ref(false)
const statusDialogVisible = ref(false)
const currentStatus = ref<any>(null)

// 表单
const createForm = ref({
  id: '',
  type: '',
  config: '{}'
})

// API基础URL
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'

// 加载协议类型
const loadProtocols = async () => {
  try {
    const response = await axios.get(`${API_BASE_URL}/protocols/types`)
    if (response.data.success) {
      protocols.value = response.data.protocols
    }
  } catch (error) {
    console.error('Failed to load protocols:', error)
    ElMessage.error('加载协议类型失败')
  }
}

// 加载协议实例
const loadInstances = async () => {
  loading.value = true
  try {
    const response = await axios.get(`${API_BASE_URL}/protocols/instances`)
    if (response.data.success) {
      instances.value = response.data.instances
    }
  } catch (error) {
    console.error('Failed to load instances:', error)
    ElMessage.error('加载协议实例失败')
  } finally {
    loading.value = false
  }
}

// 过滤协议
const filteredProtocols = (category: string) => {
  return protocols.value.filter(p => p.category === category)
}

// 获取分类名称
const getCategoryName = (category: string) => {
  const names: Record<string, string> = {
    fieldbus: '现场总线',
    industrial_ethernet: '工业以太网',
    it_layer: 'IT层协议'
  }
  return names[category] || category
}

// 显示创建对话框
const showCreateDialog = () => {
  createForm.value = {
    id: '',
    type: '',
    config: '{}'
  }
  createDialogVisible.value = true
}

// 处理类型变化
const handleTypeChange = (type: string) => {
  // 根据协议类型提供默认配置模板
  const templates: Record<string, any> = {
    modbus: {
      mode: 'rtu',
      serial_port: '/dev/ttyS0',
      baud_rate: 9600,
      data_bits: 8,
      stop_bits: 1,
      parity: 'none'
    },
    profibus: {
      serial_port: '/dev/ttyS0',
      baud_rate: 9600,
      mode: 'master',
      master_address: 0
    },
    opcua: {
      endpoint_url: 'opc.tcp://localhost:4840',
      security_policy: 'None',
      security_mode: 'None'
    },
    mqtt: {
      broker_url: 'tcp://localhost:1883',
      client_id: 'go_profibus_client',
      qos: 1
    },
    restapi: {
      base_url: 'https://api.example.com',
      auth_type: 'none',
      timeout: 30000000000
    },
    database: {
      driver: 'postgres',
      host: 'localhost',
      port: 5432,
      database: 'industrial_db',
      username: 'user',
      password: 'password'
    }
  }

  if (templates[type]) {
    createForm.value.config = JSON.stringify(templates[type], null, 2)
  }
}

// 创建实例
const createInstance = async () => {
  try {
    const config = JSON.parse(createForm.value.config)
    const response = await axios.post(`${API_BASE_URL}/protocols/instances`, {
      id: createForm.value.id,
      type: createForm.value.type,
      config: config
    })

    if (response.data.success) {
      ElMessage.success('协议实例创建成功')
      createDialogVisible.value = false
      loadInstances()
    }
  } catch (error: any) {
    console.error('Failed to create instance:', error)
    ElMessage.error(error.response?.data?.error || '创建失败')
  }
}

// 连接实例
const connectInstance = async (id: string) => {
  try {
    const response = await axios.post(`${API_BASE_URL}/protocols/instances/${id}/connect`)
    if (response.data.success) {
      ElMessage.success('连接成功')
      loadInstances()
    }
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '连接失败')
  }
}

// 断开实例
const disconnectInstance = async (id: string) => {
  try {
    const response = await axios.post(`${API_BASE_URL}/protocols/instances/${id}/disconnect`)
    if (response.data.success) {
      ElMessage.success('断开连接成功')
      loadInstances()
    }
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '断开失败')
  }
}

// 启动监听
const startMonitoring = async (id: string) => {
  try {
    const response = await axios.post(`${API_BASE_URL}/protocols/instances/${id}/monitor/start`)
    if (response.data.success) {
      ElMessage.success('启动监听成功')
      loadInstances()
    }
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '启动失败')
  }
}

// 停止监听
const stopMonitoring = async (id: string) => {
  try {
    const response = await axios.post(`${API_BASE_URL}/protocols/instances/${id}/monitor/stop`)
    if (response.data.success) {
      ElMessage.success('停止监听成功')
      loadInstances()
    }
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '停止失败')
  }
}

// 查看状态
const viewStatus = async (id: string) => {
  try {
    const response = await axios.get(`${API_BASE_URL}/protocols/instances/${id}/status`)
    if (response.data.success) {
      currentStatus.value = response.data
      statusDialogVisible.value = true
    }
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '获取状态失败')
  }
}

// 删除实例
const deleteInstance = async (id: string) => {
  try {
    await ElMessageBox.confirm('确定要删除此协议实例吗？', '警告', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })

    const response = await axios.delete(`${API_BASE_URL}/protocols/instances/${id}`)
    if (response.data.success) {
      ElMessage.success('删除成功')
      loadInstances()
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || '删除失败')
    }
  }
}

// 格式化日期
const formatDate = (dateStr: string) => {
  return new Date(dateStr).toLocaleString('zh-CN')
}

// 格式化字节
const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

// 格式化延迟
const formatLatency = (ns: number) => {
  if (ns === 0) return '0 ms'
  const ms = ns / 1000000
  return ms.toFixed(2) + ' ms'
}

// 返回
const goBack = () => {
  window.history.back()
}

// 处理分类变化
const handleCategoryChange = (category: string) => {
  activeCategory.value = category
}

// 初始化
onMounted(() => {
  loadProtocols()
  loadInstances()
})
</script>

<style scoped>
.protocols-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.protocol-card {
  margin-bottom: 16px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s;
}

.protocol-card:hover {
  transform: translateY(-5px);
}

.protocol-icon {
  color: #409eff;
  margin-bottom: 10px;
}

.protocol-card h3 {
  margin: 10px 0;
  font-size: 16px;
}

.protocol-card .description {
  font-size: 12px;
  color: #909399;
  margin-bottom: 10px;
}
</style>
