import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'

// Workflow 相关类型
export type NodeType =
  | 'data_source'
  | 'device_source'
  | 'rule_detection'
  | 'ml_analysis'
  | 'condition'
  | 'loop'
  | 'variable_set'
  | 'output'
  | 'alert_output'
  | 'device_control'
  | 'transform'
  | 'filter'

export interface Position {
  x: number
  y: number
}

export interface InputPort {
  id: string
  label: string
  type: string
  param_name: string  // 参数名称，用于节点内部引用
  data_type?: string  // 数据类型：string, number, boolean, object, array
  required?: boolean  // 是否必需
  description?: string  // 参数描述
  default_value?: any  // 默认值
}

export interface OutputPort {
  id: string
  label: string
  type: string
  param_name: string  // 参数名称，用于节点输出
  data_type?: string  // 数据类型：string, number, boolean, object, array
  description?: string  // 参数描述
}

export interface Node {
  id: string
  type: NodeType
  name: string
  config: Record<string, any>
  position: Position
  inputs: InputPort[]
  outputs: OutputPort[]
}

export interface Edge {
  id: string
  source: string
  target: string
  source_port?: string
  target_port?: string
  condition?: string
  param_mapping?: Record<string, string>  // 参数映射：目标参数名 -> 源参数名
}

export interface Variable {
  name: string
  type: string
  value: any
  description?: string
}

export interface Workflow {
  id: string
  name: string
  description?: string
  nodes: Node[]
  edges: Edge[]
  variables: Variable[]
  status: 'draft' | 'running' | 'stopped' | 'error'
  created_at: string
  updated_at: string
  created_by?: string
}

export interface CreateWorkflowRequest {
  name: string
  description?: string
  nodes?: Node[]
  edges?: Edge[]
  variables?: Variable[]
}

export interface UpdateWorkflowRequest {
  name?: string
  description?: string
  nodes?: Node[]
  edges?: Edge[]
  variables?: Variable[]
  status?: string
}

export type ExecutionStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

export interface NodeStatus {
  status: ExecutionStatus
  output?: any
  error?: string
  started_at?: string
  completed_at?: string
}

export interface WorkflowExecution {
  id: string
  workflow_id: string
  status: ExecutionStatus
  node_statuses: Record<string, NodeStatus>
  variables: Record<string, any>
  started_at?: string
  completed_at?: string
  error?: string
}

export interface ExecuteWorkflowRequest {
  inputs?: Record<string, any>
  variables?: Record<string, any>
}

export const workflowApi = {
  // 工作流 CRUD
  async createWorkflow(data: CreateWorkflowRequest): Promise<Workflow> {
    const response = await axios.post(`${API_BASE_URL}/workflows`, data)
    return response.data
  },

  async getWorkflows(): Promise<Workflow[]> {
    const response = await axios.get(`${API_BASE_URL}/workflows`)
    return response.data.workflows || response.data
  },

  async getWorkflow(id: string): Promise<Workflow> {
    const response = await axios.get(`${API_BASE_URL}/workflows/${id}`)
    return response.data
  },

  async updateWorkflow(id: string, data: UpdateWorkflowRequest): Promise<Workflow> {
    const response = await axios.put(`${API_BASE_URL}/workflows/${id}`, data)
    return response.data
  },

  async deleteWorkflow(id: string): Promise<void> {
    await axios.delete(`${API_BASE_URL}/workflows/${id}`)
  },

  // 工作流执行
  async executeWorkflow(id: string, data?: ExecuteWorkflowRequest): Promise<WorkflowExecution> {
    const response = await axios.post(`${API_BASE_URL}/workflows/${id}/execute`, data || {})
    return response.data
  },

  async getExecutions(workflowId: string): Promise<WorkflowExecution[]> {
    const response = await axios.get(`${API_BASE_URL}/workflows/${workflowId}/executions`)
    return response.data.executions || response.data
  },

  async getExecution(workflowId: string, executionId: string): Promise<WorkflowExecution> {
    const response = await axios.get(`${API_BASE_URL}/workflows/${workflowId}/executions/${executionId}`)
    return response.data
  },

  async cancelExecution(workflowId: string, executionId: string): Promise<void> {
    await axios.post(`${API_BASE_URL}/workflows/${workflowId}/executions/${executionId}/cancel`)
  },

  // 工作流模板
  async getTemplates(params?: { category?: string; tag?: string; limit?: number; offset?: number }): Promise<{ count: number; templates: WorkflowTemplate[] }> {
    const response = await axios.get(`${API_BASE_URL}/workflows/templates`, { params })
    return response.data
  },

  async getTemplate(id: string): Promise<WorkflowTemplate> {
    const response = await axios.get(`${API_BASE_URL}/workflows/templates/${id}`)
    return response.data
  },

  async createWorkflowFromTemplate(templateId: string, data: CreateWorkflowFromTemplateRequest): Promise<Workflow> {
    const response = await axios.post(`${API_BASE_URL}/workflows/templates/${templateId}/create`, data)
    return response.data
  },
}

export interface WorkflowTemplate {
  id: string
  name: string
  description?: string
  category: string
  tags: string[]
  icon?: string
  thumbnail_url?: string
  workflow_data: {
    nodes: Node[]
    edges: Edge[]
    variables: Variable[]
  }
  variables_config: Record<string, {
    type: string
    description: string
    required?: boolean
    default?: any
  }>
  usage_count: number
  rating: number
  enabled: boolean
  created_at: string
  updated_at: string
  created_by?: string
  metadata: Record<string, any>
}

export interface CreateWorkflowFromTemplateRequest {
  name: string
  description?: string
  variables?: Record<string, any>
}
