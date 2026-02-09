import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'

// 清洗规则相关类型
export type CleaningRuleType = 
  | 'deduplicate'
  | 'outlier_filter'
  | 'missing_fill'
  | 'normalize'
  | 'smooth'
  | 'validate'

export interface CleaningRule {
  id: string
  name: string
  description?: string
  rule_type: CleaningRuleType
  enabled: boolean
  config: Record<string, any>
  priority: number
  created_at: string
  updated_at: string
}

export interface CreateCleaningRuleRequest {
  name: string
  description?: string
  rule_type: CleaningRuleType
  config?: Record<string, any>
  priority?: number
}

// 归档策略相关类型
export interface ArchivePolicy {
  id: string
  name: string
  description?: string
  source_type: string
  source_id?: string
  retention_days: number
  archive_after_days: number
  compression_enabled: boolean
  archive_location?: string
  enabled: boolean
  last_run_at?: string
  next_run_at?: string
  run_interval_hours: number
  metadata?: Record<string, any>
  created_at: string
  updated_at: string
}

export interface CreateArchivePolicyRequest {
  name: string
  description?: string
  source_type: string
  source_id?: string
  retention_days?: number
  archive_after_days?: number
  compression_enabled?: boolean
  archive_location?: string
  run_interval_hours?: number
}

export interface ArchiveStats {
  total_records: number
  total_size: number
  archive_count: number
  last_archive_time?: string
}

// 归档记录相关类型
export type ArchiveStatus = 'pending' | 'running' | 'completed' | 'failed'

export interface ArchiveRecord {
  id: string
  policy_id: string
  source_type: string
  source_id?: string
  start_time: string
  end_time?: string
  record_count: number
  archive_size: number
  archive_path?: string
  status: ArchiveStatus
  error_message?: string
  created_at: string
  completed_at?: string
}

// 生命周期配置相关类型
export interface LifecycleConfig {
  id: string
  source_type: string
  source_id: string
  hot_storage_days: number
  warm_storage_days: number
  cold_storage_days: number
  delete_after_days?: number
  compression_after_days?: number
  created_at: string
  updated_at: string
}

export interface CreateLifecycleConfigRequest {
  source_type: string
  source_id: string
  hot_storage_days?: number
  warm_storage_days?: number
  cold_storage_days?: number
  delete_after_days?: number
  compression_after_days?: number
}

// 数据清洗请求（Dry Run：单条样例数据预览）
export interface CleanDataRequest {
  source_type: string
  source_id: string
  // 单条待清洗的数据对象
  data: Record<string, any>
}

// 数据清洗响应（返回清洗后的数据与是否发生清洗）
export interface CleanDataResponse {
  cleaned_data: Record<string, any>
  was_cleaned: boolean
}

export const dataManagementApi = {
  // 清洗规则
  async createCleaningRule(data: CreateCleaningRuleRequest): Promise<CleaningRule> {
    const response = await axios.post(`${API_BASE_URL}/data-management/cleaning-rules`, data)
    return response.data
  },

  async getCleaningRules(filters?: { type?: CleaningRuleType; enabled?: boolean; limit?: number }): Promise<{ count: number; rules: CleaningRule[] }> {
    const params = new URLSearchParams()
    if (filters?.type) params.append('type', filters.type)
    if (filters?.enabled !== undefined) params.append('enabled', filters.enabled.toString())
    if (filters?.limit) params.append('limit', filters.limit.toString())

    const response = await axios.get(`${API_BASE_URL}/data-management/cleaning-rules?${params.toString()}`)
    return response.data
  },

  // 归档策略
  async createArchivePolicy(data: CreateArchivePolicyRequest): Promise<ArchivePolicy> {
    const response = await axios.post(`${API_BASE_URL}/data-management/archive-policies`, data)
    return response.data
  },

  async getArchivePolicies(filters?: { source_type?: string; source_id?: string; enabled?: boolean; limit?: number }): Promise<{ count: number; policies: ArchivePolicy[] }> {
    const params = new URLSearchParams()
    if (filters?.source_type) params.append('source_type', filters.source_type)
    if (filters?.source_id) params.append('source_id', filters.source_id)
    if (filters?.enabled !== undefined) params.append('enabled', filters.enabled.toString())
    if (filters?.limit) params.append('limit', filters.limit.toString())

    const response = await axios.get(`${API_BASE_URL}/data-management/archive-policies?${params.toString()}`)
    return response.data
  },

  async executeArchive(policyId: string): Promise<void> {
    await axios.post(`${API_BASE_URL}/data-management/archive-policies/${policyId}/execute`)
  },

  async getArchiveStats(policyId: string, start?: string, end?: string): Promise<ArchiveStats> {
    const params = new URLSearchParams()
    if (start) params.append('start', start)
    if (end) params.append('end', end)

    const response = await axios.get(`${API_BASE_URL}/data-management/archive-policies/${policyId}/stats?${params.toString()}`)
    return response.data
  },

  // 归档记录
  async getArchiveRecords(filters?: { policy_id?: string; source_type?: string; status?: ArchiveStatus; limit?: number }): Promise<{ count: number; records: ArchiveRecord[] }> {
    const params = new URLSearchParams()
    if (filters?.policy_id) params.append('policy_id', filters.policy_id)
    if (filters?.source_type) params.append('source_type', filters.source_type)
    if (filters?.status) params.append('status', filters.status)
    if (filters?.limit) params.append('limit', filters.limit.toString())

    const response = await axios.get(`${API_BASE_URL}/data-management/archive-records?${params.toString()}`)
    return response.data
  },

  // 生命周期配置
  async createLifecycleConfig(data: CreateLifecycleConfigRequest): Promise<LifecycleConfig> {
    const response = await axios.post(`${API_BASE_URL}/data-management/lifecycle-configs`, data)
    return response.data
  },

  async getLifecycleConfig(sourceType: string, sourceId: string): Promise<LifecycleConfig> {
    const response = await axios.get(`${API_BASE_URL}/data-management/lifecycle-configs/${sourceType}/${sourceId}`)
    return response.data
  },

  // 数据清洗
  async cleanData(data: CleanDataRequest): Promise<CleanDataResponse> {
    const response = await axios.post(`${API_BASE_URL}/data-management/clean`, data)
    return response.data
  },
}
