import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'

export interface Device {
  id: string
  name: string
  description?: string
  type: 'PLC' | 'Sensor' | 'Instrument' | 'SmartDevice'
  status: 'online' | 'offline' | 'fault' | 'maintenance'
  health_score: number
  location: {
    x: number
    y: number
    z?: number
  }
  area?: string
  metadata?: Record<string, any>
  created_at: string
  updated_at: string
}

export interface DeviceFilters {
  type?: string
  status?: string
  area?: string
  limit?: number
  offset?: number
}

export interface CreateDeviceRequest {
  name: string
  description?: string
  type: string
  location?: {
    x: number
    y: number
    z?: number
  }
  area?: string
  metadata?: Record<string, any>
}

export interface UpdateDeviceRequest {
  name?: string
  description?: string
  type?: string
  location?: {
    x: number
    y: number
    z?: number
  }
  area?: string
  metadata?: Record<string, any>
}

export const deviceApi = {
  // 创建设备
  async createDevice(data: CreateDeviceRequest): Promise<Device> {
    const response = await axios.post(`${API_BASE_URL}/devices`, data)
    return response.data
  },

  // 获取设备列表
  async getDevices(filters?: DeviceFilters): Promise<{ count: number; devices: Device[] }> {
    const params = new URLSearchParams()
    if (filters?.type) params.append('type', filters.type)
    if (filters?.status) params.append('status', filters.status)
    if (filters?.area) params.append('area', filters.area)
    if (filters?.limit) params.append('limit', filters.limit.toString())
    if (filters?.offset) params.append('offset', filters.offset.toString())

    const response = await axios.get(`${API_BASE_URL}/devices?${params.toString()}`)
    return response.data
  },

  // 获取设备详情
  async getDevice(id: string): Promise<{ device: Device; channel_ids: string[] }> {
    const response = await axios.get(`${API_BASE_URL}/devices/${id}`)
    return response.data
  },

  // 更新设备
  async updateDevice(id: string, data: UpdateDeviceRequest): Promise<Device> {
    const response = await axios.put(`${API_BASE_URL}/devices/${id}`, data)
    return response.data
  },

  // 删除设备
  async deleteDevice(id: string): Promise<void> {
    await axios.delete(`${API_BASE_URL}/devices/${id}`)
  },

  // 更新设备状态
  async updateDeviceStatus(id: string, status: string): Promise<Device> {
    const response = await axios.patch(`${API_BASE_URL}/devices/${id}/status`, { status })
    return response.data
  },

  // 添加设备通道关联
  async addDeviceChannel(deviceId: string, channelId: string): Promise<void> {
    await axios.post(`${API_BASE_URL}/devices/${deviceId}/channels`, { channel_id: channelId })
  },

  // 移除设备通道关联
  async removeDeviceChannel(deviceId: string, channelId: string): Promise<void> {
    await axios.delete(`${API_BASE_URL}/devices/${deviceId}/channels/${channelId}`)
  },

  // 获取设备布局
  async getDeviceLayout(area?: string): Promise<{ devices: Device[] }> {
    const params = area ? `?area=${area}` : ''
    const response = await axios.get(`${API_BASE_URL}/devices/layout${params}`)
    return response.data
  },
}
