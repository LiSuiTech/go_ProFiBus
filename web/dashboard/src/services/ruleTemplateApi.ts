import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'

export interface RuleTemplate {
  id: string
  name: string
  description?: string
  category: string
  rule_type: string
  tags: string[]
  icon?: string
  condition_template: Record<string, any>
  variables_config: Record<string, {
    type: string
    description: string
    required?: boolean
    default?: any
    enum?: string[]
  }>
  output_config: Record<string, any>
  usage_count: number
  rating: number
  enabled: boolean
  created_at: string
  updated_at: string
  created_by?: string
  metadata: Record<string, any>
}

export interface RuleTestResult {
  id: string
  rule_id?: string
  template_id?: string
  test_data: Record<string, any>
  rule_config: Record<string, any>
  test_result: Record<string, any>
  triggered: boolean
  execution_time_ms: number
  created_at: string
  created_by?: string
}

export interface CreateRuleFromTemplateRequest {
  name: string
  description?: string
  level: 'info' | 'warning' | 'error' | 'critical'
  variables: Record<string, any>
  enabled?: boolean
  cooldown_seconds?: number
  max_executions?: number
}

export interface TestTemplateRequest {
  variables: Record<string, any>
  test_data: Record<string, any>
}

export interface TestRuleRequest {
  rule_config: Record<string, any>
  test_data: Record<string, any>
}

export const ruleTemplateApi = {
  // 规则模板
  async getTemplates(params?: {
    category?: string
    rule_type?: string
    tag?: string
    limit?: number
    offset?: number
  }): Promise<{ count: number; templates: RuleTemplate[] }> {
    const response = await axios.get(`${API_BASE_URL}/rule-templates`, { params })
    return response.data
  },

  async getTemplate(id: string): Promise<RuleTemplate> {
    const response = await axios.get(`${API_BASE_URL}/rule-templates/${id}`)
    return response.data
  },

  async createRuleFromTemplate(templateId: string, data: CreateRuleFromTemplateRequest): Promise<any> {
    const response = await axios.post(`${API_BASE_URL}/rule-templates/${templateId}/create-rule`, data)
    return response.data
  },

  // 规则测试
  async testTemplate(templateId: string, data: TestTemplateRequest): Promise<{ test_result: RuleTestResult }> {
    const response = await axios.post(`${API_BASE_URL}/rule-templates/${templateId}/test`, data)
    return response.data
  },

  async testRule(data: TestRuleRequest): Promise<{ test_result: RuleTestResult }> {
    const response = await axios.post(`${API_BASE_URL}/rule-templates/test-rule`, data)
    return response.data
  },

  async getTestResults(params?: {
    rule_id?: string
    template_id?: string
    limit?: number
    offset?: number
  }): Promise<{ count: number; test_results: RuleTestResult[] }> {
    const response = await axios.get(`${API_BASE_URL}/rule-templates/test-results`, { params })
    return response.data
  },
}
