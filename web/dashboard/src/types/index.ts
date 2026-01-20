// Pipeline Types
export interface Pipeline {
  id: string
  name: string
  running: boolean
  status: string
}

export interface PipelineTopology {
  pipeline_id: string
  name: string
  running: boolean
  components: {
    source: ComponentInfo
    processors: ComponentInfo[]
    analyzers: ComponentInfo[]
    sinks: ComponentInfo[]
  }
}

export interface ComponentInfo {
  id: string
  name: string
  type: string
  description?: string
  config?: Record<string, any>
}

export interface PipelineStatus {
  pipeline_id: string
  name: string
  running: boolean
  status: string
  samples_processed: number
  errors: number
  last_sample_time: string
}

// Trace Types
export interface TraceEvent {
  id: string
  pipeline_id: string
  sample_id: string
  component_type: string
  component_id: string
  component_name: string
  action: string
  timestamp: string
  duration: number
  status: string
  error?: string
  metadata?: Record<string, any>
  data_snapshot?: Record<string, any>
}

export interface TraceFilter {
  pipeline_id?: string
  sample_id?: string
  component_type?: string
  component_id?: string
  action?: string
  status?: string
  start_time?: string
  end_time?: string
  limit?: number
  offset?: number
  order_by?: string
  order_desc?: boolean
}

export interface TraceStats {
  total_events: number
  success_events: number
  error_events: number
  skip_events: number
  by_component_type: Record<string, number>
  by_action: Record<string, number>
  avg_duration_ms: number
}

// Metrics Types
export interface PipelineMetrics {
  pipeline_id: string
  start_time: string
  end_time: string
  summary: {
    total_samples: number
    success_samples: number
    error_samples: number
    success_rate: number
    avg_duration_ms: number
    max_duration_ms: number
    min_duration_ms: number
    throughput_per_sec: number
  }
  components: ComponentMetrics[]
}

export interface ComponentMetrics {
  component_id: string
  component_type: string
  component_name: string
  event_count: number
  avg_duration_ms: number
  max_duration_ms: number
  min_duration_ms: number
  error_count: number
  error_rate: number
}

// WebSocket Message Types
export interface WebSocketMessage {
  type: 'trace' | 'status' | 'error'
  data: TraceEvent | any
}

// API Response Types
export interface ApiResponse<T = any> {
  data?: T
  error?: string
  message?: string
}

export interface PipelinesResponse {
  pipelines: Pipeline[]
  total: number
  running: number
  stopped: number
}

export interface TracesResponse {
  traces: TraceEvent[]
  total: number
  filter: TraceFilter
}
