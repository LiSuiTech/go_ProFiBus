<template>
  <div class="data-management-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>数据管理</span>
        </div>
      </template>

      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <!-- 清洗规则标签页 -->
        <el-tab-pane label="清洗规则" name="cleaning-rules">
          <div class="tab-content">
            <div class="toolbar">
              <el-button type="primary" @click="handleCreateCleaningRule">新增规则</el-button>
              <el-button @click="loadCleaningRules">刷新</el-button>
            </div>

            <!-- 筛选条件 -->
            <el-form :inline="true" :model="cleaningRuleFilters" class="filter-form">
              <el-form-item label="规则类型">
                <el-select v-model="cleaningRuleFilters.type" placeholder="全部" clearable style="width: 150px">
                  <el-option label="去重" value="deduplicate" />
                  <el-option label="异常值过滤" value="outlier_filter" />
                  <el-option label="缺失值填充" value="missing_fill" />
                  <el-option label="标准化" value="normalize" />
                  <el-option label="平滑处理" value="smooth" />
                  <el-option label="数据验证" value="validate" />
                </el-select>
              </el-form-item>
              <el-form-item label="启用状态">
                <el-select v-model="cleaningRuleFilters.enabled" placeholder="全部" clearable style="width: 150px">
                  <el-option label="启用" :value="true" />
                  <el-option label="禁用" :value="false" />
                </el-select>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="loadCleaningRules">查询</el-button>
                <el-button @click="resetCleaningRuleFilters">重置</el-button>
              </el-form-item>
            </el-form>

            <el-table :data="cleaningRules" v-loading="cleaningRulesLoading" stripe>
              <el-table-column prop="name" label="规则名称" width="200" />
              <el-table-column prop="rule_type" label="规则类型" width="120">
                <template #default="{ row }">
                  <el-tag>{{ getCleaningRuleTypeLabel(row.rule_type) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="enabled" label="启用" width="80">
                <template #default="{ row }">
                  <el-switch v-model="row.enabled" @change="handleToggleCleaningRule(row)" />
                </template>
              </el-table-column>
              <el-table-column prop="priority" label="优先级" width="100" />
              <el-table-column prop="description" label="描述" show-overflow-tooltip />
              <el-table-column label="操作" width="150" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" @click="handleViewCleaningRule(row)">查看</el-button>
                  <el-button link type="danger" @click="handleDeleteCleaningRule(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- 归档策略标签页 -->
        <el-tab-pane label="归档策略" name="archive-policies">
          <div class="tab-content">
            <div class="toolbar">
              <el-button type="primary" @click="handleCreateArchivePolicy">新增策略</el-button>
              <el-button @click="loadArchivePolicies">刷新</el-button>
            </div>

            <!-- 筛选条件 -->
            <el-form :inline="true" :model="archivePolicyFilters" class="filter-form">
              <el-form-item label="数据源类型">
                <el-input v-model="archivePolicyFilters.source_type" placeholder="数据源类型" clearable style="width: 150px" />
              </el-form-item>
              <el-form-item label="启用状态">
                <el-select v-model="archivePolicyFilters.enabled" placeholder="全部" clearable style="width: 150px">
                  <el-option label="启用" :value="true" />
                  <el-option label="禁用" :value="false" />
                </el-select>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="loadArchivePolicies">查询</el-button>
                <el-button @click="resetArchivePolicyFilters">重置</el-button>
              </el-form-item>
            </el-form>

            <el-table :data="archivePolicies" v-loading="archivePoliciesLoading" stripe>
              <el-table-column prop="name" label="策略名称" width="200" />
              <el-table-column prop="source_type" label="数据源类型" width="120" />
              <el-table-column prop="source_id" label="数据源ID" width="150" />
              <el-table-column prop="enabled" label="启用" width="80">
                <template #default="{ row }">
                  <el-switch v-model="row.enabled" @change="handleToggleArchivePolicy(row)" />
                </template>
              </el-table-column>
              <el-table-column prop="retention_days" label="保留天数" width="100" />
              <el-table-column prop="archive_after_days" label="归档天数" width="100" />
              <el-table-column prop="compression_enabled" label="压缩" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.compression_enabled ? 'success' : 'info'">
                    {{ row.compression_enabled ? '是' : '否' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="next_run_at" label="下次执行" width="180">
                <template #default="{ row }">
                  {{ row.next_run_at ? formatDateTime(row.next_run_at) : '-' }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="250" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" @click="handleViewArchivePolicy(row)">查看</el-button>
                  <el-button link type="success" @click="handleExecuteArchive(row)">执行</el-button>
                  <el-button link type="info" @click="handleViewArchiveStats(row)">统计</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- 归档记录标签页 -->
        <el-tab-pane label="归档记录" name="archive-records">
          <div class="tab-content">
            <div class="toolbar">
              <el-button @click="loadArchiveRecords">刷新</el-button>
            </div>

            <!-- 筛选条件 -->
            <el-form :inline="true" :model="archiveRecordFilters" class="filter-form">
              <el-form-item label="策略ID">
                <el-input v-model="archiveRecordFilters.policy_id" placeholder="策略ID" clearable style="width: 150px" />
              </el-form-item>
              <el-form-item label="状态">
                <el-select v-model="archiveRecordFilters.status" placeholder="全部" clearable style="width: 150px">
                  <el-option label="待执行" value="pending" />
                  <el-option label="执行中" value="running" />
                  <el-option label="已完成" value="completed" />
                  <el-option label="失败" value="failed" />
                </el-select>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="loadArchiveRecords">查询</el-button>
                <el-button @click="resetArchiveRecordFilters">重置</el-button>
              </el-form-item>
            </el-form>

            <el-table :data="archiveRecords" v-loading="archiveRecordsLoading" stripe>
              <el-table-column prop="policy_id" label="策略ID" width="200" />
              <el-table-column prop="source_type" label="数据源类型" width="120" />
              <el-table-column prop="status" label="状态" width="100">
                <template #default="{ row }">
                  <el-tag :type="getArchiveStatusTagType(row.status)">{{ getArchiveStatusLabel(row.status) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="record_count" label="记录数" width="120" />
              <el-table-column prop="archive_size" label="归档大小" width="120">
                <template #default="{ row }">
                  {{ formatFileSize(row.archive_size) }}
                </template>
              </el-table-column>
              <el-table-column prop="start_time" label="开始时间" width="180">
                <template #default="{ row }">
                  {{ formatDateTime(row.start_time) }}
                </template>
              </el-table-column>
              <el-table-column prop="completed_at" label="完成时间" width="180">
                <template #default="{ row }">
                  {{ row.completed_at ? formatDateTime(row.completed_at) : '-' }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="100" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" @click="handleViewArchiveRecord(row)">查看</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- 生命周期配置标签页 -->
        <el-tab-pane label="生命周期配置" name="lifecycle-configs">
          <div class="tab-content">
            <div class="toolbar">
              <el-button type="primary" @click="handleCreateLifecycleConfig">新增配置</el-button>
              <el-button @click="loadLifecycleConfigs">刷新</el-button>
            </div>

            <el-table :data="lifecycleConfigs" v-loading="lifecycleConfigsLoading" stripe>
              <el-table-column prop="source_type" label="数据源类型" width="150" />
              <el-table-column prop="source_id" label="数据源ID" width="200" />
              <el-table-column prop="hot_storage_days" label="热存储(天)" width="120" />
              <el-table-column prop="warm_storage_days" label="温存储(天)" width="120" />
              <el-table-column prop="cold_storage_days" label="冷存储(天)" width="120" />
              <el-table-column prop="delete_after_days" label="删除(天)" width="120">
                <template #default="{ row }">
                  {{ row.delete_after_days || '-' }}
                </template>
              </el-table-column>
              <el-table-column prop="compression_after_days" label="压缩(天)" width="120">
                <template #default="{ row }">
                  {{ row.compression_after_days || '-' }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="100" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" @click="handleViewLifecycleConfig(row)">查看</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- 数据清洗标签页（Dry Run 预览） -->
        <el-tab-pane label="数据清洗预览" name="clean-data">
          <div class="tab-content">
            <el-card>
              <template #header>
                <span>数据清洗预览（Dry Run）</span>
              </template>
              <el-form :model="cleanDataForm" label-width="120px" style="max-width: 600px">
                <el-form-item label="数据源类型" required>
                  <el-input v-model="cleanDataForm.source_type" placeholder="例如: device, channel" />
                </el-form-item>
                <el-form-item label="数据源ID" required>
                  <el-input v-model="cleanDataForm.source_id" placeholder="数据源ID" />
                </el-form-item>
                <el-form-item label="样例数据(JSON)">
                  <el-input
                    v-model="cleanDataText"
                    type="textarea"
                    :rows="10"
                    placeholder='请输入单条JSON数据，例如: {"temperature": 120, "vibration": 0.8, "speed": 1500}'
                  />
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" @click="handleCleanData" :loading="cleaningData">执行清洗</el-button>
                  <el-button @click="resetCleanDataForm">重置</el-button>
                  <el-button type="success" text @click="fillSampleData">填充示例数据</el-button>
                </el-form-item>
              </el-form>

              <el-card v-if="cleanDataResult" style="margin-top: 20px">
                <template #header>
                  <span>清洗前后对比</span>
                </template>
                <el-alert
                  :type="cleanDataResult.was_cleaned ? 'success' : 'info'"
                  :closable="false"
                  style="margin-bottom: 16px"
                >
                  <template #title>
                    <span>
                      是否发生清洗：
                      <strong>{{ cleanDataResult.was_cleaned ? '是（数据已被规则修改或过滤）' : '否（数据未被修改）' }}</strong>
                    </span>
                  </template>
                </el-alert>

                <el-row :gutter="20">
                  <el-col :span="12">
                    <el-card>
                      <template #header>
                        <span>原始数据</span>
                      </template>
                      <pre style="max-height: 300px; overflow: auto; background: #f5f7fa; padding: 10px;">
{{ formatJson(originalSampleData) }}
                      </pre>
                    </el-card>
                  </el-col>
                  <el-col :span="12">
                    <el-card>
                      <template #header>
                        <span>清洗后数据</span>
                      </template>
                      <pre style="max-height: 300px; overflow: auto; background: #f5f7fa; padding: 10px;">
{{ formatJson(cleanDataResult.cleaned_data) }}
                      </pre>
                    </el-card>
                  </el-col>
                </el-row>
                <div style="margin-top: 16px; text-align: right;">
                  <el-button @click="downloadCleanedData">下载清洗后数据</el-button>
                </div>
              </el-card>
            </el-card>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 清洗规则对话框 -->
    <el-dialog v-model="cleaningRuleDialogVisible" :title="cleaningRuleDialogTitle" width="600px">
      <el-form :model="cleaningRuleForm" :rules="cleaningRuleRules" ref="cleaningRuleFormRef" label-width="120px">
        <el-form-item label="规则名称" prop="name">
          <el-input v-model="cleaningRuleForm.name" placeholder="请输入规则名称" />
        </el-form-item>
        <el-form-item label="规则类型" prop="rule_type">
          <el-select v-model="cleaningRuleForm.rule_type" placeholder="请选择规则类型" style="width: 100%">
            <el-option label="去重" value="deduplicate" />
            <el-option label="异常值过滤" value="outlier_filter" />
            <el-option label="缺失值填充" value="missing_fill" />
            <el-option label="标准化" value="normalize" />
            <el-option label="平滑处理" value="smooth" />
            <el-option label="数据验证" value="validate" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="cleaningRuleForm.description" type="textarea" :rows="3" placeholder="请输入描述" />
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="cleaningRuleForm.priority" :min="0" :max="100" />
        </el-form-item>
        <el-form-item label="配置(JSON)">
          <el-input v-model="cleaningRuleConfigText" type="textarea" :rows="5" placeholder='JSON格式，例如: {"threshold": 100}' />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="cleaningRuleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveCleaningRule">保存</el-button>
      </template>
    </el-dialog>

    <!-- 归档策略对话框 -->
    <el-dialog v-model="archivePolicyDialogVisible" :title="archivePolicyDialogTitle" width="700px">
      <el-form :model="archivePolicyForm" :rules="archivePolicyRules" ref="archivePolicyFormRef" label-width="140px">
        <el-form-item label="策略名称" prop="name">
          <el-input v-model="archivePolicyForm.name" placeholder="请输入策略名称" />
        </el-form-item>
        <el-form-item label="数据源类型" prop="source_type">
          <el-input v-model="archivePolicyForm.source_type" placeholder="例如: device, channel" />
        </el-form-item>
        <el-form-item label="数据源ID">
          <el-input v-model="archivePolicyForm.source_id" placeholder="数据源ID（可选）" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="archivePolicyForm.description" type="textarea" :rows="3" placeholder="请输入描述" />
        </el-form-item>
        <el-form-item label="保留天数">
          <el-input-number v-model="archivePolicyForm.retention_days" :min="1" />
        </el-form-item>
        <el-form-item label="归档天数">
          <el-input-number v-model="archivePolicyForm.archive_after_days" :min="1" />
        </el-form-item>
        <el-form-item label="执行间隔(小时)">
          <el-input-number v-model="archivePolicyForm.run_interval_hours" :min="1" />
        </el-form-item>
        <el-form-item label="启用压缩">
          <el-switch v-model="archivePolicyForm.compression_enabled" />
        </el-form-item>
        <el-form-item label="归档位置">
          <el-input v-model="archivePolicyForm.archive_location" placeholder="归档文件存储路径" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="archivePolicyDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveArchivePolicy">保存</el-button>
      </template>
    </el-dialog>

    <!-- 生命周期配置对话框 -->
    <el-dialog v-model="lifecycleConfigDialogVisible" :title="lifecycleConfigDialogTitle" width="600px">
      <el-form :model="lifecycleConfigForm" :rules="lifecycleConfigRules" ref="lifecycleConfigFormRef" label-width="140px">
        <el-form-item label="数据源类型" prop="source_type">
          <el-input v-model="lifecycleConfigForm.source_type" placeholder="例如: device, channel" />
        </el-form-item>
        <el-form-item label="数据源ID" prop="source_id">
          <el-input v-model="lifecycleConfigForm.source_id" placeholder="数据源ID" />
        </el-form-item>
        <el-form-item label="热存储(天)">
          <el-input-number v-model="lifecycleConfigForm.hot_storage_days" :min="1" />
        </el-form-item>
        <el-form-item label="温存储(天)">
          <el-input-number v-model="lifecycleConfigForm.warm_storage_days" :min="1" />
        </el-form-item>
        <el-form-item label="冷存储(天)">
          <el-input-number v-model="lifecycleConfigForm.cold_storage_days" :min="1" />
        </el-form-item>
        <el-form-item label="删除(天)">
          <el-input-number v-model="lifecycleConfigForm.delete_after_days" :min="1" />
          <span style="margin-left: 10px; color: #909399;">留空表示不删除</span>
        </el-form-item>
        <el-form-item label="压缩(天)">
          <el-input-number v-model="lifecycleConfigForm.compression_after_days" :min="1" />
          <span style="margin-left: 10px; color: #909399;">留空表示不压缩</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="lifecycleConfigDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveLifecycleConfig">保存</el-button>
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

    <!-- 归档统计对话框 -->
    <el-dialog v-model="archiveStatsDialogVisible" title="归档统计" width="600px">
      <el-descriptions :column="1" border v-if="archiveStats">
        <el-descriptions-item label="总记录数">{{ archiveStats.total_records }}</el-descriptions-item>
        <el-descriptions-item label="总大小">{{ formatFileSize(archiveStats.total_size) }}</el-descriptions-item>
        <el-descriptions-item label="归档次数">{{ archiveStats.archive_count }}</el-descriptions-item>
        <el-descriptions-item label="最后归档时间">
          {{ archiveStats.last_archive_time ? formatDateTime(archiveStats.last_archive_time) : '-' }}
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  dataManagementApi,
  type CleaningRule,
  type ArchivePolicy,
  type ArchiveRecord,
  type LifecycleConfig,
  type CleanDataResponse,
} from '../services/dataManagementApi'

const activeTab = ref('cleaning-rules')

// 清洗规则相关
const cleaningRules = ref<CleaningRule[]>([])
const cleaningRulesLoading = ref(false)
const cleaningRuleDialogVisible = ref(false)
const cleaningRuleDialogTitle = ref('新增规则')
const cleaningRuleFormRef = ref()
const cleaningRuleForm = reactive({
  name: '',
  description: '',
  rule_type: 'deduplicate' as any,
  priority: 0,
  config: {},
})
const cleaningRuleConfigText = ref('{}')
const cleaningRuleFilters = reactive({
  type: '' as any,
  enabled: undefined as boolean | undefined,
})
const cleaningRuleRules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  rule_type: [{ required: true, message: '请选择规则类型', trigger: 'change' }],
}

// 归档策略相关
const archivePolicies = ref<ArchivePolicy[]>([])
const archivePoliciesLoading = ref(false)
const archivePolicyDialogVisible = ref(false)
const archivePolicyDialogTitle = ref('新增策略')
const archivePolicyFormRef = ref()
const archivePolicyForm = reactive({
  name: '',
  description: '',
  source_type: '',
  source_id: '',
  retention_days: 365,
  archive_after_days: 30,
  compression_enabled: true,
  archive_location: '',
  run_interval_hours: 24,
})
const archivePolicyFilters = reactive({
  source_type: '',
  enabled: undefined as boolean | undefined,
})
const archivePolicyRules = {
  name: [{ required: true, message: '请输入策略名称', trigger: 'blur' }],
  source_type: [{ required: true, message: '请输入数据源类型', trigger: 'blur' }],
}

// 归档记录相关
const archiveRecords = ref<ArchiveRecord[]>([])
const archiveRecordsLoading = ref(false)
const archiveRecordFilters = reactive({
  policy_id: '',
  status: '' as any,
})

// 生命周期配置相关
const lifecycleConfigs = ref<LifecycleConfig[]>([])
const lifecycleConfigsLoading = ref(false)
const lifecycleConfigDialogVisible = ref(false)
const lifecycleConfigDialogTitle = ref('新增配置')
const lifecycleConfigFormRef = ref()
const lifecycleConfigForm = reactive({
  source_type: '',
  source_id: '',
  hot_storage_days: 7,
  warm_storage_days: 30,
  cold_storage_days: 365,
  delete_after_days: undefined as number | undefined,
  compression_after_days: undefined as number | undefined,
})
const lifecycleConfigRules = {
  source_type: [{ required: true, message: '请输入数据源类型', trigger: 'blur' }],
  source_id: [{ required: true, message: '请输入数据源ID', trigger: 'blur' }],
}

// 数据清洗相关
const cleanDataForm = reactive({
  source_type: '',
  source_id: '',
})
const cleanDataText = ref('{}')
const cleaningData = ref(false)
const cleanDataResult = ref<CleanDataResponse | null>(null)
const originalSampleData = ref<Record<string, any> | null>(null)

// 详情对话框
const detailDialogVisible = ref(false)
const detailDialogTitle = ref('详情')
const detailData = ref<any>({})

// 归档统计对话框
const archiveStatsDialogVisible = ref(false)
const archiveStats = ref<any>(null)

// 标签页切换
const handleTabChange = (tab: string) => {
  if (tab === 'cleaning-rules') {
    loadCleaningRules()
  } else if (tab === 'archive-policies') {
    loadArchivePolicies()
  } else if (tab === 'archive-records') {
    loadArchiveRecords()
  } else if (tab === 'lifecycle-configs') {
    loadLifecycleConfigs()
  }
}

// 清洗规则相关方法
const loadCleaningRules = async () => {
  cleaningRulesLoading.value = true
  try {
    const filters: any = { limit: 100 }
    if (cleaningRuleFilters.type) filters.type = cleaningRuleFilters.type
    if (cleaningRuleFilters.enabled !== undefined) filters.enabled = cleaningRuleFilters.enabled

    const result = await dataManagementApi.getCleaningRules(filters)
    cleaningRules.value = result.rules
  } catch (error: any) {
    ElMessage.error('加载清洗规则失败: ' + (error.message || '未知错误'))
  } finally {
    cleaningRulesLoading.value = false
  }
}

const resetCleaningRuleFilters = () => {
  Object.assign(cleaningRuleFilters, {
    type: '',
    enabled: undefined,
  })
  loadCleaningRules()
}

const handleCreateCleaningRule = () => {
  cleaningRuleDialogTitle.value = '新增规则'
  Object.assign(cleaningRuleForm, {
    name: '',
    description: '',
    rule_type: 'deduplicate',
    priority: 0,
    config: {},
  })
  cleaningRuleConfigText.value = '{}'
  cleaningRuleDialogVisible.value = true
}

const handleViewCleaningRule = (row: CleaningRule) => {
  detailDialogTitle.value = '清洗规则详情'
  detailData.value = row
  detailDialogVisible.value = true
}

const handleToggleCleaningRule = async (row: CleaningRule) => {
  // TODO: 实现更新API
  ElMessage.info('更新功能待实现')
}

const handleDeleteCleaningRule = async (row: CleaningRule) => {
  try {
    await ElMessageBox.confirm('确定要删除该规则吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    // TODO: 实现删除API
    ElMessage.success('删除成功')
    loadCleaningRules()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

const handleSaveCleaningRule = async () => {
  try {
    await cleaningRuleFormRef.value.validate()

    try {
      cleaningRuleForm.config = JSON.parse(cleaningRuleConfigText.value)
    } catch {
      ElMessage.error('配置JSON格式错误')
      return
    }

    await dataManagementApi.createCleaningRule(cleaningRuleForm)
    ElMessage.success('创建成功')
    cleaningRuleDialogVisible.value = false
    loadCleaningRules()
  } catch (error: any) {
    ElMessage.error('保存失败: ' + (error.message || '未知错误'))
  }
}

// 归档策略相关方法
const loadArchivePolicies = async () => {
  archivePoliciesLoading.value = true
  try {
    const filters: any = { limit: 100 }
    if (archivePolicyFilters.source_type) filters.source_type = archivePolicyFilters.source_type
    if (archivePolicyFilters.enabled !== undefined) filters.enabled = archivePolicyFilters.enabled

    const result = await dataManagementApi.getArchivePolicies(filters)
    archivePolicies.value = result.policies
  } catch (error: any) {
    ElMessage.error('加载归档策略失败: ' + (error.message || '未知错误'))
  } finally {
    archivePoliciesLoading.value = false
  }
}

const resetArchivePolicyFilters = () => {
  Object.assign(archivePolicyFilters, {
    source_type: '',
    enabled: undefined,
  })
  loadArchivePolicies()
}

const handleCreateArchivePolicy = () => {
  archivePolicyDialogTitle.value = '新增策略'
  Object.assign(archivePolicyForm, {
    name: '',
    description: '',
    source_type: '',
    source_id: '',
    retention_days: 365,
    archive_after_days: 30,
    compression_enabled: true,
    archive_location: '',
    run_interval_hours: 24,
  })
  archivePolicyDialogVisible.value = true
}

const handleViewArchivePolicy = (row: ArchivePolicy) => {
  detailDialogTitle.value = '归档策略详情'
  detailData.value = row
  detailDialogVisible.value = true
}

const handleToggleArchivePolicy = async (row: ArchivePolicy) => {
  // TODO: 实现更新API
  ElMessage.info('更新功能待实现')
}

const handleExecuteArchive = async (row: ArchivePolicy) => {
  try {
    await ElMessageBox.confirm('确定要执行该归档策略吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await dataManagementApi.executeArchive(row.id)
    ElMessage.success('归档执行成功')
    loadArchivePolicies()
    loadArchiveRecords()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('执行失败: ' + (error.message || '未知错误'))
    }
  }
}

const handleViewArchiveStats = async (row: ArchivePolicy) => {
  try {
    const stats = await dataManagementApi.getArchiveStats(row.id)
    archiveStats.value = stats
    archiveStatsDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error('获取统计失败: ' + (error.message || '未知错误'))
  }
}

const handleSaveArchivePolicy = async () => {
  try {
    await archivePolicyFormRef.value.validate()
    await dataManagementApi.createArchivePolicy(archivePolicyForm)
    ElMessage.success('创建成功')
    archivePolicyDialogVisible.value = false
    loadArchivePolicies()
  } catch (error: any) {
    ElMessage.error('保存失败: ' + (error.message || '未知错误'))
  }
}

// 归档记录相关方法
const loadArchiveRecords = async () => {
  archiveRecordsLoading.value = true
  try {
    const filters: any = { limit: 100 }
    if (archiveRecordFilters.policy_id) filters.policy_id = archiveRecordFilters.policy_id
    if (archiveRecordFilters.status) filters.status = archiveRecordFilters.status

    const result = await dataManagementApi.getArchiveRecords(filters)
    archiveRecords.value = result.records
  } catch (error: any) {
    ElMessage.error('加载归档记录失败: ' + (error.message || '未知错误'))
  } finally {
    archiveRecordsLoading.value = false
  }
}

const resetArchiveRecordFilters = () => {
  Object.assign(archiveRecordFilters, {
    policy_id: '',
    status: '',
  })
  loadArchiveRecords()
}

const handleViewArchiveRecord = (row: ArchiveRecord) => {
  detailDialogTitle.value = '归档记录详情'
  detailData.value = row
  detailDialogVisible.value = true
}

// 生命周期配置相关方法
const loadLifecycleConfigs = async () => {
  lifecycleConfigsLoading.value = true
  try {
    // TODO: 实现列表API，当前使用空数组
    lifecycleConfigs.value = []
  } catch (error: any) {
    ElMessage.error('加载生命周期配置失败: ' + (error.message || '未知错误'))
  } finally {
    lifecycleConfigsLoading.value = false
  }
}

const handleCreateLifecycleConfig = () => {
  lifecycleConfigDialogTitle.value = '新增配置'
  Object.assign(lifecycleConfigForm, {
    source_type: '',
    source_id: '',
    hot_storage_days: 7,
    warm_storage_days: 30,
    cold_storage_days: 365,
    delete_after_days: undefined,
    compression_after_days: undefined,
  })
  lifecycleConfigDialogVisible.value = true
}

const handleViewLifecycleConfig = async (row: LifecycleConfig) => {
  try {
    const config = await dataManagementApi.getLifecycleConfig(row.source_type, row.source_id)
    detailDialogTitle.value = '生命周期配置详情'
    detailData.value = config
    detailDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error('获取配置失败: ' + (error.message || '未知错误'))
  }
}

const handleSaveLifecycleConfig = async () => {
  try {
    await lifecycleConfigFormRef.value.validate()
    await dataManagementApi.createLifecycleConfig(lifecycleConfigForm)
    ElMessage.success('创建成功')
    lifecycleConfigDialogVisible.value = false
    loadLifecycleConfigs()
  } catch (error: any) {
    ElMessage.error('保存失败: ' + (error.message || '未知错误'))
  }
}

// 数据清洗相关方法
const resetCleanDataForm = () => {
  Object.assign(cleanDataForm, {
    source_type: '',
    source_id: '',
  })
  cleanDataText.value = '{}'
  cleanDataResult.value = null
  originalSampleData.value = null
}

const handleCleanData = async () => {
  if (!cleanDataForm.source_type || !cleanDataForm.source_id) {
    ElMessage.warning('请填写数据源类型和数据源ID')
    return
  }

  let data: any
  try {
    data = JSON.parse(cleanDataText.value)
    if (Array.isArray(data) || typeof data !== 'object' || data === null) {
      ElMessage.error('数据必须是JSON对象格式（单条样例数据）')
      return
    }
  } catch {
    ElMessage.error('数据JSON格式错误')
    return
  }

  // 记录原始样例数据，便于对比展示
  originalSampleData.value = data

  cleaningData.value = true
  try {
    const result = await dataManagementApi.cleanData({
      source_type: cleanDataForm.source_type,
      source_id: cleanDataForm.source_id,
      data,
    })
    cleanDataResult.value = result
    ElMessage.success('数据清洗完成')
  } catch (error: any) {
    ElMessage.error('数据清洗失败: ' + (error.message || '未知错误'))
  } finally {
    cleaningData.value = false
  }
}

const downloadCleanedData = () => {
  if (!cleanDataResult.value) return

  const dataStr = JSON.stringify(cleanDataResult.value.cleaned_data, null, 2)
  const blob = new Blob([dataStr], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `cleaned_data_${new Date().getTime()}.json`
  link.click()
  URL.revokeObjectURL(url)
}

// 通用 JSON 格式化（用于预览）
const formatJson = (val: any) => {
  if (val === null || val === undefined) return ''
  try {
    return JSON.stringify(val, null, 2)
  } catch {
    return String(val)
  }
}

// 填充示例数据，方便快速体验清洗效果
const fillSampleData = () => {
  cleanDataForm.source_type = 'device'
  cleanDataForm.source_id = 'device_demo_001'
  cleanDataText.value = JSON.stringify(
    {
      temperature: 120,   // 超过正常范围，适合配合异常值过滤 / 标准化规则
      vibration: 0.85,
      speed: 1500,
      status: 'ok',
      reserved: null,     // 可用于缺失值填充规则
    },
    null,
    2,
  )
  cleanDataResult.value = null
  originalSampleData.value = null
}

// 工具方法
const getCleaningRuleTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    deduplicate: '去重',
    outlier_filter: '异常值过滤',
    missing_fill: '缺失值填充',
    normalize: '标准化',
    smooth: '平滑处理',
    validate: '数据验证',
  }
  return labels[type] || type
}

const getArchiveStatusLabel = (status: string) => {
  const labels: Record<string, string> = {
    pending: '待执行',
    running: '执行中',
    completed: '已完成',
    failed: '失败',
  }
  return labels[status] || status
}

const getArchiveStatusTagType = (status: string) => {
  const types: Record<string, string> = {
    pending: 'info',
    running: 'primary',
    completed: 'success',
    failed: 'danger',
  }
  return types[status] || ''
}

const formatDateTime = (dateStr: string) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

const formatFileSize = (bytes: number) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

onMounted(() => {
  loadCleaningRules()
})
</script>

<style scoped>
.data-management-page {
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
</style>
