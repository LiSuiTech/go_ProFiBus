<template>
  <div class="control-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>设备控制</span>
        </div>
      </template>

      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <!-- 控制策略标签页 -->
        <el-tab-pane label="控制策略" name="policies">
          <div class="tab-content">
            <div class="toolbar">
              <el-button type="primary" @click="handleCreatePolicy">新增策略</el-button>
              <el-button @click="loadPolicies">刷新</el-button>
            </div>

            <el-table :data="policies" v-loading="policiesLoading" stripe>
              <el-table-column prop="name" label="策略名称" width="200" />
              <el-table-column prop="enabled" label="启用" width="80">
                <template #default="{ row }">
                  <el-switch v-model="row.enabled" @change="handleTogglePolicy(row)" />
                </template>
              </el-table-column>
              <el-table-column prop="priority" label="优先级" width="100" />
              <el-table-column prop="execution_count" label="执行次数" width="120" />
              <el-table-column prop="cooldown_seconds" label="冷却时间(秒)" width="120" />
              <el-table-column prop="description" label="描述" show-overflow-tooltip />
              <el-table-column label="操作" width="200" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" @click="handleViewPolicy(row)">查看</el-button>
                  <el-button link type="primary" @click="handleEditPolicy(row)">编辑</el-button>
                  <el-button link type="danger" @click="handleDeletePolicy(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- 控制动作标签页 -->
        <el-tab-pane label="控制动作" name="actions">
          <div class="tab-content">
            <div class="toolbar">
              <el-button type="primary" @click="handleCreateAction">执行动作</el-button>
              <el-button @click="loadActions">刷新</el-button>
            </div>

            <!-- 筛选条件 -->
            <el-form :inline="true" :model="actionFilters" class="filter-form">
              <el-form-item label="设备ID">
                <el-input v-model="actionFilters.device_id" placeholder="设备ID" clearable style="width: 150px" />
              </el-form-item>
              <el-form-item label="动作类型">
                <el-select v-model="actionFilters.action_type" placeholder="全部" clearable style="width: 150px">
                  <el-option label="紧急停止" value="emergency_stop" />
                  <el-option label="关机" value="shutdown" />
                  <el-option label="启动" value="start" />
                  <el-option label="暂停" value="pause" />
                  <el-option label="恢复" value="resume" />
                  <el-option label="设置值" value="set_value" />
                  <el-option label="调用方法" value="call_method" />
                  <el-option label="发送命令" value="send_command" />
                  <el-option label="自定义" value="custom" />
                </el-select>
              </el-form-item>
              <el-form-item label="状态">
                <el-select v-model="actionFilters.status" placeholder="全部" clearable style="width: 150px">
                  <el-option label="待执行" value="pending" />
                  <el-option label="执行中" value="executing" />
                  <el-option label="已完成" value="completed" />
                  <el-option label="失败" value="failed" />
                  <el-option label="已取消" value="cancelled" />
                </el-select>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="loadActions">查询</el-button>
                <el-button @click="resetActionFilters">重置</el-button>
              </el-form-item>
            </el-form>

            <el-table :data="actions" v-loading="actionsLoading" stripe>
              <el-table-column prop="device_id" label="设备ID" width="150" />
              <el-table-column prop="action_type" label="动作类型" width="120">
                <template #default="{ row }">
                  <el-tag>{{ getActionTypeLabel(row.action_type) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="status" label="状态" width="100">
                <template #default="{ row }">
                  <el-tag :type="getActionStatusTagType(row.status)">{{ getActionStatusLabel(row.status) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="severity" label="严重程度" width="100" />
              <el-table-column prop="reason" label="原因" show-overflow-tooltip />
              <el-table-column prop="created_at" label="创建时间" width="180">
                <template #default="{ row }">
                  {{ formatDateTime(row.created_at) }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="150" fixed="right">
                <template #default="{ row }">
                  <el-button 
                    v-if="row.status === 'pending' && row.require_confirmation" 
                    link 
                    type="primary" 
                    @click="handleConfirmAction(row)"
                  >
                    确认
                  </el-button>
                  <el-button link type="primary" @click="handleViewAction(row)">查看</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- 审计日志标签页 -->
        <el-tab-pane label="审计日志" name="audit-logs">
          <div class="tab-content">
            <div class="toolbar">
              <el-button @click="loadAuditLogs">刷新</el-button>
            </div>

            <!-- 筛选条件 -->
            <el-form :inline="true" :model="auditFilters" class="filter-form">
              <el-form-item label="动作ID">
                <el-input v-model="auditFilters.action_id" placeholder="动作ID" clearable style="width: 150px" />
              </el-form-item>
              <el-form-item label="用户ID">
                <el-input v-model="auditFilters.user_id" placeholder="用户ID" clearable style="width: 150px" />
              </el-form-item>
              <el-form-item label="事件类型">
                <el-select v-model="auditFilters.event_type" placeholder="全部" clearable style="width: 150px">
                  <el-option label="创建" value="created" />
                  <el-option label="确认" value="confirmed" />
                  <el-option label="执行" value="executed" />
                  <el-option label="完成" value="completed" />
                  <el-option label="失败" value="failed" />
                  <el-option label="取消" value="cancelled" />
                </el-select>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="loadAuditLogs">查询</el-button>
                <el-button @click="resetAuditFilters">重置</el-button>
              </el-form-item>
            </el-form>

            <el-table :data="auditLogs" v-loading="auditLogsLoading" stripe>
              <el-table-column prop="action_id" label="动作ID" width="200" />
              <el-table-column prop="event_type" label="事件类型" width="120">
                <template #default="{ row }">
                  <el-tag>{{ getEventTypeLabel(row.event_type) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="user_name" label="用户" width="120" />
              <el-table-column prop="ip_address" label="IP地址" width="150" />
              <el-table-column prop="created_at" label="时间" width="180">
                <template #default="{ row }">
                  {{ formatDateTime(row.created_at) }}
                </template>
              </el-table-column>
              <el-table-column prop="details" label="详细信息" show-overflow-tooltip>
                <template #default="{ row }">
                  <el-button link type="primary" @click="handleViewAuditDetails(row)">查看详情</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- 权限管理标签页 -->
        <el-tab-pane label="权限管理" name="permissions">
          <div class="tab-content">
            <div class="toolbar">
              <el-button type="primary" @click="handleCreatePermission">新增权限</el-button>
              <el-button @click="loadPermissions">刷新</el-button>
            </div>

            <!-- 筛选条件 -->
            <el-form :inline="true" :model="permissionFilters" class="filter-form">
              <el-form-item label="用户ID">
                <el-input v-model="permissionFilters.user_id" placeholder="用户ID" clearable style="width: 150px" />
              </el-form-item>
              <el-form-item label="动作类型">
                <el-select v-model="permissionFilters.action_type" placeholder="全部" clearable style="width: 150px">
                  <el-option label="紧急停止" value="emergency_stop" />
                  <el-option label="关机" value="shutdown" />
                  <el-option label="启动" value="start" />
                  <el-option label="暂停" value="pause" />
                  <el-option label="恢复" value="resume" />
                  <el-option label="设置值" value="set_value" />
                  <el-option label="调用方法" value="call_method" />
                  <el-option label="发送命令" value="send_command" />
                  <el-option label="自定义" value="custom" />
                </el-select>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="loadPermissions">查询</el-button>
                <el-button @click="resetPermissionFilters">重置</el-button>
              </el-form-item>
            </el-form>

            <el-table :data="permissions" v-loading="permissionsLoading" stripe>
              <el-table-column prop="user_id" label="用户ID" width="150" />
              <el-table-column prop="action_type" label="动作类型" width="120">
                <template #default="{ row }">
                  <el-tag>{{ getActionTypeLabel(row.action_type) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="enabled" label="启用" width="80">
                <template #default="{ row }">
                  <el-switch v-model="row.enabled" @change="handleTogglePermission(row)" />
                </template>
              </el-table-column>
              <el-table-column prop="max_severity" label="最大严重程度" width="120" />
              <el-table-column prop="target_devices" label="目标设备" show-overflow-tooltip>
                <template #default="{ row }">
                  {{ row.target_devices.length === 0 ? '全部设备' : row.target_devices.join(', ') }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="150" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" @click="handleEditPermission(row)">编辑</el-button>
                  <el-button link type="danger" @click="handleDeletePermission(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 策略对话框 -->
    <el-dialog v-model="policyDialogVisible" :title="policyDialogTitle" width="700px">
      <el-form :model="policyForm" :rules="policyRules" ref="policyFormRef" label-width="120px">
        <el-form-item label="策略名称" prop="name">
          <el-input v-model="policyForm.name" placeholder="请输入策略名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="policyForm.description" type="textarea" :rows="3" placeholder="请输入描述" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="policyForm.enabled" />
        </el-form-item>
        <el-form-item label="优先级" prop="priority">
          <el-input-number v-model="policyForm.priority" :min="0" :max="100" />
        </el-form-item>
        <el-form-item label="冷却时间(秒)">
          <el-input-number v-model="policyForm.cooldown_seconds" :min="0" />
        </el-form-item>
        <el-form-item label="最大执行次数">
          <el-input-number v-model="policyForm.max_executions" :min="0" />
          <span style="margin-left: 10px; color: #909399;">0表示无限制</span>
        </el-form-item>
        <el-form-item label="条件配置">
          <el-input v-model="conditionConfigText" type="textarea" :rows="5" placeholder='JSON格式，例如: {"threshold": 100}' />
        </el-form-item>
        <el-form-item label="动作配置" prop="action_config">
          <el-input v-model="actionConfigText" type="textarea" :rows="5" placeholder='JSON格式，例如: {"action_type": "emergency_stop", "parameters": {}}' />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="policyDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSavePolicy">保存</el-button>
      </template>
    </el-dialog>

    <!-- 动作对话框 -->
    <el-dialog v-model="actionDialogVisible" :title="actionDialogTitle" width="600px">
      <el-form :model="actionForm" :rules="actionRules" ref="actionFormRef" label-width="120px">
        <el-form-item label="设备ID" prop="device_id">
          <el-input v-model="actionForm.device_id" placeholder="请输入设备ID" />
        </el-form-item>
        <el-form-item label="动作类型" prop="action_type">
          <el-select v-model="actionForm.action_type" placeholder="请选择动作类型" style="width: 100%">
            <el-option label="紧急停止" value="emergency_stop" />
            <el-option label="关机" value="shutdown" />
            <el-option label="启动" value="start" />
            <el-option label="暂停" value="pause" />
            <el-option label="恢复" value="resume" />
            <el-option label="设置值" value="set_value" />
            <el-option label="调用方法" value="call_method" />
            <el-option label="发送命令" value="send_command" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item label="原因">
          <el-input v-model="actionForm.reason" type="textarea" :rows="3" placeholder="请输入执行原因" />
        </el-form-item>
        <el-form-item label="严重程度">
          <el-input-number v-model="actionForm.severity" :min="1" :max="5" />
        </el-form-item>
        <el-form-item label="需要确认">
          <el-switch v-model="actionForm.require_confirmation" />
        </el-form-item>
        <el-form-item label="参数">
          <el-input v-model="actionParametersText" type="textarea" :rows="4" placeholder='JSON格式，例如: {"value": 100}' />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="actionDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveAction">执行</el-button>
      </template>
    </el-dialog>

    <!-- 权限对话框 -->
    <el-dialog v-model="permissionDialogVisible" :title="permissionDialogTitle" width="600px">
      <el-form :model="permissionForm" :rules="permissionRules" ref="permissionFormRef" label-width="120px">
        <el-form-item label="用户ID" prop="user_id">
          <el-input v-model="permissionForm.user_id" placeholder="请输入用户ID" />
        </el-form-item>
        <el-form-item label="动作类型" prop="action_type">
          <el-select v-model="permissionForm.action_type" placeholder="请选择动作类型" style="width: 100%">
            <el-option label="紧急停止" value="emergency_stop" />
            <el-option label="关机" value="shutdown" />
            <el-option label="启动" value="start" />
            <el-option label="暂停" value="pause" />
            <el-option label="恢复" value="resume" />
            <el-option label="设置值" value="set_value" />
            <el-option label="调用方法" value="call_method" />
            <el-option label="发送命令" value="send_command" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="permissionForm.enabled" />
        </el-form-item>
        <el-form-item label="最大严重程度">
          <el-input-number v-model="permissionForm.max_severity" :min="1" :max="5" />
        </el-form-item>
        <el-form-item label="需要确认">
          <el-switch v-model="permissionForm.require_confirmation" />
        </el-form-item>
        <el-form-item label="目标设备">
          <el-input v-model="targetDevicesText" type="textarea" :rows="3" placeholder="设备ID列表，每行一个，留空表示全部设备" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="permissionDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSavePermission">保存</el-button>
      </template>
    </el-dialog>

    <!-- 详情对话框 -->
    <el-dialog v-model="detailDialogVisible" :title="detailDialogTitle" width="800px">
      <el-descriptions :column="2" border>
        <el-descriptions-item v-for="(value, key) in detailData" :key="key" :label="key">
          {{ typeof value === 'object' ? JSON.stringify(value, null, 2) : value }}
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { controlApi, type ControlPolicy, type ControlAction, type AuditLog, type ControlPermission } from '../services/controlApi'

const activeTab = ref('policies')

// 策略相关
const policies = ref<ControlPolicy[]>([])
const policiesLoading = ref(false)
const policyDialogVisible = ref(false)
const policyDialogTitle = ref('新增策略')
const policyFormRef = ref()
const editingPolicyId = ref<string | null>(null)
const policyForm = reactive({
  name: '',
  description: '',
  enabled: true,
  priority: 0,
  cooldown_seconds: 300,
  max_executions: 0,
  condition_config: {},
  action_config: {},
})
const conditionConfigText = ref('{}')
const actionConfigText = ref('{}')
const policyRules = {
  name: [{ required: true, message: '请输入策略名称', trigger: 'blur' }],
  action_config: [{ required: true, message: '请输入动作配置', trigger: 'blur' }],
}

// 动作相关
const actions = ref<ControlAction[]>([])
const actionsLoading = ref(false)
const actionDialogVisible = ref(false)
const actionDialogTitle = ref('执行控制动作')
const actionFormRef = ref()
const actionForm = reactive({
  device_id: '',
  action_type: 'emergency_stop' as any,
  reason: '',
  severity: 1,
  require_confirmation: false,
  parameters: {},
})
const actionParametersText = ref('{}')
const actionFilters = reactive({
  device_id: '',
  action_type: '' as any,
  status: '' as any,
})
const actionRules = {
  device_id: [{ required: true, message: '请输入设备ID', trigger: 'blur' }],
  action_type: [{ required: true, message: '请选择动作类型', trigger: 'change' }],
}

// 审计日志相关
const auditLogs = ref<AuditLog[]>([])
const auditLogsLoading = ref(false)
const auditFilters = reactive({
  action_id: '',
  user_id: '',
  event_type: '' as any,
})

// 权限相关
const permissions = ref<ControlPermission[]>([])
const permissionsLoading = ref(false)
const permissionDialogVisible = ref(false)
const permissionDialogTitle = ref('新增权限')
const permissionFormRef = ref()
const editingPermissionId = ref<string | null>(null)
const permissionForm = reactive({
  user_id: '',
  action_type: 'emergency_stop' as any,
  enabled: true,
  max_severity: 3,
  require_confirmation: false,
  target_devices: [] as string[],
})
const targetDevicesText = ref('')
const permissionFilters = reactive({
  user_id: '',
  action_type: '' as any,
})
const permissionRules = {
  user_id: [{ required: true, message: '请输入用户ID', trigger: 'blur' }],
  action_type: [{ required: true, message: '请选择动作类型', trigger: 'change' }],
}

// 详情对话框
const detailDialogVisible = ref(false)
const detailDialogTitle = ref('详情')
const detailData = ref<any>({})

// 标签页切换
const handleTabChange = (tab: string) => {
  if (tab === 'policies') {
    loadPolicies()
  } else if (tab === 'actions') {
    loadActions()
  } else if (tab === 'audit-logs') {
    loadAuditLogs()
  } else if (tab === 'permissions') {
    loadPermissions()
  }
}

// 策略相关方法
const loadPolicies = async () => {
  policiesLoading.value = true
  try {
    const result = await controlApi.getPolicies({ limit: 100 })
    policies.value = result.policies
  } catch (error: any) {
    ElMessage.error('加载策略失败: ' + (error.message || '未知错误'))
  } finally {
    policiesLoading.value = false
  }
}

const handleCreatePolicy = () => {
  policyDialogTitle.value = '新增策略'
  editingPolicyId.value = null
  Object.assign(policyForm, {
    name: '',
    description: '',
    enabled: true,
    priority: 0,
    cooldown_seconds: 300,
    max_executions: 0,
    condition_config: {},
    action_config: {},
  })
  conditionConfigText.value = '{}'
  actionConfigText.value = '{}'
  policyDialogVisible.value = true
}

const handleViewPolicy = (row: ControlPolicy) => {
  detailDialogTitle.value = '策略详情'
  detailData.value = row
  detailDialogVisible.value = true
}

const handleEditPolicy = (row: ControlPolicy) => {
  policyDialogTitle.value = '编辑策略'
  editingPolicyId.value = row.id
  Object.assign(policyForm, {
    name: row.name,
    description: row.description || '',
    enabled: row.enabled,
    priority: row.priority,
    cooldown_seconds: row.cooldown_seconds,
    max_executions: row.max_executions,
    condition_config: row.condition_config,
    action_config: row.action_config,
  })
  conditionConfigText.value = JSON.stringify(row.condition_config, null, 2)
  actionConfigText.value = JSON.stringify(row.action_config, null, 2)
  policyDialogVisible.value = true
}

const handleTogglePolicy = async (row: ControlPolicy) => {
  try {
    await controlApi.updatePolicy(row.id, { enabled: row.enabled })
    ElMessage.success('更新成功')
  } catch (error: any) {
    row.enabled = !row.enabled
    ElMessage.error('更新失败: ' + (error.message || '未知错误'))
  }
}

const handleDeletePolicy = async (row: ControlPolicy) => {
  try {
    await ElMessageBox.confirm('确定要删除该策略吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await controlApi.deletePolicy(row.id)
    ElMessage.success('删除成功')
    loadPolicies()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + (error.message || '未知错误'))
    }
  }
}

const handleSavePolicy = async () => {
  try {
    await policyFormRef.value.validate()
    
    try {
      policyForm.condition_config = JSON.parse(conditionConfigText.value)
    } catch {
      ElMessage.error('条件配置JSON格式错误')
      return
    }
    
    try {
      policyForm.action_config = JSON.parse(actionConfigText.value)
    } catch {
      ElMessage.error('动作配置JSON格式错误')
      return
    }

    if (editingPolicyId.value === null) {
      await controlApi.createPolicy(policyForm)
      ElMessage.success('创建成功')
    } else {
      await controlApi.updatePolicy(editingPolicyId.value, policyForm)
      ElMessage.success('更新成功')
    }
    policyDialogVisible.value = false
    loadPolicies()
  } catch (error: any) {
    ElMessage.error('保存失败: ' + (error.message || '未知错误'))
  }
}

// 动作相关方法
const loadActions = async () => {
  actionsLoading.value = true
  try {
    const filters: any = { limit: 100 }
    if (actionFilters.device_id) filters.device_id = actionFilters.device_id
    if (actionFilters.action_type) filters.action_type = actionFilters.action_type
    if (actionFilters.status) filters.status = actionFilters.status
    
    const result = await controlApi.getActions(filters)
    actions.value = result.actions
  } catch (error: any) {
    ElMessage.error('加载动作失败: ' + (error.message || '未知错误'))
  } finally {
    actionsLoading.value = false
  }
}

const resetActionFilters = () => {
  Object.assign(actionFilters, {
    device_id: '',
    action_type: '',
    status: '',
  })
  loadActions()
}

const handleCreateAction = () => {
  actionDialogTitle.value = '执行控制动作'
  Object.assign(actionForm, {
    device_id: '',
    action_type: 'emergency_stop',
    reason: '',
    severity: 1,
    require_confirmation: false,
    parameters: {},
  })
  actionParametersText.value = '{}'
  actionDialogVisible.value = true
}

const handleViewAction = (row: ControlAction) => {
  detailDialogTitle.value = '动作详情'
  detailData.value = row
  detailDialogVisible.value = true
}

const handleConfirmAction = async (row: ControlAction) => {
  try {
    await ElMessageBox.confirm('确定要确认执行该动作吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await controlApi.confirmAction(row.id)
    ElMessage.success('确认成功')
    loadActions()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('确认失败: ' + (error.message || '未知错误'))
    }
  }
}

const handleSaveAction = async () => {
  try {
    await actionFormRef.value.validate()
    
    try {
      actionForm.parameters = JSON.parse(actionParametersText.value)
    } catch {
      ElMessage.error('参数JSON格式错误')
      return
    }

    await controlApi.createAction(actionForm)
    ElMessage.success('动作已提交')
    actionDialogVisible.value = false
    loadActions()
  } catch (error: any) {
    ElMessage.error('执行失败: ' + (error.message || '未知错误'))
  }
}

// 审计日志相关方法
const loadAuditLogs = async () => {
  auditLogsLoading.value = true
  try {
    const filters: any = { limit: 100 }
    if (auditFilters.action_id) filters.action_id = auditFilters.action_id
    if (auditFilters.user_id) filters.user_id = auditFilters.user_id
    if (auditFilters.event_type) filters.event_type = auditFilters.event_type
    
    const result = await controlApi.getAuditLogs(filters)
    auditLogs.value = result.logs
  } catch (error: any) {
    ElMessage.error('加载审计日志失败: ' + (error.message || '未知错误'))
  } finally {
    auditLogsLoading.value = false
  }
}

const resetAuditFilters = () => {
  Object.assign(auditFilters, {
    action_id: '',
    user_id: '',
    event_type: '',
  })
  loadAuditLogs()
}

const handleViewAuditDetails = (row: AuditLog) => {
  detailDialogTitle.value = '审计日志详情'
  detailData.value = row
  detailDialogVisible.value = true
}

// 权限相关方法
const loadPermissions = async () => {
  permissionsLoading.value = true
  try {
    const filters: any = { limit: 100 }
    if (permissionFilters.user_id) filters.user_id = permissionFilters.user_id
    if (permissionFilters.action_type) filters.action_type = permissionFilters.action_type
    
    const result = await controlApi.getPermissions(filters)
    permissions.value = result.permissions
  } catch (error: any) {
    ElMessage.error('加载权限失败: ' + (error.message || '未知错误'))
  } finally {
    permissionsLoading.value = false
  }
}

const resetPermissionFilters = () => {
  Object.assign(permissionFilters, {
    user_id: '',
    action_type: '',
  })
  loadPermissions()
}

const handleCreatePermission = () => {
  permissionDialogTitle.value = '新增权限'
  editingPermissionId.value = null
  Object.assign(permissionForm, {
    user_id: '',
    action_type: 'emergency_stop',
    enabled: true,
    max_severity: 3,
    require_confirmation: false,
    target_devices: [],
  })
  targetDevicesText.value = ''
  permissionDialogVisible.value = true
}

const handleEditPermission = (row: ControlPermission) => {
  permissionDialogTitle.value = '编辑权限'
  editingPermissionId.value = row.id
  Object.assign(permissionForm, {
    user_id: row.user_id,
    action_type: row.action_type,
    enabled: row.enabled,
    max_severity: row.max_severity,
    require_confirmation: row.require_confirmation,
    target_devices: row.target_devices,
  })
  targetDevicesText.value = row.target_devices.join('\n')
  permissionDialogVisible.value = true
}

const handleTogglePermission = async (row: ControlPermission) => {
  try {
    await controlApi.updatePermission(row.id, { enabled: row.enabled })
    ElMessage.success('更新成功')
  } catch (error: any) {
    row.enabled = !row.enabled
    ElMessage.error('更新失败: ' + (error.message || '未知错误'))
  }
}

const handleDeletePermission = async (row: ControlPermission) => {
  try {
    await ElMessageBox.confirm('确定要删除该权限吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await controlApi.deletePermission(row.id)
    ElMessage.success('删除成功')
    loadPermissions()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + (error.message || '未知错误'))
    }
  }
}

const handleSavePermission = async () => {
  try {
    await permissionFormRef.value.validate()
    
    permissionForm.target_devices = targetDevicesText.value
      .split('\n')
      .map(line => line.trim())
      .filter(line => line.length > 0)

    if (editingPermissionId.value === null) {
      await controlApi.createPermission(permissionForm)
      ElMessage.success('创建成功')
    } else {
      await controlApi.updatePermission(editingPermissionId.value, permissionForm)
      ElMessage.success('更新成功')
    }
    permissionDialogVisible.value = false
    loadPermissions()
  } catch (error: any) {
    ElMessage.error('保存失败: ' + (error.message || '未知错误'))
  }
}

// 工具方法
const getActionTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    emergency_stop: '紧急停止',
    shutdown: '关机',
    start: '启动',
    pause: '暂停',
    resume: '恢复',
    set_value: '设置值',
    call_method: '调用方法',
    send_command: '发送命令',
    custom: '自定义',
  }
  return labels[type] || type
}

const getActionStatusLabel = (status: string) => {
  const labels: Record<string, string> = {
    pending: '待执行',
    executing: '执行中',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消',
  }
  return labels[status] || status
}

const getActionStatusTagType = (status: string) => {
  const types: Record<string, string> = {
    pending: 'warning',
    executing: 'primary',
    completed: 'success',
    failed: 'danger',
    cancelled: 'info',
  }
  return types[status] || ''
}

const getEventTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    created: '创建',
    confirmed: '确认',
    executed: '执行',
    completed: '完成',
    failed: '失败',
    cancelled: '取消',
  }
  return labels[type] || type
}

const formatDateTime = (dateStr: string) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

onMounted(() => {
  loadPolicies()
})
</script>

<style scoped>
.control-page {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.tab-content {
  padding: 20px 0;
}

.toolbar {
  margin-bottom: 20px;
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
