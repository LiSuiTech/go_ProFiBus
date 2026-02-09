<template>
  <div class="workflow-editor-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>工作流编辑器</span>
          <div>
            <el-button @click="handleLoadWorkflow" :loading="loading">加载工作流</el-button>
            <el-button type="primary" @click="handleSaveWorkflow" :loading="saving">保存</el-button>
            <el-button type="success" @click="handleExecuteWorkflow" :disabled="!currentWorkflow?.id">执行</el-button>
          </div>
        </div>
      </template>

      <div class="editor-container">
        <!-- 左侧节点面板 -->
        <div class="node-panel">
          <el-card shadow="never">
            <template #header>
              <span>节点类型</span>
            </template>
            <div class="node-types">
              <div
                v-for="nodeType in nodeTypes"
                :key="nodeType.type"
                class="node-type-item"
                :draggable="true"
                @dragstart="handleDragStart($event, nodeType)"
              >
                <el-icon><component :is="nodeType.icon" /></el-icon>
                <span>{{ nodeType.label }}</span>
              </div>
            </div>
          </el-card>

          <!-- 工作流模板 -->
          <el-card shadow="never" style="margin-top: 20px">
            <template #header>
              <div style="display: flex; justify-content: space-between; align-items: center">
                <span>工作流模板</span>
                <el-button size="small" @click="loadTemplates">刷新</el-button>
              </div>
            </template>
            <el-select
              v-model="selectedCategory"
              placeholder="选择分类"
              size="small"
              style="width: 100%; margin-bottom: 10px"
              @change="loadTemplates"
            >
              <el-option label="全部" value="" />
              <el-option label="监控" value="monitoring" />
              <el-option label="控制" value="control" />
              <el-option label="数据处理" value="data_processing" />
            </el-select>
            <div v-loading="templatesLoading" style="max-height: 300px; overflow-y: auto">
              <div
                v-for="template in templates"
                :key="template.id"
                class="template-item"
                @click="handleSelectTemplate(template)"
              >
                <div class="template-header">
                  <el-icon v-if="template.icon"><component :is="template.icon" /></el-icon>
                  <span class="template-name">{{ template.name }}</span>
                </div>
                <div class="template-description">{{ template.description || '无描述' }}</div>
                <div class="template-meta">
                  <el-tag size="small" type="info">{{ template.category }}</el-tag>
                  <span class="template-usage">使用 {{ template.usage_count }} 次</span>
                </div>
              </div>
              <el-empty v-if="templates.length === 0" description="暂无模板" :image-size="60" />
            </div>
          </el-card>

          <!-- 工作流列表 -->
          <el-card shadow="never" style="margin-top: 20px">
            <template #header>
              <span>工作流列表</span>
            </template>
            <el-table :data="workflows" v-loading="loading" size="small" max-height="300">
              <el-table-column prop="name" label="名称" />
              <el-table-column prop="status" label="状态" width="80">
                <template #default="{ row }">
                  <el-tag :type="getStatusTagType(row.status)" size="small">{{ getStatusLabel(row.status) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="120">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="handleSelectWorkflow(row)">编辑</el-button>
                  <el-button link type="danger" size="small" @click="handleDeleteWorkflow(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </div>

        <!-- 中间画布区域 -->
        <div class="canvas-container" ref="canvasContainerRef">
          <div class="canvas-toolbar">
            <el-input
              v-model="workflowForm.name"
              placeholder="工作流名称"
              style="width: 200px; margin-right: 10px"
            />
            <el-input
              v-model="workflowForm.description"
              placeholder="描述"
              style="width: 200px"
            />
          </div>
          <div class="canvas" ref="canvasRef" @drop="handleDrop" @dragover.prevent>
            <svg :width="canvasWidth" :height="canvasHeight" class="workflow-svg">
              <!-- 绘制连接线 -->
              <g class="edges">
                <g v-for="edge in currentWorkflow?.edges || []" :key="edge.id">
                  <line
                    :x1="getEdgeStartX(edge)"
                    :y1="getEdgeStartY(edge)"
                    :x2="getEdgeEndX(edge)"
                    :y2="getEdgeEndY(edge)"
                    stroke="#409eff"
                    stroke-width="2"
                    marker-end="url(#arrowhead)"
                    class="edge-line"
                    @click="handleEdgeClick(edge)"
                  />
                  <!-- 显示参数映射标签 -->
                  <text
                    v-if="edge.param_mapping && Object.keys(edge.param_mapping).length > 0"
                    :x="(getEdgeStartX(edge) + getEdgeEndX(edge)) / 2"
                    :y="(getEdgeStartY(edge) + getEdgeEndY(edge)) / 2 - 5"
                    text-anchor="middle"
                    font-size="9"
                    fill="#409eff"
                    class="edge-label"
                    style="pointer-events: none;"
                  >
                    {{ Object.entries(edge.param_mapping).slice(0, 2).map(([k, v]) => `${k}←${v}`).join(', ') }}
                    <tspan v-if="Object.keys(edge.param_mapping).length > 2" dx="0" dy="10">...</tspan>
                  </text>
                </g>
              </g>
              <!-- 箭头标记 -->
              <defs>
                <marker id="arrowhead" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">
                  <polygon points="0 0, 10 3, 0 6" fill="#409eff" />
                </marker>
              </defs>
              <!-- 绘制节点 -->
              <g class="nodes">
                <g
                  v-for="node in currentWorkflow?.nodes || []"
                  :key="node.id"
                  :transform="`translate(${node.position.x}, ${node.position.y})`"
                  class="node-group"
                  @mousedown="handleNodeMouseDown($event, node)"
                >
                  <rect
                    :width="nodeWidth"
                    :height="nodeHeight"
                    :fill="getNodeColor(node.type)"
                    stroke="#333"
                    stroke-width="2"
                    rx="5"
                    class="node-rect"
                  />
                  <text
                    x="50%"
                    y="30%"
                    text-anchor="middle"
                    fill="#fff"
                    font-size="12"
                    font-weight="bold"
                  >
                    {{ getNodeTypeLabel(node.type) }}
                  </text>
                  <text
                    x="50%"
                    y="60%"
                    text-anchor="middle"
                    fill="#fff"
                    font-size="10"
                  >
                    {{ node.name }}
                  </text>
                  <!-- 输入端口 -->
                  <g v-for="(input, index) in node.inputs" :key="input.id">
                    <circle
                      :cx="0"
                      :cy="(index + 1) * 20 + 10"
                      r="6"
                      fill="#67c23a"
                      stroke="#fff"
                      stroke-width="1"
                      class="port-input"
                      @mousedown.stop="handlePortMouseDown($event, node.id, input.id, 'input')"
                    />
                    <text
                      :x="-5"
                      :y="(index + 1) * 20 + 10"
                      text-anchor="end"
                      font-size="10"
                      fill="#333"
                      class="port-label"
                    >
                      {{ input.param_name }}
                    </text>
                  </g>
                  <!-- 输出端口 -->
                  <g v-for="(output, index) in node.outputs" :key="output.id">
                    <circle
                      :cx="nodeWidth"
                      :cy="(index + 1) * 20 + 10"
                      r="6"
                      fill="#f56c6c"
                      stroke="#fff"
                      stroke-width="1"
                      class="port-output"
                      @mousedown.stop="handlePortMouseDown($event, node.id, output.id, 'output')"
                    />
                    <text
                      :x="nodeWidth + 5"
                      :y="(index + 1) * 20 + 10"
                      text-anchor="start"
                      font-size="10"
                      fill="#333"
                      class="port-label"
                    >
                      {{ output.param_name }}
                    </text>
                  </g>
                </g>
              </g>
            </svg>
          </div>
        </div>

        <!-- 右侧属性面板 -->
        <div class="property-panel">
          <el-card shadow="never">
            <template #header>
              <span>节点属性</span>
            </template>
            <div v-if="selectedNode">
              <el-tabs v-model="nodePropertyTab" type="border-card" size="small">
                <el-tab-pane label="基本信息" name="basic">
                  <el-form :model="selectedNode" label-width="100px" size="small">
                    <el-form-item label="节点名称">
                      <el-input v-model="selectedNode.name" />
                    </el-form-item>
                    <el-form-item label="节点类型">
                      <el-tag>{{ getNodeTypeLabel(selectedNode.type) }}</el-tag>
                    </el-form-item>
                    <el-form-item label="配置">
                      <el-input
                        v-model="nodeConfigText"
                        type="textarea"
                        :rows="6"
                        placeholder="JSON格式配置"
                      />
                    </el-form-item>
                    <el-form-item>
                      <el-button type="primary" size="small" @click="handleUpdateNodeConfig">更新配置</el-button>
                      <el-button size="small" @click="handleDeleteNode">删除节点</el-button>
                    </el-form-item>
                  </el-form>
                </el-tab-pane>
                
                <el-tab-pane label="输入端口" name="inputs">
                  <div style="margin-bottom: 10px">
                    <el-button size="small" type="primary" @click="handleAddInputPort">添加输入端口</el-button>
                  </div>
                  <el-table :data="selectedNode.inputs" size="small" border max-height="300">
                    <el-table-column label="标签" width="100">
                      <template #default="{ row }">
                        <el-input v-model="row.label" size="small" />
                      </template>
                    </el-table-column>
                    <el-table-column label="参数名" width="120">
                      <template #default="{ row }">
                        <el-input v-model="row.param_name" size="small" />
                      </template>
                    </el-table-column>
                    <el-table-column label="数据类型" width="100">
                      <template #default="{ row }">
                        <el-select v-model="row.data_type" size="small" style="width: 100%">
                          <el-option label="string" value="string" />
                          <el-option label="number" value="number" />
                          <el-option label="boolean" value="boolean" />
                          <el-option label="object" value="object" />
                          <el-option label="array" value="array" />
                        </el-select>
                      </template>
                    </el-table-column>
                    <el-table-column label="必需" width="60">
                      <template #default="{ row }">
                        <el-checkbox v-model="row.required" />
                      </template>
                    </el-table-column>
                    <el-table-column label="描述" min-width="120">
                      <template #default="{ row }">
                        <el-input v-model="row.description" size="small" placeholder="参数描述" />
                      </template>
                    </el-table-column>
                    <el-table-column label="操作" width="80" fixed="right">
                      <template #default="{ row, $index }">
                        <el-button link type="danger" size="small" @click="handleDeleteInputPort($index)">删除</el-button>
                      </template>
                    </el-table-column>
                  </el-table>
                </el-tab-pane>
                
                <el-tab-pane label="输出端口" name="outputs">
                  <div style="margin-bottom: 10px">
                    <el-button size="small" type="primary" @click="handleAddOutputPort">添加输出端口</el-button>
                  </div>
                  <el-table :data="selectedNode.outputs" size="small" border max-height="300">
                    <el-table-column label="标签" width="100">
                      <template #default="{ row }">
                        <el-input v-model="row.label" size="small" />
                      </template>
                    </el-table-column>
                    <el-table-column label="参数名" width="120">
                      <template #default="{ row }">
                        <el-input v-model="row.param_name" size="small" />
                      </template>
                    </el-table-column>
                    <el-table-column label="数据类型" width="100">
                      <template #default="{ row }">
                        <el-select v-model="row.data_type" size="small" style="width: 100%">
                          <el-option label="string" value="string" />
                          <el-option label="number" value="number" />
                          <el-option label="boolean" value="boolean" />
                          <el-option label="object" value="object" />
                          <el-option label="array" value="array" />
                        </el-select>
                      </template>
                    </el-table-column>
                    <el-table-column label="描述" min-width="120">
                      <template #default="{ row }">
                        <el-input v-model="row.description" size="small" placeholder="参数描述" />
                      </template>
                    </el-table-column>
                    <el-table-column label="操作" width="80" fixed="right">
                      <template #default="{ row, $index }">
                        <el-button link type="danger" size="small" @click="handleDeleteOutputPort($index)">删除</el-button>
                      </template>
                    </el-table-column>
                  </el-table>
                </el-tab-pane>
              </el-tabs>
            </div>
            <el-empty v-else description="请选择一个节点" />
          </el-card>

          <!-- 变量管理 -->
          <el-card shadow="never" style="margin-top: 20px">
            <template #header>
              <div style="display: flex; justify-content: space-between; align-items: center">
                <span>变量</span>
                <el-button size="small" @click="handleAddVariable">添加</el-button>
              </div>
            </template>
            <el-table :data="currentWorkflow?.variables || []" size="small" max-height="200">
              <el-table-column prop="name" label="名称" width="100" />
              <el-table-column prop="type" label="类型" width="80" />
              <el-table-column prop="value" label="值" />
              <el-table-column label="操作" width="80">
                <template #default="{ row, $index }">
                  <el-button link type="danger" size="small" @click="handleDeleteVariable($index)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </div>
      </div>
    </el-card>

    <!-- 连接参数映射对话框 -->
    <el-dialog v-model="edgeMappingDialogVisible" title="配置参数映射" width="700px">
      <div v-if="connectingEdge">
        <el-alert
          type="info"
          :closable="false"
          style="margin-bottom: 20px"
        >
          <template #title>
            <div>
              <div><strong>源节点:</strong> {{ getNodeName(connectingEdge.source) }}</div>
              <div><strong>目标节点:</strong> {{ getNodeName(connectingEdge.target) }}</div>
            </div>
          </template>
        </el-alert>
        
        <el-table :data="getTargetNodeInputPorts(connectingEdge.target)" border size="small" max-height="400">
          <el-table-column label="目标参数" width="200">
            <template #default="{ row }">
              <div>
                <div><strong>{{ row.label }}</strong></div>
                <div style="font-size: 12px; color: #909399;">{{ row.param_name }}</div>
                <div style="font-size: 11px; color: #c0c4cc;">{{ row.data_type || 'object' }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="映射到源参数" min-width="250">
            <template #default="{ row }">
              <el-select
                v-model="edgeParamMapping[row.param_name]"
                placeholder="选择源参数（可选）"
                style="width: 100%"
                clearable
                filterable
              >
                <el-option
                  v-for="sourcePort in getSourceNodeOutputPorts(connectingEdge.source)"
                  :key="sourcePort.id"
                  :label="`${sourcePort.label} (${sourcePort.param_name}) [${sourcePort.data_type || 'object'}]`"
                  :value="sourcePort.param_name"
                >
                  <div>
                    <div><strong>{{ sourcePort.label }}</strong></div>
                    <div style="font-size: 11px; color: #909399;">{{ sourcePort.param_name }} - {{ sourcePort.data_type || 'object' }}</div>
                  </div>
                </el-option>
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="必需" width="80" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.required" type="danger" size="small">必需</el-tag>
              <el-tag v-else type="info" size="small">可选</el-tag>
            </template>
          </el-table-column>
        </el-table>
        
        <el-alert
          v-if="getTargetNodeInputPorts(connectingEdge.target).some(p => p.required && !edgeParamMapping[p.param_name])"
          type="warning"
          :closable="false"
          style="margin-top: 15px"
        >
          <template #title>
            警告：有必需参数未映射
          </template>
        </el-alert>
      </div>
      <template #footer>
        <el-button @click="handleCancelEdgeMapping">取消</el-button>
        <el-button type="primary" @click="handleSaveEdgeMapping">保存</el-button>
      </template>
    </el-dialog>

    <!-- 新建工作流对话框 -->
    <el-dialog v-model="createDialogVisible" title="新建工作流" width="500px">
      <el-form :model="workflowForm" label-width="100px">
        <el-form-item label="工作流名称" required>
          <el-input v-model="workflowForm.name" placeholder="请输入工作流名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="workflowForm.description" type="textarea" :rows="3" placeholder="请输入描述" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreateWorkflow">创建</el-button>
      </template>
    </el-dialog>

    <!-- 从模板创建工作流对话框 -->
    <el-dialog v-model="templateDialogVisible" title="从模板创建工作流" width="600px">
      <div v-if="selectedTemplate">
        <el-alert type="info" :closable="false" style="margin-bottom: 20px">
          <template #title>
            <div>
              <div><strong>模板名称:</strong> {{ selectedTemplate.name }}</div>
              <div><strong>描述:</strong> {{ selectedTemplate.description || '无描述' }}</div>
              <div><strong>分类:</strong> {{ selectedTemplate.category }}</div>
            </div>
          </template>
        </el-alert>

        <el-form label-width="120px">
          <el-form-item label="工作流名称" required>
            <el-input v-model="workflowForm.name" placeholder="请输入工作流名称" />
          </el-form-item>
          <el-form-item label="描述">
            <el-input v-model="workflowForm.description" type="textarea" :rows="2" placeholder="请输入描述（可选）" />
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
              v-if="(config as any).type === 'string'"
              v-model="templateVariables[key]"
              :placeholder="`请输入${(config as any).description || key}`"
            />
            <el-input-number
              v-else-if="(config as any).type === 'number'"
              v-model="templateVariables[key]"
              style="width: 100%"
            />
            <el-switch
              v-else-if="(config as any).type === 'boolean'"
              v-model="templateVariables[key]"
            />
            <el-input
              v-else
              v-model="templateVariables[key]"
              type="textarea"
              :rows="2"
              :placeholder="`请输入${(config as any).description || key} (JSON格式)`"
            />
            <div v-if="(config as any).description" style="font-size: 12px; color: #909399; margin-top: 5px">
              {{ (config as any).description }}
            </div>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="templateDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreateFromTemplate">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Monitor,
  Connection,
  Bell,
  SwitchButton,
  DataAnalysis,
  Filter,
  Setting,
  Refresh,
  Document,
  Operation,
} from '@element-plus/icons-vue'
import { workflowApi, type Workflow, type Node, type NodeType, type Edge, type Variable, type WorkflowTemplate } from '../services/workflowApi'

const loading = ref(false)
const saving = ref(false)
const workflows = ref<Workflow[]>([])
const currentWorkflow = ref<Workflow | null>(null)
const selectedNode = ref<Node | null>(null)
const nodeConfigText = ref('{}')
const nodePropertyTab = ref('basic')
const createDialogVisible = ref(false)
const edgeMappingDialogVisible = ref(false)
const connectingEdge = ref<Edge | null>(null)
const edgeParamMapping = ref<Record<string, string>>({})

// 模板相关
const templates = ref<WorkflowTemplate[]>([])
const templatesLoading = ref(false)
const selectedCategory = ref('')
const templateDialogVisible = ref(false)
const selectedTemplate = ref<WorkflowTemplate | null>(null)
const templateVariables = ref<Record<string, any>>({})

const canvasContainerRef = ref<HTMLElement>()
const canvasRef = ref<HTMLElement>()
const canvasWidth = ref(1200)
const canvasHeight = ref(800)
const nodeWidth = 120
const nodeHeight = 80

const workflowForm = reactive({
  name: '',
  description: '',
})

// 节点类型定义
const nodeTypes = [
  { type: 'data_source' as NodeType, label: '数据源', icon: Monitor },
  { type: 'device_source' as NodeType, label: '设备数据源', icon: Connection },
  { type: 'rule_detection' as NodeType, label: '规则检测', icon: Document },
  { type: 'ml_analysis' as NodeType, label: 'ML分析', icon: DataAnalysis },
  { type: 'condition' as NodeType, label: '条件分支', icon: Operation },
  { type: 'transform' as NodeType, label: '数据转换', icon: Refresh },
  { type: 'filter' as NodeType, label: '数据过滤', icon: Filter },
  { type: 'alert_output' as NodeType, label: '告警输出', icon: Bell },
  { type: 'device_control' as NodeType, label: '设备控制', icon: SwitchButton },
  { type: 'output' as NodeType, label: '输出', icon: Setting },
]

// 拖拽相关
let draggedNodeType: NodeType | null = null
let connectingFrom: { nodeId: string; portId: string } | null = null

// 加载工作流列表
const loadWorkflows = async () => {
  loading.value = true
  try {
    workflows.value = await workflowApi.getWorkflows()
  } catch (error: any) {
    ElMessage.error('加载工作流列表失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const handleLoadWorkflow = () => {
  createDialogVisible.value = true
}

const handleCreateWorkflow = async () => {
  if (!workflowForm.name) {
    ElMessage.warning('请输入工作流名称')
    return
  }

  try {
    const workflow = await workflowApi.createWorkflow({
      name: workflowForm.name,
      description: workflowForm.description,
      nodes: [],
      edges: [],
      variables: [],
    })
    ElMessage.success('创建工作流成功')
    createDialogVisible.value = false
    currentWorkflow.value = workflow
    await loadWorkflows()
  } catch (error: any) {
    ElMessage.error('创建工作流失败: ' + (error.message || '未知错误'))
  }
}

const handleSelectWorkflow = async (workflow: Workflow) => {
  try {
    const fullWorkflow = await workflowApi.getWorkflow(workflow.id)
    currentWorkflow.value = fullWorkflow
    workflowForm.name = fullWorkflow.name
    workflowForm.description = fullWorkflow.description || ''
  } catch (error: any) {
    ElMessage.error('加载工作流失败: ' + (error.message || '未知错误'))
  }
}

const handleDeleteWorkflow = async (workflow: Workflow) => {
  try {
    await ElMessageBox.confirm('确定要删除该工作流吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await workflowApi.deleteWorkflow(workflow.id)
    ElMessage.success('删除成功')
    if (currentWorkflow.value?.id === workflow.id) {
      currentWorkflow.value = null
    }
    await loadWorkflows()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + (error.message || '未知错误'))
    }
  }
}

const handleSaveWorkflow = async () => {
  if (!currentWorkflow.value) {
    ElMessage.warning('请先创建或选择一个工作流')
    return
  }

  saving.value = true
  try {
    await workflowApi.updateWorkflow(currentWorkflow.value.id, {
      name: workflowForm.name,
      description: workflowForm.description,
      nodes: currentWorkflow.value.nodes,
      edges: currentWorkflow.value.edges,
      variables: currentWorkflow.value.variables,
    })
    ElMessage.success('保存成功')
    await loadWorkflows()
  } catch (error: any) {
    ElMessage.error('保存失败: ' + (error.message || '未知错误'))
  } finally {
    saving.value = false
  }
}

const handleExecuteWorkflow = async () => {
  if (!currentWorkflow.value) {
    ElMessage.warning('请先创建或选择一个工作流')
    return
  }

  try {
    const execution = await workflowApi.executeWorkflow(currentWorkflow.value.id)
    ElMessage.success('工作流执行已启动')
    console.log('Execution:', execution)
  } catch (error: any) {
    ElMessage.error('执行失败: ' + (error.message || '未知错误'))
  }
}

// 拖拽处理
const handleDragStart = (event: DragEvent, nodeType: { type: NodeType; label: string }) => {
  draggedNodeType = nodeType.type
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'copy'
  }
}

const handleDrop = (event: DragEvent) => {
  event.preventDefault()
  if (!draggedNodeType || !currentWorkflow.value) {
    ElMessage.warning('请先创建或选择一个工作流')
    return
  }

  if (!canvasRef.value) return

  const rect = canvasRef.value.getBoundingClientRect()
  const x = event.clientX - rect.left
  const y = event.clientY - rect.top

  const newNode: Node = {
    id: `node_${Date.now()}`,
    type: draggedNodeType,
    name: getNodeTypeLabel(draggedNodeType),
    config: getDefaultConfig(draggedNodeType),
    position: { x, y },
    inputs: getDefaultInputs(draggedNodeType),
    outputs: getDefaultOutputs(draggedNodeType),
  }

  currentWorkflow.value.nodes.push(newNode)
  selectedNode.value = newNode
  nodeConfigText.value = JSON.stringify(newNode.config, null, 2)
  draggedNodeType = null
}

const handleNodeMouseDown = (event: MouseEvent, node: Node) => {
  selectedNode.value = node
  nodeConfigText.value = JSON.stringify(node.config, null, 2)
}

const handlePortMouseDown = (event: MouseEvent, nodeId: string, portId: string, type: 'input' | 'output') => {
  event.stopPropagation()
  if (type === 'output') {
    connectingFrom = { nodeId, portId }
  } else if (type === 'input' && connectingFrom) {
    // 创建连接
    if (!currentWorkflow.value) return

    const sourceNode = currentWorkflow.value.nodes.find(n => n.id === connectingFrom!.nodeId)
    const targetNode = currentWorkflow.value.nodes.find(n => n.id === nodeId)
    
    if (!sourceNode || !targetNode) {
      connectingFrom = null
      return
    }

    const sourcePort = sourceNode.outputs.find(p => p.id === connectingFrom!.portId)
    const targetPort = targetNode.inputs.find(p => p.id === portId)

    const newEdge: Edge = {
      id: `edge_${Date.now()}`,
      source: connectingFrom.nodeId,
      target: nodeId,
      source_port: connectingFrom.portId,
      target_port: portId,
      param_mapping: {},
    }

    // 初始化参数映射：如果端口有参数名，自动映射
    const initialMapping: Record<string, string> = {}
    if (sourcePort && targetPort && sourcePort.param_name && targetPort.param_name) {
      initialMapping[targetPort.param_name] = sourcePort.param_name
    }
    edgeParamMapping.value = initialMapping

    // 打开参数映射配置对话框
    connectingEdge.value = newEdge
    edgeMappingDialogVisible.value = true
    connectingFrom = null
  }
}

const handleSaveEdgeMapping = () => {
  if (!connectingEdge.value || !currentWorkflow.value) return

  // 过滤掉空值
  const mapping: Record<string, string> = {}
  for (const [key, value] of Object.entries(edgeParamMapping.value)) {
    if (value) {
      mapping[key] = value
    }
  }

  connectingEdge.value.param_mapping = Object.keys(mapping).length > 0 ? mapping : undefined
  
  // 检查连接是否已存在
  const existingIndex = currentWorkflow.value.edges.findIndex(e => e.id === connectingEdge.value!.id)
  if (existingIndex >= 0) {
    // 更新现有连接
    currentWorkflow.value.edges[existingIndex] = connectingEdge.value
  } else {
    // 添加新连接
    currentWorkflow.value.edges.push(connectingEdge.value)
  }
  
  edgeMappingDialogVisible.value = false
  connectingEdge.value = null
  edgeParamMapping.value = {}
}

const getSourceNodeOutputPorts = (nodeId: string) => {
  if (!currentWorkflow.value) return []
  const node = currentWorkflow.value.nodes.find(n => n.id === nodeId)
  return node?.outputs || []
}

const getTargetNodeInputPorts = (nodeId: string) => {
  if (!currentWorkflow.value) return []
  const node = currentWorkflow.value.nodes.find(n => n.id === nodeId)
  return node?.inputs || []
}

const getNodeName = (nodeId: string) => {
  if (!currentWorkflow.value) return ''
  const node = currentWorkflow.value.nodes.find(n => n.id === nodeId)
  return node?.name || nodeId
}

const handleUpdateNodeConfig = () => {
  if (!selectedNode.value || !currentWorkflow.value) return

  try {
    selectedNode.value.config = JSON.parse(nodeConfigText.value)
    ElMessage.success('配置已更新')
  } catch {
    ElMessage.error('JSON格式错误')
  }
}

const handleDeleteNode = () => {
  if (!selectedNode.value || !currentWorkflow.value) return

  currentWorkflow.value.nodes = currentWorkflow.value.nodes.filter(n => n.id !== selectedNode.value!.id)
  currentWorkflow.value.edges = currentWorkflow.value.edges.filter(
    e => e.source !== selectedNode.value!.id && e.target !== selectedNode.value!.id
  )
  selectedNode.value = null
  ElMessage.success('节点已删除')
}

const handleAddVariable = () => {
  if (!currentWorkflow.value) return

  const newVar: Variable = {
    name: `var_${Date.now()}`,
    type: 'string',
    value: '',
    description: '',
  }
  currentWorkflow.value.variables.push(newVar)
}

const handleDeleteVariable = (index: number) => {
  if (!currentWorkflow.value) return
  currentWorkflow.value.variables.splice(index, 1)
}

// 端口管理
const handleAddInputPort = () => {
  if (!selectedNode.value) return
  const newPort = {
    id: `input_${Date.now()}`,
    label: `输入${selectedNode.value.inputs.length + 1}`,
    type: 'data',
    param_name: `input_${selectedNode.value.inputs.length + 1}`,
    data_type: 'object',
    required: false,
    description: '',
  }
  selectedNode.value.inputs.push(newPort)
}

const handleDeleteInputPort = (index: number) => {
  if (!selectedNode.value) return
  const port = selectedNode.value.inputs[index]
  selectedNode.value.inputs.splice(index, 1)
  
  // 删除相关的连接
  if (currentWorkflow.value) {
    currentWorkflow.value.edges = currentWorkflow.value.edges.filter(
      e => !(e.target === selectedNode.value!.id && e.target_port === port.id)
    )
  }
}

const handleAddOutputPort = () => {
  if (!selectedNode.value) return
  const newPort = {
    id: `output_${Date.now()}`,
    label: `输出${selectedNode.value.outputs.length + 1}`,
    type: 'data',
    param_name: `output_${selectedNode.value.outputs.length + 1}`,
    data_type: 'object',
    description: '',
  }
  selectedNode.value.outputs.push(newPort)
}

const handleDeleteOutputPort = (index: number) => {
  if (!selectedNode.value) return
  const port = selectedNode.value.outputs[index]
  selectedNode.value.outputs.splice(index, 1)
  
  // 删除相关的连接
  if (currentWorkflow.value) {
    currentWorkflow.value.edges = currentWorkflow.value.edges.filter(
      e => !(e.source === selectedNode.value!.id && e.source_port === port.id)
    )
  }
}

// 工具方法
const getNodeTypeLabel = (type: NodeType) => {
  const nodeType = nodeTypes.find(nt => nt.type === type)
  return nodeType?.label || type
}

const getNodeColor = (type: NodeType) => {
  const colors: Record<NodeType, string> = {
    data_source: '#409eff',
    device_source: '#67c23a',
    rule_detection: '#e6a23c',
    ml_analysis: '#f56c6c',
    condition: '#909399',
    loop: '#909399',
    variable_set: '#909399',
    output: '#409eff',
    alert_output: '#f56c6c',
    device_control: '#e6a23c',
    transform: '#67c23a',
    filter: '#909399',
  }
  return colors[type] || '#909399'
}

const getDefaultConfig = (type: NodeType): Record<string, any> => {
  const configs: Record<NodeType, Record<string, any>> = {
    device_source: { device_id: '' },
    alert_output: { level: 'warning', message: '' },
    device_control: { device_id: '', action: 'emergency_stop', parameters: {} },
    data_source: {},
    rule_detection: {},
    ml_analysis: {},
    condition: { condition: '' },
    loop: {},
    variable_set: { variable_name: '', value: '' },
    output: {},
    transform: {},
    filter: {},
  }
  return configs[type] || {}
}

const getDefaultInputs = (type: NodeType): any[] => {
  if (type === 'device_source' || type === 'data_source') {
    return []
  }
  
  // 根据节点类型返回默认输入端口
  const defaultInputs: Record<NodeType, any[]> = {
    device_source: [],
    data_source: [],
    rule_detection: [
      { id: 'input_1', label: '数据', type: 'data', param_name: 'data', data_type: 'object', required: true, description: '输入数据' },
    ],
    ml_analysis: [
      { id: 'input_1', label: '数据', type: 'data', param_name: 'data', data_type: 'object', required: true, description: '输入数据' },
    ],
    condition: [
      { id: 'input_1', label: '数据', type: 'data', param_name: 'data', data_type: 'object', required: true, description: '输入数据' },
    ],
    transform: [
      { id: 'input_1', label: '数据', type: 'data', param_name: 'data', data_type: 'object', required: true, description: '输入数据' },
    ],
    filter: [
      { id: 'input_1', label: '数据', type: 'data', param_name: 'data', data_type: 'object', required: true, description: '输入数据' },
    ],
    alert_output: [
      { id: 'input_1', label: '分析结果', type: 'data', param_name: 'analysis_result', data_type: 'object', required: true, description: '分析结果' },
    ],
    device_control: [
      { id: 'input_1', label: '控制指令', type: 'data', param_name: 'control_command', data_type: 'object', required: true, description: '控制指令' },
    ],
    output: [
      { id: 'input_1', label: '数据', type: 'data', param_name: 'data', data_type: 'object', required: true, description: '输出数据' },
    ],
    loop: [
      { id: 'input_1', label: '数据', type: 'data', param_name: 'data', data_type: 'array', required: true, description: '输入数组' },
    ],
    variable_set: [
      { id: 'input_1', label: '值', type: 'data', param_name: 'value', data_type: 'object', required: false, description: '变量值' },
    ],
  }
  
  return defaultInputs[type] || [{ id: 'input_1', label: '输入', type: 'data', param_name: 'input', data_type: 'object', required: false, description: '' }]
}

const getDefaultOutputs = (type: NodeType): any[] => {
  if (type === 'output' || type === 'alert_output' || type === 'device_control') {
    return []
  }
  
  // 根据节点类型返回默认输出端口
  const defaultOutputs: Record<NodeType, any[]> = {
    device_source: [
      { id: 'output_1', label: '设备数据', type: 'data', param_name: 'device_data', data_type: 'object', description: '设备采集的数据' },
    ],
    data_source: [
      { id: 'output_1', label: '数据', type: 'data', param_name: 'data', data_type: 'object', description: '采集的数据' },
    ],
    rule_detection: [
      { id: 'output_1', label: '检测结果', type: 'data', param_name: 'detection_result', data_type: 'object', description: '规则检测结果' },
    ],
    ml_analysis: [
      { id: 'output_1', label: '分析结果', type: 'data', param_name: 'analysis_result', data_type: 'object', description: 'ML分析结果' },
    ],
    condition: [
      { id: 'output_1', label: '真分支', type: 'data', param_name: 'true_output', data_type: 'object', description: '条件为真时的输出' },
      { id: 'output_2', label: '假分支', type: 'data', param_name: 'false_output', data_type: 'object', description: '条件为假时的输出' },
    ],
    transform: [
      { id: 'output_1', label: '转换后数据', type: 'data', param_name: 'transformed_data', data_type: 'object', description: '转换后的数据' },
    ],
    filter: [
      { id: 'output_1', label: '过滤后数据', type: 'data', param_name: 'filtered_data', data_type: 'object', description: '过滤后的数据' },
    ],
    alert_output: [],
    device_control: [],
    output: [],
    loop: [
      { id: 'output_1', label: '循环结果', type: 'data', param_name: 'loop_result', data_type: 'array', description: '循环处理结果' },
    ],
    variable_set: [
      { id: 'output_1', label: '变量值', type: 'data', param_name: 'variable_value', data_type: 'object', description: '设置的变量值' },
    ],
  }
  
  return defaultOutputs[type] || [{ id: 'output_1', label: '输出', type: 'data', param_name: 'output', data_type: 'object', description: '' }]
}

const getNodePosition = (nodeId: string) => {
  const node = currentWorkflow.value?.nodes.find(n => n.id === nodeId)
  return node?.position
}

const getEdgeStartX = (edge: Edge) => {
  const node = currentWorkflow.value?.nodes.find(n => n.id === edge.source)
  if (!node) return 0
  return node.position.x + nodeWidth
}

const getEdgeStartY = (edge: Edge) => {
  const node = currentWorkflow.value?.nodes.find(n => n.id === edge.source)
  if (!node || !edge.source_port) return node.position.y + nodeHeight / 2
  
  const portIndex = node.outputs.findIndex(p => p.id === edge.source_port)
  if (portIndex >= 0) {
    return node.position.y + (portIndex + 1) * 20 + 10
  }
  return node.position.y + nodeHeight / 2
}

const getEdgeEndX = (edge: Edge) => {
  const node = currentWorkflow.value?.nodes.find(n => n.id === edge.target)
  if (!node) return 0
  return node.position.x
}

const getEdgeEndY = (edge: Edge) => {
  const node = currentWorkflow.value?.nodes.find(n => n.id === edge.target)
  if (!node || !edge.target_port) return node.position.y + nodeHeight / 2
  
  const portIndex = node.inputs.findIndex(p => p.id === edge.target_port)
  if (portIndex >= 0) {
    return node.position.y + (portIndex + 1) * 20 + 10
  }
  return node.position.y + nodeHeight / 2
}

const handleEdgeClick = (edge: Edge) => {
  connectingEdge.value = { ...edge }
  // 初始化参数映射：为所有目标输入端口创建映射项
  const targetPorts = getTargetNodeInputPorts(edge.target)
  const mapping: Record<string, string> = {}
  
  // 加载已有的映射
  if (edge.param_mapping) {
    Object.assign(mapping, edge.param_mapping)
  }
  
  // 为没有映射的端口创建空映射
  for (const port of targetPorts) {
    if (!(port.param_name in mapping)) {
      mapping[port.param_name] = ''
    }
  }
  
  edgeParamMapping.value = mapping
  edgeMappingDialogVisible.value = true
}

const handleCancelEdgeMapping = () => {
  edgeMappingDialogVisible.value = false
  connectingEdge.value = null
  edgeParamMapping.value = {}
  connectingFrom = null
}

const getStatusLabel = (status: string) => {
  const labels: Record<string, string> = {
    draft: '草稿',
    running: '运行中',
    stopped: '已停止',
    error: '错误',
  }
  return labels[status] || status
}

const getStatusTagType = (status: string) => {
  const types: Record<string, string> = {
    draft: 'info',
    running: 'success',
    stopped: '',
    error: 'danger',
  }
  return types[status] || ''
}

// 模板相关方法
const loadTemplates = async () => {
  templatesLoading.value = true
  try {
    const params: any = {}
    if (selectedCategory.value) {
      params.category = selectedCategory.value
    }
    const result = await workflowApi.getTemplates(params)
    templates.value = result.templates || []
  } catch (error: any) {
    ElMessage.error('加载模板失败: ' + (error.message || '未知错误'))
  } finally {
    templatesLoading.value = false
  }
}

const handleSelectTemplate = (template: WorkflowTemplate) => {
  selectedTemplate.value = template
  templateVariables.value = {}
  
  // 初始化模板变量
  if (template.variables_config) {
    for (const [key, config] of Object.entries(template.variables_config)) {
      const varConfig = config as any
      if (varConfig.default !== undefined) {
        templateVariables.value[key] = varConfig.default
      }
    }
  }
  
  templateDialogVisible.value = true
}

const handleCreateFromTemplate = async () => {
  if (!selectedTemplate.value) return
  
  if (!workflowForm.name) {
    ElMessage.warning('请输入工作流名称')
    return
  }

  try {
    const workflow = await workflowApi.createWorkflowFromTemplate(selectedTemplate.value.id, {
      name: workflowForm.name,
      description: workflowForm.description || selectedTemplate.value.description,
      variables: templateVariables.value,
    })
    
    ElMessage.success('从模板创建工作流成功')
    templateDialogVisible.value = false
    currentWorkflow.value = workflow
    workflowForm.name = workflow.name
    workflowForm.description = workflow.description || ''
    await loadWorkflows()
  } catch (error: any) {
    ElMessage.error('从模板创建工作流失败: ' + (error.message || '未知错误'))
  }
}

onMounted(() => {
  loadWorkflows()
  loadTemplates()
})
</script>

<style scoped>
.workflow-editor-page {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.editor-container {
  display: flex;
  gap: 20px;
  height: calc(100vh - 200px);
}

.node-panel {
  width: 250px;
  overflow-y: auto;
}

.node-types {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.node-type-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  cursor: move;
  transition: all 0.3s;
}

.node-type-item:hover {
  background-color: #f5f7fa;
  border-color: #409eff;
}

.template-item {
  padding: 12px;
  margin-bottom: 10px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.3s;
  background-color: #fff;
}

.template-item:hover {
  background-color: #f5f7fa;
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.2);
}

.template-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-weight: 500;
  color: #303133;
}

.template-name {
  flex: 1;
  font-size: 14px;
}

.template-description {
  font-size: 12px;
  color: #909399;
  margin-bottom: 8px;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.template-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
}

.template-usage {
  color: #909399;
}

.canvas-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  overflow: hidden;
}

