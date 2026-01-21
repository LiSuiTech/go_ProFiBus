import axios from 'axios'
import type {
  Channel,
  ChannelsResponse,
  Point,
  PointsResponse,
  ProtocolsResponse,
  CreateChannelRequest,
  UpdateChannelRequest,
  CreatePointRequest,
  UpdatePointRequest,
} from '../types'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'
const API_URL = `${API_BASE_URL}/api/v1`

// ========== Protocol APIs ==========

export const getProtocols = async (): Promise<ProtocolsResponse> => {
  const response = await axios.get(`${API_URL}/protocols`)
  return response.data
}

// ========== Channel APIs ==========

export const getChannels = async (): Promise<ChannelsResponse> => {
  const response = await axios.get(`${API_URL}/channels`)
  return response.data
}

export const getChannel = async (id: string): Promise<Channel> => {
  const response = await axios.get(`${API_URL}/channels/${id}`)
  return response.data
}

export const createChannel = async (data: CreateChannelRequest): Promise<Channel> => {
  const response = await axios.post(`${API_URL}/channels`, data)
  return response.data
}

export const updateChannel = async (
  id: string,
  data: UpdateChannelRequest
): Promise<Channel> => {
  const response = await axios.put(`${API_URL}/channels/${id}`, data)
  return response.data
}

export const deleteChannel = async (id: string): Promise<void> => {
  await axios.delete(`${API_URL}/channels/${id}`)
}

export const startChannel = async (id: string): Promise<void> => {
  await axios.post(`${API_URL}/channels/${id}/start`)
}

export const stopChannel = async (id: string): Promise<void> => {
  await axios.post(`${API_URL}/channels/${id}/stop`)
}

// ========== Point APIs ==========

export const getChannelPoints = async (channelId: string): Promise<PointsResponse> => {
  const response = await axios.get(`${API_URL}/channels/${channelId}/points`)
  return response.data
}

export const createPoint = async (
  channelId: string,
  data: CreatePointRequest
): Promise<Point> => {
  const response = await axios.post(`${API_URL}/channels/${channelId}/points`, data)
  return response.data
}

export const updatePoint = async (
  channelId: string,
  pointId: string,
  data: UpdatePointRequest
): Promise<Point> => {
  const response = await axios.put(
    `${API_URL}/channels/${channelId}/points/${pointId}`,
    data
  )
  return response.data
}

export const deletePoint = async (channelId: string, pointId: string): Promise<void> => {
  await axios.delete(`${API_URL}/channels/${channelId}/points/${pointId}`)
}
