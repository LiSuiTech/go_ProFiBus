import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'

export interface Alert {
  id: string
  rule_id?: string
  device_id?: string
  channel_id?: string
  event_id?: string
  level: 'info' | 'warning' | 'error' | 'critical'
  status: 'active' | 'acknowledged' | 'resolved' | 'suppressed'
  message: string
  details: Record<string, any>
  first_occurred_at: string
  last_occurred_at: string
  acknowledged_at?: string
  acknowledged_by?: string
  resolved_at?: string
  resolved_by?: string
  count: number
  created_at: string
  updated_at: string
}

export interface AlertRule {
  id: string
  name: string
  description?: string
  condition: Record<string, any>
  level: 'info' | 'warning' | 'error' | 'critical'
  enabled: boolean
  cooldown_seconds: number
  max_executions: number
  created_at: string
  updated_at: string
}

export interface AlertFilters {
  rule_id?: string
  device_id?: string
  channel_id?: string
  level?: string
  status?: string
  start?: string
  end?: string
  limit?: number
  offset?: number
}

export interface AlertStats {
  total_alerts: number
  active_alerts: number
  acknowledged_alerts: number
  resolved_alerts: number
  alerts_by_level: Record<string, number>
  alerts_by_status: Record<string, number>
}

export const alertApi = {
  // 获取告警列表
  async getAlerts(filters?: AlertFilters): Promise<{ count: number; alerts: Alert[] }> {
    const params = new URLSearchParams()
    if (filters?.rule_id) params.append('rule_id', filters.rule_id)
    if (filters?.device_id) params.append('device_id', filters.device_id)
    if (filters?.channel_id) params.append('channel_id', filters.channel_id)
    if (filters?.level) params.append('level', filters.level)
    if (filters?.status) params.append('status', filters.status)
    if (filters?.start) params.append('start', filters.start)
    if (filters?.end) params.append('end', filters.end)
    if (filters?.limit) params.append('limit', filters.limit.toString())
    if (filters?.offset) params.append('offset', filters.offset.toString())

    const response = await axios.get(`${API_BASE_URL}/alerts?${params.toString()}`)
    return response.data
  },

  // 获取告警详情
  async getAlert(id: string): Promise<Alert> {
    const response = await axios.get(`${API_BASE_URL}/alerts/${id}`)
    return response.data
  },

  // 确认告警
  async acknowledgeAlert(id: string, acknowledgedBy: string): Promise<Alert> {
    const response = await axios.post(`${API_BASE_URL}/alerts/${id}/acknowledge`, {
      acknowledged_by: acknowledgedBy,
    })
    return response.data
  },

  // 解决告警
  async resolveAlert(id: string, resolvedBy: string): Promise<Alert> {
    const response = await axios.post(`${API_BASE_URL}/alerts/${id}/resolve`, {
      resolved_by: resolvedBy,
    })
    return response.data
  },

  // 获取告警统计
  async getAlertStats(deviceId?: string, start?: string, end?: string): Promise<AlertStats> {
    const params = new URLSearchParams()
    if (deviceId) params.append('device_id', deviceId)
    if (start) params.append('start', start)
    if (end) params.append('end', end)

    const response = await axios.get(`${API_BASE_URL}/alerts/stats?${params.toString()}`)
    return response.data
  },

  // 创建告警规则
  async createAlertRule(data: {
    name: string
    description?: string
    condition: Record<string, any>
    level: string
    enabled?: boolean
    cooldown_seconds?: number
    max_executions?: number
  }): Promise<AlertRule> {
    const response = await axios.post(`${API_BASE_URL}/alerts/rules`, data)
    return response.data
  },

  // 获取告警规则列表
  async getAlertRules(enabled?: boolean): Promise<{ count: number; rules: AlertRule[] }> {
    const params = enabled !== undefined ? `?enabled=${enabled}` : ''
    const response = await axios.get(`${API_BASE_URL}/alerts/rules${params}`)
    return response.data
  },

  // 获取告警规则详情
  async getAlertRule(id: string): Promise<AlertRule> {
    const response = await axios.get(`${API_BASE_URL}/alerts/rules/${id}`)
    return response.data
  },

  // 更新告警规则
  async updateAlertRule(id: string, data: Partial<AlertRule>): Promise<AlertRule> {
    const response = await axios.put(`${API_BASE_URL}/alerts/rules/${id}`, data)
    return response.data
  },

  // 删除告警规则
  async deleteAlertRule(id: string): Promise<void> {
    await axios.delete(`${API_BASE_URL}/alerts/rules/${id}`)
  },
}