.canvas-toolbar {
  padding: 10px;
  background-color: #f5f7fa;
  border-bottom: 1px solid #dcdfe6;
}

.canvas {
  flex: 1;
  position: relative;
  overflow: auto;
  background-color: #fafafa;
  background-image: 
    linear-gradient(rgba(0,0,0,.05) 1px, transparent 1px),
    linear-gradient(90deg, rgba(0,0,0,.05) 1px, transparent 1px);
  background-size: 20px 20px;
}

.workflow-svg {
  width: 100%;
  height: 100%;
}

.node-group {
  cursor: move;
}

.node-rect {
  transition: all 0.2s;
}

.node-group:hover .node-rect {
  stroke-width: 3;
  filter: drop-shadow(0 0 5px rgba(64, 158, 255, 0.5));
}

.port-input,
.port-output {
  cursor: crosshair;
}

.port-label {
  pointer-events: none;
  user-select: none;
}

.edge-line {
  cursor: pointer;
}

.edge-line:hover {
  stroke-width: 3;
}

.edge-label {
  pointer-events: none;
  user-select: none;
  background-color: rgba(255, 255, 255, 0.8);
  padding: 2px 4px;
  border-radius: 2px;
}

.property-panel {
  width: 300px;
  overflow-y: auto;
}
</style>
