<template>
  <div class="predictions-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>预测分析中心</span>
          <div>
            <el-button @click="showModels = !showModels">
              {{ showModels ? '隐藏模型' : '显示模型' }}
            </el-button>
            <el-button type="primary" @click="handleCreateModel">新增模型</el-button>
            <el-button type="success" @click="handleForecast">执行预测</el-button>
          </div>
        </div>
      </template>

      <!-- 预测模型列表（可折叠） -->
      <el-collapse v-if="showModels" v-model="activeModelCollapse" class="models-section">
        <el-collapse-item title="预测模型管理" name="models">
          <el-table :data="models" v-loading="modelsLoading" stripe>
            <el-table-column prop="name" label="模型名称" width="200" />
            <el-table-column prop="type" label="类型" width="150">
              <template #default="{ row }">
                <el-tag>{{ getTypeLabel(row.type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="version" label="版本" width="100" />
            <el-table-column prop="status" label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="getStatusTagType(row.status)">{{ getStatusLabel(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="accuracy" label="准确度" width="120">
              <template #default="{ row }">
                <el-progress
                  v-if="row.accuracy"
                  :percentage="row.accuracy * 100"
                  :color="getAccuracyColor(row.accuracy)"
                />
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column prop="training_samples" label="训练样本" width="120" />
            <el-table-column prop="file_path" label="文件路径" width="200" show-overflow-tooltip />
            <el-table-column label="操作" width="250" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="handleEditModel(row)">编辑</el-button>
                <el-button
                  v-if="row.status !== 'deployed'"
                  link
                  type="success"
                  @click="handleDeployModel(row)"
                >
                  部署
                </el-button>
                <el-button link type="info" @click="handleUploadModel(row)">上传模型</el-button>
                <el-button link type="danger" @click="handleDeleteModel(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-collapse-item>
      </el-collapse>

      <!-- 筛选条件 -->
      <el-form :inline="true" :model="filters" class="filter-form">
        <el-form-item label="预测类型">
          <el-select v-model="filters.type" placeholder="全部" clearable style="width: 150px">
            <el-option label="趋势预测" value="forecast" />
            <el-option label="异常预测" value="anomaly" />
            <el-option label="趋势分析" value="trend" />
            <el-option label="性能预测" value="performance" />
          </el-select>
        </el-form-item>
        <el-form-item label="设备ID">
          <el-input v-model="filters.device_id" placeholder="设备ID" clearable style="width: 150px" />
        </el-form-item>
        <el-form-item label="模型ID">
          <el-input v-model="filters.model_id" placeholder="模型ID" clearable style="width: 150px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadPredictions">查询</el-button>
          <el-button @click="resetFilters">重置</el-button>
        </el-form-item>
      </el-form>

      <!-- 预测结果列表 -->
      <el-table :data="predictions" v-loading="loading" stripe>
        <el-table-column prop="field_name" label="字段名" width="150" />
        <el-table-column prop="prediction_type" label="预测类型" width="120">
          <template #default="{ row }">
            <el-tag>{{ getTypeLabel(row.prediction_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="predicted_value" label="预测值" width="120">
          <template #default="{ row }">
            <span :style="{ color: getValueColor(row.predicted_value) }">
              {{ row.predicted_value.toFixed(2) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="confidence" label="置信度" width="120">
          <template #default="{ row }">
            <el-progress
              :percentage="row.confidence * 100"
              :color="getConfidenceColor(row.confidence)"
              :stroke-width="8"
            />
          </template>
        </el-table-column>
        <el-table-column prop="actual_value" label="实际值" width="120">
          <template #default="{ row }">
            <span v-if="row.actual_value">{{ row.actual_value.toFixed(2) }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="error_rate" label="误差率" width="100">
          <template #default="{ row }">
            <span v-if="row.error_rate" :style="{ color: getErrorColor(row.error_rate) }">
              {{ (row.error_rate * 100).toFixed(2) }}%
            </span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="device_id" label="设备ID" width="150" />
        <el-table-column prop="time_range_start" label="时间范围" width="200">
          <template #default="{ row }">
            {{ formatTime(row.time_range_start) }} ~ {{ formatTime(row.time_range_end) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
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
          @size-change="loadPredictions"
          @current-change="loadPredictions"
        />
      </div>
    </el-card>

    <!-- 创建/编辑模型对话框 -->
    <el-dialog v-model="modelDialogVisible" :title="modelDialogTitle" width="600px">
      <el-form :model="modelForm" :rules="modelRules" ref="modelFormRef" label-width="120px">
        <el-form-item label="模型名称" prop="name">
          <el-input v-model="modelForm.name" placeholder="请输入模型名称" />
        </el-form-item>
        <el-form-item label="模型类型" prop="type">
          <el-select v-model="modelForm.type" placeholder="请选择模型类型" style="width: 100%">
            <el-option label="线性回归" value="linear_regression" />
            <el-option label="神经网络" value="neural_network" />
            <el-option label="SVM" value="svm" />
            <el-option label="决策树" value="decision_tree" />
            <el-option label="LSTM" value="lstm" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item label="版本">
          <el-input v-model="modelForm.version" placeholder="例如: 1.0.0" />
        </el-form-item>
        <el-form-item label="文件路径">
          <el-input v-model="modelForm.file_path" placeholder="模型文件路径" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="modelForm.description" type="textarea" :rows="3" placeholder="请输入模型描述" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="modelDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmitModel">确定</el-button>
      </template>
    </el-dialog>

    <!-- 执行预测对话框 -->
    <el-dialog v-model="forecastDialogVisible" title="执行预测" width="600px">
      <el-form :model="forecastForm" :rules="forecastRules" ref="forecastFormRef" label-width="120px">
        <el-form-item label="模型" prop="model_id">
          <el-select v-model="forecastForm.model_id" placeholder="请选择模型" style="width: 100%">
            <el-option
              v-for="model in deployedModels"
              :key="model.id"
              :label="model.name"
              :value="model.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="设备ID" prop="device_id">
          <el-input v-model="forecastForm.device_id" placeholder="请输入设备ID" />
        </el-form-item>
        <el-form-item label="字段名" prop="field_name">
          <el-input v-model="forecastForm.field_name" placeholder="请输入要预测的字段名" />
        </el-form-item>
        <el-form-item label="预测结束时间" prop="time_range_end">
          <el-date-picker
            v-model="forecastForm.time_range_end"
            type="datetime"
            placeholder="选择预测结束时间"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="预测步数">
          <el-input-number v-model="forecastForm.forecast_steps" :min="1" :max="1000" style="width: 100%" />
          <span style="margin-left: 10px; color: #909399">预测的时间步数</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="forecastDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmitForecast" :loading="forecasting">执行预测</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  predictionApi,
  type Prediction,
  type PredictionModel,
  type PredictionFilters,
  type ModelFilters,
} from '@/services/predictionApi'

const loading = ref(false)
const modelsLoading = ref(false)
const forecasting = ref(false)
const predictions = ref<Prediction[]>([])
const models = ref<PredictionModel[]>([])
const deployedModels = ref<PredictionModel[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)
const showModels = ref(false)
const activeModelCollapse = ref(['models'])

const filters = reactive<PredictionFilters>({
  type: undefined,
  device_id: undefined,
  model_id: undefined,
})

const modelDialogVisible = ref(false)
const modelDialogTitle = ref('新增预测模型')
const modelFormRef = ref()
const modelForm = reactive({
  name: '',
  type: '',
  version: '1.0.0',
  file_path: '',
  description: '',
})
const editingModelId = ref<string | null>(null)

const modelRules = {
  name: [{ required: true, message: '请输入模型名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择模型类型', trigger: 'change' }],
}

const forecastDialogVisible = ref(false)
const forecastFormRef = ref()
const forecastForm = reactive({
  model_id: '',
  device_id: '',
  field_name: '',
  time_range_end: new Date(),
  forecast_steps: 24,
})

const forecastRules = {
  model_id: [{ required: true, message: '请选择模型', trigger: 'change' }],
  device_id: [{ required: true, message: '请输入设备ID', trigger: 'blur' }],
  field_name: [{ required: true, message: '请输入字段名', trigger: 'blur' }],
  time_range_end: [{ required: true, message: '请选择预测结束时间', trigger: 'change' }],
}

const loadPredictions = async () => {
  loading.value = true
  try {
    const result = await predictionApi.getPredictions({
      ...filters,
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value,
    })
    predictions.value = result.predictions
    total.value = result.count
  } catch (error: any) {
    ElMessage.error('加载预测结果失败: ' + (error.response?.data?.error || error.message))
  } finally {
    loading.value = false
  }
}

const loadModels = async () => {
  modelsLoading.value = true
  try {
    const result = await predictionApi.getModels({ limit: 1000 })
    models.value = result.models

    // 获取已部署的模型
    const deployedResult = await predictionApi.getModels({ status: 'deployed', limit: 100 })
    deployedModels.value = deployedResult.models
  } catch (error: any) {
    ElMessage.error('加载预测模型失败: ' + (error.response?.data?.error || error.message))
  } finally {
    modelsLoading.value = false
  }
}

const resetFilters = () => {
  filters.type = undefined
  filters.device_id = undefined
  filters.model_id = undefined
  loadPredictions()
}

const handleCreateModel = () => {
  modelDialogTitle.value = '新增预测模型'
  editingModelId.value = null
  modelForm.name = ''
  modelForm.type = ''
  modelForm.version = '1.0.0'
  modelForm.file_path = ''
  modelForm.description = ''
  modelDialogVisible.value = true
}

const handleEditModel = (model: PredictionModel) => {
  modelDialogTitle.value = '编辑预测模型'
  editingModelId.value = model.id
  modelForm.name = model.name
  modelForm.type = model.type
  modelForm.version = model.version
  modelForm.file_path = model.file_path || ''
  modelForm.description = model.description || ''
  modelDialogVisible.value = true
}

const handleSubmitModel = async () => {
  if (!modelFormRef.value) return
  await modelFormRef.value.validate(async (valid: boolean) => {
    if (!valid) return

    try {
      if (editingModelId.value) {
        await predictionApi.updateModel(editingModelId.value, modelForm)
        ElMessage.success('更新成功')
      } else {
        await predictionApi.createModel(modelForm)
        ElMessage.success('创建成功')
      }
      modelDialogVisible.value = false
      loadModels()
    } catch (error: any) {
      ElMessage.error((editingModelId.value ? '更新' : '创建') + '失败: ' + (error.response?.data?.error || error.message))
    }
  })
}

const handleDeployModel = async (model: PredictionModel) => {
  try {
    await ElMessageBox.confirm(`确定要部署模型 "${model.name}" 吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await predictionApi.deployModel(model.id)
    ElMessage.success('部署成功')
    loadModels()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('部署失败: ' + (error.response?.data?.error || error.message))
    }
  }
}

const handleDeleteModel = async (model: PredictionModel) => {
  try {
    await ElMessageBox.confirm(`确定要删除模型 "${model.name}" 吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await predictionApi.deleteModel(model.id)
    ElMessage.success('删除成功')
    loadModels()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + (error.response?.data?.error || error.message))
    }
  }
}

const handleUploadModel = (model: PredictionModel) => {
  // 创建文件输入元素
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = '.pkl,.h5,.onnx,.pb,.pt,.pth'
  input.onchange = async (e: Event) => {
    const target = e.target as HTMLInputElement
    if (target.files && target.files.length > 0) {
      const file = target.files[0]
      const formData = new FormData()
      formData.append('model', file)

      try {
        // 使用预测模型的API端点上传模型文件
        const response = await fetch(`${import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'}/predictions/models/${model.id}/upload`, {
          method: 'POST',
          body: formData,
        })

        if (!response.ok) {
          const error = await response.json()
          throw new Error(error.error || '上传失败')
        }

        const result = await response.json()
        ElMessage.success('模型文件上传成功')
        
        // 更新模型文件路径
        await predictionApi.updateModel(model.id, {
          file_path: result.model_path,
        })
        
        loadModels()
      } catch (error: any) {
        ElMessage.error('上传失败: ' + error.message)
      }
    }
  }
  input.click()
}

const handleForecast = () => {
  forecastForm.model_id = ''
  forecastForm.device_id = ''
  forecastForm.field_name = ''
  forecastForm.time_range_end = new Date()
  forecastForm.forecast_steps = 24
  forecastDialogVisible.value = true
}

const handleSubmitForecast = async () => {
  if (!forecastFormRef.value) return
  await forecastFormRef.value.validate(async (valid: boolean) => {
    if (!valid) return

    forecasting.value = true
    try {
      const result = await predictionApi.forecast({
        model_id: forecastForm.model_id,
        device_id: forecastForm.device_id,
        field_name: forecastForm.field_name,
        time_range_end: forecastForm.time_range_end.toISOString(),
        forecast_steps: forecastForm.forecast_steps,
      })
      ElMessage.success(result.message || '预测完成')
      forecastDialogVisible.value = false
      loadPredictions()
    } catch (error: any) {
      ElMessage.error('预测失败: ' + (error.response?.data?.error || error.message))
    } finally {
      forecasting.value = false
    }
  })
}

const handleView = (prediction: Prediction) => {
  // TODO: 显示预测详情
  ElMessage.info('查看预测详情功能待实现')
}

const getTypeLabel = (type: string) => {
  const map: Record<string, string> = {
    forecast: '趋势预测',
    anomaly: '异常预测',
    trend: '趋势分析',
    performance: '性能预测',
    linear_regression: '线性回归',
    neural_network: '神经网络',
    svm: 'SVM',
    decision_tree: '决策树',
    lstm: 'LSTM',
    custom: '自定义',
  }
  return map[type] || type
}

const getStatusLabel = (status: string) => {
  const map: Record<string, string> = {
    draft: '草稿',
    training: '训练中',
    deployed: '已部署',
    archived: '已归档',
  }
  return map[status] || status
}

const getStatusTagType = (status: string) => {
  const map: Record<string, string> = {
    draft: 'info',
    training: 'warning',
    deployed: 'success',
    archived: '',
  }
  return map[status] || ''
}

const getValueColor = (value: number) => {
  // 根据值的大小返回不同颜色
  if (value > 0) return '#67c23a'
  if (value < 0) return '#f56c6c'
  return '#909399'
}

const getConfidenceColor = (confidence: number) => {
  if (confidence >= 0.8) return '#67c23a'
  if (confidence >= 0.6) return '#e6a23c'
  return '#f56c6c'
}

const getAccuracyColor = (accuracy: number) => {
  if (accuracy >= 0.8) return '#67c23a'
  if (accuracy >= 0.6) return '#e6a23c'
  return '#f56c6c'
}

const getErrorColor = (errorRate: number) => {
  if (errorRate < 0.1) return '#67c23a'
  if (errorRate < 0.2) return '#e6a23c'
  return '#f56c6c'
}

const formatTime = (timeStr: string) => {
  if (!timeStr) return '-'
  return new Date(timeStr).toLocaleString('zh-CN')
}

onMounted(() => {
  loadPredictions()
  loadModels()
})
</script>

<style scoped>
.predictions-page {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.models-section {
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
