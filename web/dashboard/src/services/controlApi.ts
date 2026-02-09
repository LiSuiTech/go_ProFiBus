import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'

// 控制策略相关类型
export interface ControlPolicy {
  id: string
  name: string
  description?: string
  enabled: boolean
  priority: number
  condition_config: Record<string, any>
  action_config: Record<string, any>
  cooldown_seconds: number
  max_executions: number
  execution_count: number
  last_executed_at?: string
  metadata?: Record<string, any>
  created_at: string
  updated_at: string
}

export interface CreateControlPolicyRequest {
  name: string
  description?: string
  enabled?: boolean
  priority?: number
  condition_config?: Record<string, any>
  action_config: Record<string, any>
  cooldown_seconds?: number
  max_executions?: number
  metadata?: Record<string, any>
}

export interface UpdateControlPolicyRequest {
  name?: string
  description?: string
  enabled?: boolean
  priority?: number
  condition_config?: Record<string, any>
  action_config?: Record<string, any>
  cooldown_seconds?: number
  max_executions?: number
  metadata?: Record<string, any>
}

// 控制动作相关类型
export type ActionType = 
  | 'emergency_stop'
  | 'shutdown'
  | 'start'
  | 'pause'
  | 'resume'
  | 'set_value'
  | 'call_method'
  | 'send_command'
  | 'custom'

export type ActionStatus = 'pending' | 'executing' | 'completed' | 'failed' | 'cancelled'

export interface ControlAction {
  id: string
  policy_id?: string
  device_id: string
  action_type: ActionType
  parameters?: Record<string, any>
  reason?: string
  severity: number
  status: ActionStatus
  result?: Record<string, any>
  error_message?: string
  executed_by?: string
  executed_at?: string
  completed_at?: string
  duration_ms?: number
  require_confirmation: boolean
  confirmed_by?: string
  confirmed_at?: string
  metadata?: Record<string, any>
  created_at: string
}

export interface CreateControlActionRequest {
  policy_id?: string
  device_id: string
  action_type: ActionType
  parameters?: Record<string, any>
  reason?: string
  severity?: number
  require_confirmation?: boolean
  metadata?: Record<string, any>
}

export interface ControlActionFilters {
  policy_id?: string
  device_id?: string
  action_type?: ActionType
  status?: ActionStatus
  executed_by?: string
  start_time?: string
  end_time?: string
  limit?: number
  offset?: number
}

// 审计日志相关类型
export type AuditEventType = 'created' | 'confirmed' | 'executed' | 'completed' | 'failed' | 'cancelled'

export interface AuditLog {
  id: string
  action_id: string
  event_type: AuditEventType
  user_id?: string
  user_name?: string
  details: Record<string, any>
  ip_address?: string
  user_agent?: string
  created_at: string
}

export interface AuditLogFilters {
  action_id?: string
  user_id?: string
  event_type?: AuditEventType
  start_time?: string
  end_time?: string
  limit?: number
  offset?: number
}

// 控制权限相关类型
export interface TimeRange {
  start: string
  end: string
}

