<template>
  <div class="alerts-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>告警中心</span>
          <div>
            <el-button @click="showRules = !showRules">
              {{ showRules ? '隐藏规则' : '显示规则' }}
            </el-button>
            <el-button @click="showRuleTemplates = !showRuleTemplates">
              {{ showRuleTemplates ? '隐藏模板' : '规则模板' }}
            </el-button>
            <el-button type="primary" @click="handleCreateRule">新增规则</el-button>
          </div>
        </div>
      </template>

      <!-- 规则模板列表（可折叠） -->
      <el-collapse v-if="showRuleTemplates" v-model="activeTemplateCollapse" class="templates-section" style="margin-bottom: 20px">
        <el-collapse-item title="规则模板库" name="templates">
          <div style="margin-bottom: 15px">
            <el-select
              v-model="templateCategory"
              placeholder="选择分类"
              clearable
              style="width: 200px; margin-right: 10px"
              @change="loadTemplates"
            >
              <el-option label="全部" value="" />
              <el-option label="阈值" value="threshold" />
              <el-option label="异常检测" value="anomaly" />
              <el-option label="趋势分析" value="trend" />
              <el-option label="复合规则" value="composite" />
              <el-option label="变化率" value="rate" />
            </el-select>
            <el-button @click="loadTemplates">刷新</el-button>
          </div>
          <el-table :data="ruleTemplates" v-loading="templatesLoading" stripe max-height="400">
            <el-table-column prop="name" label="模板名称" width="200" />
            <el-table-column prop="category" label="分类" width="120">
              <template #default="{ row }">
                <el-tag size="small">{{ row.category }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="description" label="描述" show-overflow-tooltip />
            <el-table-column prop="usage_count" label="使用次数" width="100" />
            <el-table-column label="操作" width="200" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="handleUseTemplate(row)">使用</el-button>
                <el-button link type="success" size="small" @click="handleTestTemplate(row)">测试</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-collapse-item>
      </el-collapse>

      <!-- 告警规则列表（可折叠） -->
      <el-collapse v-if="showRules" v-model="activeRuleCollapse" class="rules-section">
        <el-collapse-item title="告警规则管理" name="rules">
          <el-table :data="alertRules" v-loading="rulesLoading" stripe>
            <el-table-column prop="name" label="规则名称" width="200" />
            <el-table-column prop="level" label="级别" width="100">
              <template #default="{ row }">
                <el-tag :type="getLevelTagType(row.level)">{{ getLevelLabel(row.level) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="enabled" label="启用" width="80">
              <template #default="{ row }">
                <el-switch v-model="row.enabled" @change="handleToggleRule(row)" />
              </template>
            </el-table-column>
            <el-table-column prop="description" label="描述" show-overflow-tooltip />
            <el-table-column label="操作" width="150" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="handleEditRule(row)">编辑</el-button>
                <el-button link type="danger" @click="handleDeleteRule(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-collapse-item>
      </el-collapse>

      <!-- 筛选条件 -->
      <el-form :inline="true" :model="filters" class="filter-form">
        <el-form-item label="告警级别">
          <el-select v-model="filters.level" placeholder="全部" clearable style="width: 150px">
            <el-option label="信息" value="info" />
            <el-option label="警告" value="warning" />
            <el-option label="错误" value="error" />
            <el-option label="严重" value="critical" />
          </el-select>
        </el-form-item>
        <el-form-item label="告警状态">
          <el-select v-model="filters.status" placeholder="全部" clearable style="width: 150px">
            <el-option label="活动" value="active" />
            <el-option label="已确认" value="acknowledged" />
            <el-option label="已解决" value="resolved" />
            <el-option label="已抑制" value="suppressed" />
          </el-select>
        </el-form-item>
        <el-form-item label="设备ID">
          <el-input v-model="filters.device_id" placeholder="设备ID" clearable style="width: 150px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadAlerts">查询</el-button>
          <el-button @click="resetFilters">重置</el-button>
        </el-form-item>
      </el-form>

      <!-- 告警统计 -->
      <el-row :gutter="20" class="stats-row">
        <el-col :span="6">
          <el-statistic title="总告警数" :value="stats.total_alerts" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="活动告警" :value="stats.active_alerts">
            <template #suffix>
              <el-tag type="danger" size="small">需处理</el-tag>
            </template>
          </el-statistic>
        </el-col>
        <el-col :span="6">
          <el-statistic title="已确认" :value="stats.acknowledged_alerts" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="已解决" :value="stats.resolved_alerts" />
        </el-col>
      </el-row>

      <!-- 告警列表 -->
      <el-table :data="alerts" v-loading="loading" stripe>
        <el-table-column prop="message" label="告警消息" min-width="200" show-overflow-tooltip />
        <el-table-column prop="level" label="级别" width="100">
          <template #default="{ row }">
            <el-tag :type="getLevelTagType(row.level)">{{ getLevelLabel(row.level) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.status)">{{ getStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="device_id" label="设备ID" width="150" />
        <el-table-column prop="count" label="次数" width="80" />
        <el-table-column prop="first_occurred_at" label="首次发生" width="180">
          <template #default="{ row }">
            {{ formatTime(row.first_occurred_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="last_occurred_at" label="最后发生" width="180">
          <template #default="{ row }">
            {{ formatTime(row.last_occurred_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'active'"
              link
              type="primary"
              @click="handleAcknowledge(row)"
            >
              确认
            </el-button>
            <el-button
              v-if="row.status !== 'resolved'"
              link
              type="success"
              @click="handleResolve(row)"
            >
              解决
            </el-button>
            <el-button link type="primary" @click="handleView(row)">详情</el-button>
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
          @size-change="loadAlerts"
          @current-change="loadAlerts"
        />
      </div>
    </el-card>

    <!-- 创建/编辑告警规则对话框 -->
    <el-dialog v-model="ruleDialogVisible" :title="ruleDialogTitle" width="700px">
      <el-form :model="ruleForm" :rules="ruleRules" ref="ruleFormRef" label-width="120px">
        <el-form-item label="规则名称" prop="name">
          <el-input v-model="ruleForm.name" placeholder="请输入规则名称" />
        </el-form-item>
        <el-form-item label="告警级别" prop="level">
          <el-select v-model="ruleForm.level" placeholder="请选择告警级别" style="width: 100%">
            <el-option label="信息" value="info" />
            <el-option label="警告" value="warning" />
            <el-option label="错误" value="error" />
            <el-option label="严重" value="critical" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="ruleForm.description" type="textarea" :rows="3" placeholder="请输入规则描述" />
        </el-form-item>
        <el-form-item label="触发条件" prop="condition">
          <el-input
            v-model="conditionJson"
            type="textarea"
            :rows="5"
            placeholder='请输入JSON格式的条件，例如: {"field": "temperature", "operator": ">", "threshold": 80}'
          />
        </el-form-item>
        <el-form-item label="冷却时间（秒）">
          <el-input-number v-model="ruleForm.cooldown_seconds" :min="0" style="width: 100%" />
        </el-form-item>
        <el-form-item label="最大执行次数">
          <el-input-number v-model="ruleForm.max_executions" :min="0" style="width: 100%" />
          <span style="margin-left: 10px; color: #909399">0表示无限制</span>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="ruleForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmitRule">确定</el-button>
      </template>
    </el-dialog>

    <!-- 规则模板使用对话框 -->
    <el-dialog v-model="templateDialogVisible" :title="`从模板创建规则: ${selectedTemplate?.name}`" width="700px">
      <div v-if="selectedTemplate">
        <el-alert type="info" :closable="false" style="margin-bottom: 20px">
          <template #title>
            <div>
              <div><strong>模板描述:</strong> {{ selectedTemplate.description || '无描述' }}</div>
              <div><strong>分类:</strong> {{ selectedTemplate.category }}</div>
            </div>
          </template>
        </el-alert>

        <el-form :model="templateRuleForm" label-width="120px">
          <el-form-item label="规则名称" required>
            <el-input v-model="templateRuleForm.name" placeholder="请输入规则名称" />
          </el-form-item>
          <el-form-item label="描述">
            <el-input v-model="templateRuleForm.description" type="textarea" :rows="2" placeholder="请输入描述（可选）" />
          </el-form-item>
          <el-form-item label="告警级别" required>
            <el-select v-model="templateRuleForm.level" style="width: 100%">
              <el-option label="信息" value="info" />
              <el-option label="警告" value="warning" />
              <el-option label="错误" value="error" />
              <el-option label="严重" value="critical" />
            </el-select>
          </el-form-item>

          <el-divider>模板变量配置</el-divider>
          <div v-if="Object.keys(selectedTemplate.variables_config || {}).length === 0">
            <el-empty description="此模板无需配置变量" :image-size="60" />
          </div>
          <el-form-item
            v-for="[key, config] in Object.entries(selectedTemplate.variables_config || {})"
            :key="key"
            :label="(config as any).description || key"
            :required="(config as any).required"
          >
            <el-input
              v-if="(config as any).type === 'string' && !(config as any).enum"
              v-model="templateVariables[key]"
              :placeholder="`请输入${(config as any).description || key}`"
            />
            <el-select
              v-else-if="(config as any).enum"
              v-model="templateVariables[key]"
              :placeholder="`请选择${(config as any).description || key}`"
              style="width: 100%"
            >
              <el-option
                v-for="opt in (config as any).enum"
                :key="opt"
                :label="opt"
                :value="opt"
              />
            </el-select>
            <el-input-number
              v-else-if="(config as any).type === 'number'"
              v-model="templateVariables[key]"
              style="width: 100%"
            />
            <el-switch
              v-else-if="(config as any).type === 'boolean'"
              v-model="templateVariables[key]"
            />
            <div v-if="(config as any).description" style="font-size: 12px; color: #909399; margin-top: 5px">
              {{ (config as any).description }}
            </div>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="templateDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreateRuleFromTemplate">创建规则</el-button>
      </template>
    </el-dialog>

    <!-- 规则测试对话框 -->
    <el-dialog v-model="testDialogVisible" :title="`测试规则: ${testTemplate?.name || '自定义规则'}`" width="800px">
      <el-tabs v-model="testTab">
        <el-tab-pane label="测试数据" name="data">
          <el-form label-width="120px">
            <el-form-item label="测试数据 (JSON)">
              <el-input
                v-model="testDataText"
                type="textarea"
                :rows="10"
                placeholder='{"field_name": 100, "mean": 50, "stddev": 10}'
              />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleRunTest">运行测试</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="测试结果" name="result">
          <div v-if="testResult">
            <el-alert
              :type="testResult.triggered ? 'warning' : 'success'"
              :closable="false"
              style="margin-bottom: 20px"
            >
              <template #title>
                <div>
                  <div><strong>触发状态:</strong> {{ testResult.triggered ? '已触发' : '未触发' }}</div>
                  <div><strong>执行时间:</strong> {{ testResult.execution_time_ms }}ms</div>
                </div>
              </template>
            </el-alert>
            <el-descriptions title="测试详情" border>
              <el-descriptions-item
                v-for="[key, value] in Object.entries(testResult.test_result || {})"
                :key="key"
                :label="key"
              >
                {{ typeof value === 'object' ? JSON.stringify(value) : value }}
              </el-descriptions-item>
            </el-descriptions>
          </div>
          <el-empty v-else description="请先运行测试" :image-size="60" />
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <el-button @click="testDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { alertApi, type Alert, type AlertRule, type AlertFilters, type AlertStats } from '@/services/alertApi'
import { ruleTemplateApi, type RuleTemplate, type RuleTestResult } from '@/services/ruleTemplateApi'

const loading = ref(false)
const rulesLoading = ref(false)
const alerts = ref<Alert[]>([])
const alertRules = ref<AlertRule[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)
const showRules = ref(false)
const activeRuleCollapse = ref(['rules'])

// 规则模板相关
const showRuleTemplates = ref(false)
const activeTemplateCollapse = ref(['templates'])
const ruleTemplates = ref<RuleTemplate[]>([])
const templatesLoading = ref(false)
const templateCategory = ref('')
const selectedTemplate = ref<RuleTemplate | null>(null)
const templateDialogVisible = ref(false)
const templateVariables = ref<Record<string, any>>({})
const templateRuleForm = reactive({
  name: '',
  description: '',
  level: 'warning' as 'info' | 'warning' | 'error' | 'critical',
})

// 规则测试相关
const testDialogVisible = ref(false)
const testTemplate = ref<RuleTemplate | null>(null)
const testTab = ref('data')
const testDataText = ref('{}')
const testResult = ref<RuleTestResult | null>(null)

const stats = reactive<AlertStats>({
  total_alerts: 0,
  active_alerts: 0,
  acknowledged_alerts: 0,
  resolved_alerts: 0,
  alerts_by_level: {},
  alerts_by_status: {},
})

const filters = reactive<AlertFilters>({
  level: undefined,
  status: undefined,
  device_id: undefined,
})

const ruleDialogVisible = ref(false)
const ruleDialogTitle = ref('新增告警规则')
const ruleFormRef = ref()
const ruleForm = reactive({
  name: '',
  level: '',
  description: '',
  condition: {} as Record<string, any>,
  cooldown_seconds: 300,
  max_executions: 0,
  enabled: true,
})
const conditionJson = ref('{}')
const editingRuleId = ref<string | null>(null)

const ruleRules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  level: [{ required: true, message: '请选择告警级别', trigger: 'change' }],
  condition: [{ required: true, message: '请输入触发条件', trigger: 'blur' }],
}

const loadAlerts = async () => {
  loading.value = true
  try {
    const result = await alertApi.getAlerts({
      ...filters,
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value,
    })
    alerts.value = result.alerts
    total.value = result.count
  } catch (error: any) {
    ElMessage.error('加载告警列表失败: ' + (error.response?.data?.error || error.message))
  } finally {
    loading.value = false
  }
}

const loadAlertRules = async () => {
  rulesLoading.value = true
  try {
    const result = await alertApi.getAlertRules()
    alertRules.value = result.rules
  } catch (error: any) {
    ElMessage.error('加载告警规则失败: ' + (error.response?.data?.error || error.message))
  } finally {
    rulesLoading.value = false
  }
}

const loadStats = async () => {
  try {
    const statsData = await alertApi.getAlertStats()
    Object.assign(stats, statsData)
  } catch (error: any) {
    console.error('加载告警统计失败:', error)
  }
}

const resetFilters = () => {
  filters.level = undefined
  filters.status = undefined
  filters.device_id = undefined
  loadAlerts()
}

const handleCreateRule = () => {
  ruleDialogTitle.value = '新增告警规则'
  editingRuleId.value = null
  ruleForm.name = ''
  ruleForm.level = ''
  ruleForm.description = ''
  ruleForm.condition = {}
  conditionJson.value = '{}'
  ruleForm.cooldown_seconds = 300
  ruleForm.max_executions = 0
  ruleForm.enabled = true
  ruleDialogVisible.value = true
}

const handleEditRule = async (rule: AlertRule) => {
  ruleDialogTitle.value = '编辑告警规则'
  editingRuleId.value = rule.id
  ruleForm.name = rule.name
  ruleForm.level = rule.level
  ruleForm.description = rule.description || ''
  ruleForm.condition = rule.condition
  conditionJson.value = JSON.stringify(rule.condition, null, 2)
  ruleForm.cooldown_seconds = rule.cooldown_seconds
  ruleForm.max_executions = rule.max_executions
  ruleForm.enabled = rule.enabled
  ruleDialogVisible.value = true
}

const handleToggleRule = async (rule: AlertRule) => {
  try {
    await alertApi.updateAlertRule(rule.id, { enabled: rule.enabled })
    ElMessage.success('更新成功')
  } catch (error: any) {
    ElMessage.error('更新失败: ' + (error.response?.data?.error || error.message))
    rule.enabled = !rule.enabled // 回滚
  }
}

const handleDeleteRule = async (rule: AlertRule) => {
  try {
    await ElMessageBox.confirm(`确定要删除告警规则 "${rule.name}" 吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await alertApi.deleteAlertRule(rule.id)
    ElMessage.success('删除成功')
    loadAlertRules()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + (error.response?.data?.error || error.message))
    }
  }
}

const handleSubmitRule = async () => {
  if (!ruleFormRef.value) return
  await ruleFormRef.value.validate(async (valid: boolean) => {
    if (!valid) return

    try {
      // 解析条件JSON
      let condition: Record<string, any>
      try {
        condition = JSON.parse(conditionJson.value)
      } catch (e) {
        ElMessage.error('条件JSON格式错误')
        return
      }

      if (editingRuleId.value) {
        await alertApi.updateAlertRule(editingRuleId.value, {
          ...ruleForm,
          condition,
        })
        ElMessage.success('更新成功')
      } else {
        await alertApi.createAlertRule({
          ...ruleForm,
          condition,
        })
        ElMessage.success('创建成功')
      }
      ruleDialogVisible.value = false
      loadAlertRules()
    } catch (error: any) {
      ElMessage.error((editingRuleId.value ? '更新' : '创建') + '失败: ' + (error.response?.data?.error || error.message))
    }
  })
}

const handleAcknowledge = async (alert: Alert) => {
  try {
    await alertApi.acknowledgeAlert(alert.id, 'current_user') // TODO: 使用实际用户
    ElMessage.success('告警已确认')
    loadAlerts()
    loadStats()
  } catch (error: any) {
    ElMessage.error('确认失败: ' + (error.response?.data?.error || error.message))
  }
}

const handleResolve = async (alert: Alert) => {
  try {
    await alertApi.resolveAlert(alert.id, 'current_user') // TODO: 使用实际用户
    ElMessage.success('告警已解决')
    loadAlerts()
    loadStats()
  } catch (error: any) {
    ElMessage.error('解决失败: ' + (error.response?.data?.error || error.message))
  }
}

const handleView = (alert: Alert) => {
  // TODO: 显示告警详情
  ElMessage.info('查看告警详情功能待实现')
}

const getLevelLabel = (level: string) => {
  const map: Record<string, string> = {
    info: '信息',
    warning: '警告',
    error: '错误',
    critical: '严重',
  }
  return map[level] || level
}

const getLevelTagType = (level: string) => {
  const map: Record<string, string> = {
    info: 'info',
    warning: 'warning',
    error: 'danger',
    critical: 'danger',
  }
  return map[level] || ''
}

const getStatusLabel = (status: string) => {
  const map: Record<string, string> = {
    active: '活动',
    acknowledged: '已确认',
    resolved: '已解决',
    suppressed: '已抑制',
  }
  return map[status] || status
}

const getStatusTagType = (status: string) => {
  const map: Record<string, string> = {
    active: 'danger',
    acknowledged: 'warning',
    resolved: 'success',
    suppressed: 'info',
  }
  return map[status] || ''
}

const formatTime = (timeStr: string) => {
  if (!timeStr) return '-'
  return new Date(timeStr).toLocaleString('zh-CN')
}

onMounted(() => {
  loadAlerts()
  loadAlertRules()
  loadStats()
  loadTemplates()
  // 定时刷新统计
  setInterval(() => {
    loadStats()
  }, 30000) // 每30秒刷新一次
})
</script>

<style scoped>
.alerts-page {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.rules-section {
  margin-bottom: 20px;
}

.filter-form {
  margin-bottom: 20px;
}

.stats-row {
  margin-bottom: 20px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
