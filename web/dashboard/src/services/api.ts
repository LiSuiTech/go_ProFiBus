import axios, { type AxiosInstance } from 'axios'
import type {
  PipelinesResponse,
  PipelineTopology,
  PipelineStatus,
  PipelineMetrics,
  TracesResponse,
  TraceFilter,
  TraceStats,
  ComponentMetrics,
} from '@/types'

class ApiClient {
  private client: AxiosInstance

  constructor() {
    const baseURL = import.meta.env.DEV
      ? 'http://localhost:8080/api/v1'
      : '/api/v1'

    this.client = axios.create({
      baseURL,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    })

    // Request interceptor
    this.client.interceptors.request.use(
      (config) => {
        return config
      },
      (error) => {
        return Promise.reject(error)
      }
    )

    // Response interceptor
    this.client.interceptors.response.use(
      (response) => {
        return response.data
      },
      (error) => {
        console.error('[API] Error:', error)
        return Promise.reject(error)
      }
    )
  }

  // ==================== Pipeline APIs ====================

  /**
   * Get all pipelines
   */
  async getPipelines(): Promise<PipelinesResponse> {
    return this.client.get('/pipelines')
  }

  /**
   * Get pipeline topology
   */
  async getPipelineTopology(pipelineId: string): Promise<PipelineTopology> {
    return this.client.get(`/pipelines/${pipelineId}/topology`)
  }

  /**
   * Get pipeline status
   */
  async getPipelineStatus(pipelineId: string): Promise<PipelineStatus> {
    return this.client.get(`/pipelines/${pipelineId}/status`)
  }

  /**
   * Start pipeline
   */
  async startPipeline(pipelineId: string): Promise<{ message: string }> {
    return this.client.post(`/pipelines/${pipelineId}/start`)
  }

  /**
   * Stop pipeline
   */
  async stopPipeline(pipelineId: string): Promise<{ message: string }> {
    return this.client.post(`/pipelines/${pipelineId}/stop`)
  }

  // ==================== Trace APIs ====================

  /**
   * Get traces with filter
   */
  async getTraces(filter?: TraceFilter): Promise<TracesResponse> {
    return this.client.get('/traces', { params: filter })
  }

  /**
   * Get trace by sample ID
   */
  async getTraceBySample(sampleId: string): Promise<{
    sample_id: string
    total_events: number
    trace_chain: any[]
    events: any[]
  }> {
    return this.client.get(`/traces/samples/${sampleId}`)
  }

  /**
   * Get trace stats
   */
  async getTraceStats(params?: {
    pipeline_id?: string
    start_time?: string
    end_time?: string
  }): Promise<{
    start_time: string
    end_time: string
    pipeline_id?: string
    stats: TraceStats
  }> {
    return this.client.get('/traces/stats', { params })
  }

  // ==================== Metrics APIs ====================

  /**
   * Get pipeline metrics
   */
  async getPipelineMetrics(
    pipelineId: string,
    params?: {
      start_time?: string
      end_time?: string
    }
  ): Promise<PipelineMetrics> {
    return this.client.get(`/pipelines/${pipelineId}/metrics`, { params })
  }

  /**
   * Get component metrics
   */
  async getComponentMetrics(params: {
    pipeline_id: string
    component_id: string
    start_time?: string
    end_time?: string
  }): Promise<{
    pipeline_id: string
    component_id: string
    component_type: string
    component_name: string
    start_time: string
    end_time: string
    metrics: Omit<ComponentMetrics, 'component_id' | 'component_type' | 'component_name'>
  }> {
    return this.client.get('/metrics/component', { params })
  }

  /**
   * Get system metrics
   */
  async getSystemMetrics(params?: {
    start_time?: string
    end_time?: string
  }): Promise<{
    start_time: string
    end_time: string
    system_metrics: {
      total_pipelines: number
      total_samples: number
      success_samples: number
      error_samples: number
      avg_duration_ms: number
      throughput_per_sec: number
    }
  }> {
    return this.client.get('/metrics/system', { params })
  }

  // ==================== Health Check ====================

  /**
   * Check API health（后端健康检查在根路径 /health，不在 /api/v1 下）
   */
  async health(): Promise<{
    status: string
    database: boolean
    time: string
  }> {
    const url = import.meta.env.DEV ? 'http://localhost:8080/health' : '/health'
    return this.client.get(url, { baseURL: '' })
  }
}

// Singleton instance
export const apiClient = new ApiClient()