export interface ControlPermission {
  id: string
  user_id: string
  action_type: ActionType
  target_devices: string[]
  max_severity: number
  require_confirmation: boolean
  allowed_time_ranges: TimeRange[]
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateControlPermissionRequest {
  user_id: string
  action_type: ActionType
  enabled?: boolean
  target_devices?: string[]
  max_severity?: number
  require_confirmation?: boolean
  allowed_time_ranges?: TimeRange[]
}

export interface UpdateControlPermissionRequest {
  enabled?: boolean
  target_devices?: string[]
  max_severity?: number
  require_confirmation?: boolean
  allowed_time_ranges?: TimeRange[]
}

export interface ControlPermissionFilters {
  user_id?: string
  action_type?: ActionType
  enabled?: boolean
  limit?: number
  offset?: number
}

export const controlApi = {
  // 控制策略
  async createPolicy(data: CreateControlPolicyRequest): Promise<ControlPolicy> {
    const response = await axios.post(`${API_BASE_URL}/control/policies`, data)
    return response.data
  },

  async getPolicies(filters?: { enabled?: boolean; limit?: number; offset?: number }): Promise<{ policies: ControlPolicy[]; total: number }> {
    const params = new URLSearchParams()
    if (filters?.enabled !== undefined) params.append('enabled', filters.enabled.toString())
    if (filters?.limit) params.append('limit', filters.limit.toString())
    if (filters?.offset) params.append('offset', filters.offset.toString())

    const response = await axios.get(`${API_BASE_URL}/control/policies?${params.toString()}`)
    return response.data
  },

  async getPolicy(id: string): Promise<ControlPolicy> {
    const response = await axios.get(`${API_BASE_URL}/control/policies/${id}`)
    return response.data
  },

  async updatePolicy(id: string, data: UpdateControlPolicyRequest): Promise<ControlPolicy> {
    const response = await axios.put(`${API_BASE_URL}/control/policies/${id}`, data)
    return response.data
  },

  async deletePolicy(id: string): Promise<void> {
    await axios.delete(`${API_BASE_URL}/control/policies/${id}`)
  },

  // 控制动作
  async createAction(data: CreateControlActionRequest): Promise<ControlAction> {
    const response = await axios.post(`${API_BASE_URL}/control/actions`, data)
    return response.data
  },

  async getActions(filters?: ControlActionFilters): Promise<{ actions: ControlAction[]; total: number }> {
    const params = new URLSearchParams()
    if (filters?.policy_id) params.append('policy_id', filters.policy_id)
    if (filters?.device_id) params.append('device_id', filters.device_id)
    if (filters?.action_type) params.append('action_type', filters.action_type)
    if (filters?.status) params.append('status', filters.status)
    if (filters?.executed_by) params.append('executed_by', filters.executed_by)
    if (filters?.start_time) params.append('start_time', filters.start_time)
    if (filters?.end_time) params.append('end_time', filters.end_time)
    if (filters?.limit) params.append('limit', filters.limit.toString())
    if (filters?.offset) params.append('offset', filters.offset.toString())

    const response = await axios.get(`${API_BASE_URL}/control/actions?${params.toString()}`)
    return response.data
  },

  async getAction(id: string): Promise<ControlAction> {
    const response = await axios.get(`${API_BASE_URL}/control/actions/${id}`)
    return response.data
  },

  async confirmAction(id: string): Promise<ControlAction> {
    const response = await axios.post(`${API_BASE_URL}/control/actions/${id}/confirm`)
    return response.data
  },

  // 审计日志
  async getAuditLogs(filters?: AuditLogFilters): Promise<{ logs: AuditLog[]; total: number }> {
    const params = new URLSearchParams()
    if (filters?.action_id) params.append('action_id', filters.action_id)
    if (filters?.user_id) params.append('user_id', filters.user_id)
    if (filters?.event_type) params.append('event_type', filters.event_type)
    if (filters?.start_time) params.append('start_time', filters.start_time)
    if (filters?.end_time) params.append('end_time', filters.end_time)
    if (filters?.limit) params.append('limit', filters.limit.toString())
    if (filters?.offset) params.append('offset', filters.offset.toString())

    const response = await axios.get(`${API_BASE_URL}/control/audit-logs?${params.toString()}`)
    return response.data
  },

  // 控制权限
  async createPermission(data: CreateControlPermissionRequest): Promise<ControlPermission> {
    const response = await axios.post(`${API_BASE_URL}/control/permissions`, data)
    return response.data
  },

  async getPermissions(filters?: ControlPermissionFilters): Promise<{ permissions: ControlPermission[]; total: number }> {
    const params = new URLSearchParams()
    if (filters?.user_id) params.append('user_id', filters.user_id)
    if (filters?.action_type) params.append('action_type', filters.action_type)
    if (filters?.enabled !== undefined) params.append('enabled', filters.enabled.toString())
    if (filters?.limit) params.append('limit', filters.limit.toString())
    if (filters?.offset) params.append('offset', filters.offset.toString())

    const response = await axios.get(`${API_BASE_URL}/control/permissions?${params.toString()}`)
    return response.data
  },

  async updatePermission(id: string, data: UpdateControlPermissionRequest): Promise<ControlPermission> {
    const response = await axios.put(`${API_BASE_URL}/control/permissions/${id}`, data)
    return response.data
  },

  async deletePermission(id: string): Promise<void> {
    await axios.delete(`${API_BASE_URL}/control/permissions/${id}`)
  },
}
