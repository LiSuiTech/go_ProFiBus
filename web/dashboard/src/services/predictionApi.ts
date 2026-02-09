import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'

export interface Prediction {
  id: string
  model_id: string
  device_id?: string
  channel_id?: string
  prediction_type: 'forecast' | 'anomaly' | 'trend' | 'performance'
  field_name?: string
  predicted_value: number
  confidence: number
  actual_value?: number
  error_rate?: number
  time_range_start: string
  time_range_end: string
  metadata?: Record<string, any>
  created_at: string
}

export interface PredictionModel {
  id: string
  name: string
  description?: string
  type: 'linear_regression' | 'neural_network' | 'svm' | 'decision_tree' | 'lstm' | 'custom'
  version: string
  file_path?: string
  status: 'draft' | 'training' | 'deployed' | 'archived'
  accuracy?: number
  training_samples: number
  metadata?: Record<string, any>
  created_at: string
  updated_at: string
  deployed_at?: string
}

export interface PredictionFilters {
  model_id?: string
  device_id?: string
  channel_id?: string
  type?: string
  start?: string
  end?: string
  limit?: number
  offset?: number
}

export interface ModelFilters {
  type?: string
  status?: string
  limit?: number
  offset?: number
}

export const predictionApi = {
  // 创建预测结果
  async createPrediction(data: {
    model_id: string
    device_id?: string
    channel_id?: string
    prediction_type: string
    field_name?: string
    predicted_value: number
    confidence?: number
    time_range_start: string
    time_range_end: string
    metadata?: Record<string, any>
  }): Promise<Prediction> {
    const response = await axios.post(`${API_BASE_URL}/predictions`, data)
    return response.data
  },

  // 获取预测结果列表
  async getPredictions(filters?: PredictionFilters): Promise<{ count: number; predictions: Prediction[] }> {
    const params = new URLSearchParams()
    if (filters?.model_id) params.append('model_id', filters.model_id)
    if (filters?.device_id) params.append('device_id', filters.device_id)
    if (filters?.channel_id) params.append('channel_id', filters.channel_id)
    if (filters?.type) params.append('type', filters.type)
    if (filters?.start) params.append('start', filters.start)
    if (filters?.end) params.append('end', filters.end)
    if (filters?.limit) params.append('limit', filters.limit.toString())
    if (filters?.offset) params.append('offset', filters.offset.toString())

    const response = await axios.get(`${API_BASE_URL}/predictions?${params.toString()}`)
    return response.data
  },

  // 获取预测结果详情
  async getPrediction(id: string): Promise<Prediction> {
    const response = await axios.get(`${API_BASE_URL}/predictions/${id}`)
    return response.data
  },

  // 趋势预测
  async forecast(data: {
    model_id: string
    device_id: string
    field_name: string
    time_range_end: string
    forecast_steps?: number
  }): Promise<{ predictions: Prediction[]; message: string }> {
    const response = await axios.post(`${API_BASE_URL}/predictions/forecast`, data)
    return response.data
  },

  // 获取预测历史
  async getPredictionHistory(deviceId: string, type?: string, limit?: number): Promise<{ count: number; predictions: Prediction[] }> {
    const params = new URLSearchParams()
    params.append('device_id', deviceId)
    if (type) params.append('type', type)
    if (limit) params.append('limit', limit.toString())

    const response = await axios.get(`${API_BASE_URL}/predictions/history?${params.toString()}`)
    return response.data
  },

  // 创建预测模型
  async createModel(data: {
    name: string
    description?: string
    type: string
    version?: string
    file_path?: string
  }): Promise<PredictionModel> {
    const response = await axios.post(`${API_BASE_URL}/predictions/models`, data)
    return response.data
  },

  // 获取预测模型列表
  async getModels(filters?: ModelFilters): Promise<{ count: number; models: PredictionModel[] }> {
    const params = new URLSearchParams()
    if (filters?.type) params.append('type', filters.type)
    if (filters?.status) params.append('status', filters.status)
    if (filters?.limit) params.append('limit', filters.limit.toString())
    if (filters?.offset) params.append('offset', filters.offset.toString())

    const response = await axios.get(`${API_BASE_URL}/predictions/models?${params.toString()}`)
    return response.data
  },

  // 获取预测模型详情
  async getModel(id: string): Promise<PredictionModel> {
    const response = await axios.get(`${API_BASE_URL}/predictions/models/${id}`)
    return response.data
  },

  // 更新预测模型
  async updateModel(id: string, data: Partial<PredictionModel>): Promise<PredictionModel> {
    const response = await axios.put(`${API_BASE_URL}/predictions/models/${id}`, data)
    return response.data
  },

  // 部署模型
  async deployModel(id: string): Promise<PredictionModel> {
    const response = await axios.post(`${API_BASE_URL}/predictions/models/${id}/deploy`)
    return response.data
  },

  // 删除预测模型
  async deleteModel(id: string): Promise<void> {
    await axios.delete(`${API_BASE_URL}/predictions/models/${id}`)
  },
}
