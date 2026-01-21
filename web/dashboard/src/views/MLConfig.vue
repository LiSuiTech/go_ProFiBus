<template>
  <div class="ml-config">
    <div class="header">
      <h1>AI模型检测配置</h1>
      <p class="subtitle">配置多数据融合和AI模型检测</p>
    </div>

    <!-- 标签页 -->
    <div class="tabs">
      <button
        :class="['tab', { active: activeTab === 'analyzers' }]"
        @click="activeTab = 'analyzers'"
      >
        ML分析器
      </button>
      <button
        :class="['tab', { active: activeTab === 'fusion' }]"
        @click="activeTab = 'fusion'"
      >
        数据融合
      </button>
      <button
        :class="['tab', { active: activeTab === 'models' }]"
        @click="activeTab = 'models'"
      >
        模型管理
      </button>
    </div>

    <!-- ML分析器配置 -->
    <div v-if="activeTab === 'analyzers'" class="content">
      <div class="toolbar">
        <button class="btn-primary" @click="showCreateAnalyzerDialog">
          + 创建分析器
        </button>
      </div>

      <div class="analyzer-list">
        <div
          v-for="analyzer in analyzers"
          :key="analyzer.id"
          class="analyzer-card"
        >
          <div class="card-header">
            <h3>{{ analyzer.name }}</h3>
            <div class="actions">
              <button @click="editAnalyzer(analyzer.id)">编辑</button>
              <button @click="viewStats(analyzer.id)">统计</button>
              <button class="danger" @click="deleteAnalyzer(analyzer.id)">删除</button>
            </div>
          </div>
          <div class="card-body">
            <div class="info-row">
              <span class="label">模型类型:</span>
              <span class="value">{{ analyzer.type }}</span>
            </div>
            <div class="info-row">
              <span class="label">异常阈值:</span>
              <span class="value">{{ analyzer.threshold }}</span>
            </div>
            <div class="info-row">
              <span class="label">异常率:</span>
              <span class="value">{{ (analyzer.anomaly_rate * 100).toFixed(2) }}%</span>
            </div>
            <div class="info-row">
              <span class="label">总样本数:</span>
              <span class="value">{{ analyzer.total_samples }}</span>
            </div>
            <div class="info-row">
              <span class="label">准确率:</span>
              <span class="value">{{ (analyzer.accuracy * 100).toFixed(2) }}%</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 数据融合配置 -->
    <div v-if="activeTab === 'fusion'" class="content">
      <div class="toolbar">
        <button class="btn-primary" @click="showCreateFusionDialog">
          + 创建融合处理器
        </button>
      </div>

      <div class="fusion-list">
        <div
          v-for="fusion in fusionProcessors"
          :key="fusion.id"
          class="fusion-card"
        >
          <div class="card-header">
            <h3>{{ fusion.name }}</h3>
            <div class="actions">
              <button @click="editFusion(fusion.id)">编辑</button>
              <button class="danger" @click="deleteFusion(fusion.id)">删除</button>
            </div>
          </div>
          <div class="card-body">
            <div class="info-row">
              <span class="label">融合策略:</span>
              <span class="value">{{ fusion.strategy }}</span>
            </div>
            <div class="info-row">
              <span class="label">数据源权重:</span>
              <div class="weights">
                <div
                  v-for="(weight, source) in fusion.source_weights"
                  :key="source"
                  class="weight-item"
                >
                  {{ source }}: {{ weight }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 模型管理 -->
    <div v-if="activeTab === 'models'" class="content">
      <div class="toolbar">
        <button class="btn-primary" @click="showUploadModelDialog">
          + 上传模型
        </button>
      </div>

      <div class="model-list">
        <div
          v-for="model in models"
          :key="model.id"
          class="model-card"
        >
          <div class="card-header">
            <h3>{{ model.name }}</h3>
            <span class="badge">{{ model.type }}</span>
          </div>
          <div class="card-body">
            <p>{{ model.description }}</p>
            <div class="info-row">
              <span class="label">准确率:</span>
              <span class="value">{{ (model.accuracy * 100).toFixed(2) }}%</span>
            </div>
            <div class="info-row">
              <span class="label">输入维度:</span>
              <span class="value">{{ model.input_size }}</span>
            </div>
            <div class="info-row">
              <span class="label">输出维度:</span>
              <span class="value">{{ model.output_size }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 创建/编辑分析器对话框 -->
    <div v-if="showAnalyzerDialog" class="dialog-overlay" @click="closeDialogs">
      <div class="dialog" @click.stop>
        <div class="dialog-header">
          <h2>{{ editingAnalyzerId ? '编辑分析器' : '创建分析器' }}</h2>
          <button class="close-btn" @click="closeDialogs">×</button>
        </div>
        <div class="dialog-body">
          <form @submit.prevent="saveAnalyzer">
            <div class="form-group">
              <label>分析器ID *</label>
              <input
                v-model="analyzerForm.id"
                type="text"
                required
                :disabled="!!editingAnalyzerId"
              />
            </div>

            <div class="form-group">
              <label>分析器名称 *</label>
              <input v-model="analyzerForm.name" type="text" required />
            </div>

            <div class="form-group">
              <label>模型名称 *</label>
              <input v-model="analyzerForm.model_name" type="text" required />
            </div>

            <div class="form-group">
              <label>模型类型 *</label>
              <select v-model="analyzerForm.model_type" required>
                <option value="neural_network">神经网络</option>
                <option value="linear_regression">线性回归</option>
                <option value="logistic_regression">逻辑回归</option>
                <option value="decision_tree">决策树</option>
                <option value="svm">支持向量机</option>
                <option value="knn">K近邻</option>
                <option value="custom">自定义</option>
              </select>
            </div>

            <div class="form-group">
              <label>模型路径</label>
              <input v-model="analyzerForm.model_path" type="text" />
            </div>

            <div class="form-group">
              <label>异常阈值 (0-1)</label>
              <input
                v-model.number="analyzerForm.anomaly_threshold"
                type="number"
                min="0"
                max="1"
                step="0.1"
              />
            </div>

            <div class="form-group">
              <label>最低置信度 (0-1)</label>
              <input
                v-model.number="analyzerForm.confidence_min"
                type="number"
                min="0"
                max="1"
                step="0.1"
              />
            </div>

            <div class="form-group">
              <label>
                <input
                  v-model="analyzerForm.use_feature_extractor"
                  type="checkbox"
                />
                使用特征提取器
              </label>
            </div>

            <div v-if="analyzerForm.use_feature_extractor" class="form-group">
              <label>特征列表</label>
              <div class="checkbox-group">
                <label v-for="feature in availableFeatures" :key="feature">
                  <input
                    type="checkbox"
                    :value="feature"
                    v-model="analyzerForm.features"
                  />
                  {{ featureLabels[feature] }}
                </label>
              </div>
            </div>

            <div class="dialog-actions">
              <button type="button" @click="closeDialogs">取消</button>
              <button type="submit" class="btn-primary">保存</button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <!-- 创建/编辑融合处理器对话框 -->
    <div v-if="showFusionDialog" class="dialog-overlay" @click="closeDialogs">
      <div class="dialog" @click.stop>
        <div class="dialog-header">
          <h2>{{ editingFusionId ? '编辑融合处理器' : '创建融合处理器' }}</h2>
          <button class="close-btn" @click="closeDialogs">×</button>
        </div>
        <div class="dialog-body">
          <form @submit.prevent="saveFusion">
            <div class="form-group">
              <label>处理器ID *</label>
              <input
                v-model="fusionForm.id"
                type="text"
                required
                :disabled="!!editingFusionId"
              />
            </div>

            <div class="form-group">
              <label>处理器名称 *</label>
              <input v-model="fusionForm.name" type="text" required />
            </div>

            <div class="form-group">
              <label>融合策略 *</label>
              <select v-model="fusionForm.strategy" required>
                <option value="average">平均融合</option>
                <option value="weighted">加权融合</option>
                <option value="kalman">卡尔曼滤波</option>
                <option value="bayesian">贝叶斯融合</option>
                <option value="dempster_shafer">D-S证据理论</option>
                <option value="time_sync">时间同步</option>
                <option value="interpolation">插值</option>
                <option value="extrapolation">外推</option>
                <option value="moving_average">移动平均</option>
                <option value="exponential_sma">指数移动平均</option>
              </select>
            </div>

            <div class="form-group">
              <label>数据源权重</label>
              <div class="weight-editor">
                <div
                  v-for="(weight, source, index) in fusionForm.source_weights"
                  :key="index"
                  class="weight-row"
                >
                  <input
                    v-model="Object.keys(fusionForm.source_weights)[index]"
                    type="text"
                    placeholder="数据源ID"
                  />
                  <input
                    v-model.number="fusionForm.source_weights[source]"
                    type="number"
                    min="0"
                    max="1"
                    step="0.1"
                    placeholder="权重"
                  />
                  <button type="button" @click="removeWeight(source)">-</button>
                </div>
                <button type="button" @click="addWeight">+ 添加数据源</button>
              </div>
            </div>

            <div class="form-group">
              <label>时间窗口</label>
              <input v-model="fusionForm.time_window" type="text" placeholder="1s" />
            </div>

            <div class="form-group">
              <label>缓冲区大小</label>
              <input v-model.number="fusionForm.buffer_size" type="number" min="1" />
            </div>

            <div class="dialog-actions">
              <button type="button" @click="closeDialogs">取消</button>
              <button type="submit" class="btn-primary">保存</button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import api from '../services/api'

export default {
  name: 'MLConfig',
  setup() {
    const activeTab = ref('analyzers')
    const analyzers = ref([])
    const fusionProcessors = ref([])
    const models = ref([])

    const showAnalyzerDialog = ref(false)
    const showFusionDialog = ref(false)
    const editingAnalyzerId = ref(null)
    const editingFusionId = ref(null)

    const analyzerForm = ref({
      id: '',
      name: '',
      model_name: '',
      model_type: 'neural_network',
      model_path: '',
      anomaly_threshold: 0.7,
      confidence_min: 0.5,
      use_feature_extractor: true,
      features: ['numeric_mean', 'numeric_stddev', 'quality', 'confidence'],
      model_config: {}
    })

    const fusionForm = ref({
      id: '',
      name: '',
      strategy: 'weighted',
      source_weights: {},
      time_window: '1s',
      buffer_size: 100
    })

    const availableFeatures = [
      'numeric_mean',
      'numeric_stddev',
      'numeric_min',
      'numeric_max',
      'rate_of_change',
      'trend',
      'variance',
      'quality',
      'confidence',
      'source_count'
    ]

    const featureLabels = {
      'numeric_mean': '数值平均值',
      'numeric_stddev': '标准差',
      'numeric_min': '最小值',
      'numeric_max': '最大值',
      'rate_of_change': '变化率',
      'trend': '趋势',
      'variance': '方差',
      'quality': '数据质量',
      'confidence': '置信度',
      'source_count': '数据源数量'
    }

    // 加载数据
    const loadAnalyzers = async () => {
      try {
        const response = await api.get('/api/v1/ml/analyzers')
        analyzers.value = response.data.analyzers || []
      } catch (error) {
        console.error('加载分析器失败:', error)
      }
    }

    const loadFusionProcessors = async () => {
      try {
        const response = await api.get('/api/v1/ml/fusion')
        fusionProcessors.value = response.data.processors || []
      } catch (error) {
        console.error('加载融合处理器失败:', error)
      }
    }

    const loadModels = async () => {
      try {
        const response = await api.get('/api/v1/ml/models')
        models.value = response.data.models || []
      } catch (error) {
        console.error('加载模型失败:', error)
      }
    }

    // 分析器操作
    const showCreateAnalyzerDialog = () => {
      editingAnalyzerId.value = null
      analyzerForm.value = {
        id: '',
        name: '',
        model_name: '',
        model_type: 'neural_network',
        model_path: '',
        anomaly_threshold: 0.7,
        confidence_min: 0.5,
        use_feature_extractor: true,
        features: ['numeric_mean', 'numeric_stddev', 'quality', 'confidence'],
        model_config: {}
      }
      showAnalyzerDialog.value = true
    }

    const saveAnalyzer = async () => {
      try {
        if (editingAnalyzerId.value) {
          await api.put(`/api/v1/ml/analyzers/${editingAnalyzerId.value}`, analyzerForm.value)
        } else {
          await api.post('/api/v1/ml/analyzers', analyzerForm.value)
        }
        await loadAnalyzers()
        closeDialogs()
      } catch (error) {
        console.error('保存分析器失败:', error)
        alert('保存失败: ' + error.message)
      }
    }

    const deleteAnalyzer = async (id) => {
      if (!confirm('确定要删除这个分析器吗？')) return
      try {
        await api.delete(`/api/v1/ml/analyzers/${id}`)
        await loadAnalyzers()
      } catch (error) {
        console.error('删除分析器失败:', error)
      }
    }

    // 融合处理器操作
    const showCreateFusionDialog = () => {
      editingFusionId.value = null
      fusionForm.value = {
        id: '',
        name: '',
        strategy: 'weighted',
        source_weights: {},
        time_window: '1s',
        buffer_size: 100
      }
      showFusionDialog.value = true
    }

    const saveFusion = async () => {
      try {
        if (editingFusionId.value) {
          await api.put(`/api/v1/ml/fusion/${editingFusionId.value}`, fusionForm.value)
        } else {
          await api.post('/api/v1/ml/fusion', fusionForm.value)
        }
        await loadFusionProcessors()
        closeDialogs()
      } catch (error) {
        console.error('保存融合处理器失败:', error)
        alert('保存失败: ' + error.message)
      }
    }

    const deleteFusion = async (id) => {
      if (!confirm('确定要删除这个融合处理器吗？')) return
      try {
        await api.delete(`/api/v1/ml/fusion/${id}`)
        await loadFusionProcessors()
      } catch (error) {
        console.error('删除融合处理器失败:', error)
      }
    }

    const addWeight = () => {
      const newSource = `source-${Object.keys(fusionForm.value.source_weights).length + 1}`
      fusionForm.value.source_weights[newSource] = 0.5
    }

    const removeWeight = (source) => {
      delete fusionForm.value.source_weights[source]
    }

    const closeDialogs = () => {
      showAnalyzerDialog.value = false
      showFusionDialog.value = false
    }

    onMounted(() => {
      loadAnalyzers()
      loadFusionProcessors()
      loadModels()
    })

    return {
      activeTab,
      analyzers,
      fusionProcessors,
      models,
      showAnalyzerDialog,
      showFusionDialog,
      editingAnalyzerId,
      editingFusionId,
      analyzerForm,
      fusionForm,
      availableFeatures,
      featureLabels,
      showCreateAnalyzerDialog,
      showCreateFusionDialog,
      saveAnalyzer,
      saveFusion,
      deleteAnalyzer,
      deleteFusion,
      addWeight,
      removeWeight,
      closeDialogs
    }
  }
}
</script>

<style scoped>
.ml-config {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.header {
  margin-bottom: 30px;
}

.header h1 {
  font-size: 28px;
  font-weight: 600;
  margin-bottom: 8px;
}

.subtitle {
  color: #666;
  font-size: 14px;
}

.tabs {
  display: flex;
  gap: 10px;
  margin-bottom: 30px;
  border-bottom: 2px solid #eee;
}

.tab {
  padding: 12px 24px;
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: #666;
  margin-bottom: -2px;
  transition: all 0.2s;
}

.tab.active {
  color: #2563eb;
  border-bottom-color: #2563eb;
}

.toolbar {
  margin-bottom: 20px;
}

.btn-primary {
  padding: 10px 20px;
  background: #2563eb;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
}

.btn-primary:hover {
  background: #1d4ed8;
}

.analyzer-list,
.fusion-list,
.model-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 20px;
}

.analyzer-card,
.fusion-card,
.model-card {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.card-header h3 {
  font-size: 18px;
  font-weight: 600;
}

.actions {
  display: flex;
  gap: 8px;
}

.actions button {
  padding: 6px 12px;
  background: #f3f4f6;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
}

.actions button:hover {
  background: #e5e7eb;
}

.actions button.danger {
  color: #dc2626;
}

.info-row {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #f3f4f6;
}

.info-row .label {
  color: #666;
  font-size: 13px;
}

.info-row .value {
  font-weight: 500;
  font-size: 13px;
}

.badge {
  padding: 4px 12px;
  background: #dbeafe;
  color: #1e40af;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.dialog {
  background: white;
  border-radius: 12px;
  max-width: 600px;
  width: 90%;
  max-height: 90vh;
  overflow-y: auto;
}

.dialog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #e5e7eb;
}

.dialog-header h2 {
  font-size: 20px;
  font-weight: 600;
}

.close-btn {
  background: none;
  border: none;
  font-size: 28px;
  cursor: pointer;
  color: #666;
}

.dialog-body {
  padding: 20px;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: #374151;
}

.form-group input[type="text"],
.form-group input[type="number"],
.form-group select {
  width: 100%;
  padding: 10px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
}

.checkbox-group {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
}

.checkbox-group label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: normal;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
}

.dialog-actions button {
  padding: 10px 20px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.dialog-actions button:not(.btn-primary) {
  background: #f3f4f6;
}

.weight-editor {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.weight-row {
  display: grid;
  grid-template-columns: 1fr 1fr auto;
  gap: 10px;
}

.weight-row button {
  padding: 10px 16px;
  background: #fee2e2;
  color: #dc2626;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}
</style>
