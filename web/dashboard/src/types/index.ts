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

// ========== Phase 3: Configuration Types ==========

export interface RuleConfig {
  id: string
  name: string
  description: string
  type: 'threshold' | 'statistical' | 'ml'
  version: string
  parameters: Record<string, any>
  enabled: boolean
  priority: number
  created_at: string
  updated_at: string
  created_by: string
  updated_by: string
}

export interface AnalyzerConfig {
  id: string
  name: string
  description: string
  type: 'statistical' | 'ml' | 'ensemble'
  version: string
  parameters: Record<string, any>
  rule_ids: string[]
  enabled: boolean
  created_at: string
  updated_at: string
  created_by: string
  updated_by: string
}

export interface ProcessorConfig {
  id: string
  name: string
  description: string
  type: 'filter' | 'transform' | 'aggregate'
  version: string
  parameters: Record<string, any>
  order: number
  enabled: boolean
  created_at: string
  updated_at: string
  created_by: string
  updated_by: string
}

export interface ConfigTemplate {
  id: string
  name: string
  description: string
  type: 'rule' | 'analyzer' | 'processor'
  category: string
  schema: ConfigSchema
  defaults: Record<string, any>
  created_at: string
  updated_at: string
}

export interface ConfigSchema {
  type: string
  properties: Record<string, SchemaProperty>
  required: string[]
  examples?: Record<string, any>[]
}

export interface SchemaProperty {
  type: string
  description: string
  default?: any
  minimum?: number
  maximum?: number
  enum?: string[]
  pattern?: string
}

export interface ConfigHistory {
  id: string
  config_id: string
  config_type: string
  action: 'created' | 'updated' | 'deleted'
  changed_by: string
  changed_at: string
  old_value: Record<string, any>
  new_value: Record<string, any>
  reason: string
}

// ========== Phase 3: RBAC Types ==========

export interface User {
  id: string
  username: string
  email: string
  full_name: string
  role_ids: string[]
  enabled: boolean
  created_at: string
  updated_at: string
  last_login_at?: string
}

export interface Role {
  id: string
  name: string
  description: string
  permissions: string[]
  created_at: string
  updated_at: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: User
  expires_at: string
}

export interface CreateUserRequest {
  username: string
  email: string
  password: string
  full_name?: string
  role_ids?: string[]
}

export interface UpdateUserRequest {
  email?: string
  full_name?: string
  role_ids?: string[]
  enabled?: boolean
}

export interface ChangePasswordRequest {
  old_password: string
  new_password: string
}

// API Response Types for Configuration and RBAC
export interface RulesResponse {
  rules: RuleConfig[]
  count: number
}

export interface AnalyzersResponse {
  analyzers: AnalyzerConfig[]
  count: number
}

export interface ProcessorsResponse {
  processors: ProcessorConfig[]
  count: number
}

export interface TemplatesResponse {
  templates: ConfigTemplate[]
  count: number
}

export interface UsersResponse {
  users: User[]
  count: number
}

export interface RolesResponse {
  roles: Role[]
  count: number
}

// ========== Phase 4: Channel Management Types ==========

export type ProtocolType =
  | 'UART'
  | 'CAN'
  | 'USB'
  | '1-Wire'
  | 'Modbus'
  | 'RS-232'
  | 'RS-485'
  | 'I2C'
  | 'SPI'

export type ChannelStatus = 'stopped' | 'running' | 'error'

export type DataType = 'int' | 'float' | 'bool' | 'string' | 'bytes'

export interface ProtocolConfig {
  // 串口通用配置
  baud_rate?: number
  data_bits?: number
  stop_bits?: number
  parity?: 'none' | 'odd' | 'even'

  // Modbus 配置
  slave_id?: number
  modbus_mode?: 'RTU' | 'TCP'

  // I2C 配置
  i2c_address?: string

  // 其他配置
  timeout?: number
  retry_count?: number
  poll_interval?: number
}

export interface Channel {
  id: string
  name: string
  description: string
  protocol: ProtocolType
  device_name: string
  device_port: string
  status: ChannelStatus
  config: ProtocolConfig
  enabled: boolean
  created_at: string
  updated_at: string
  points?: Point[]
}

export interface Point {
  id: string
  channel_id: string
  name: string
  description: string
  address: string
  data_type: DataType
  unit: string
  scale: number
  offset: number
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface ProtocolInfo {
  type: ProtocolType
  name: string
  description: string
  config_fields: ConfigField[]
}

export interface ConfigField {
  key: string
  label: string
  type: 'text' | 'number' | 'select'
  required: boolean
  default_value?: any
  options?: Option[]
}

export interface Option {
  label: string
  value: string
}

export interface ChannelsResponse {
  channels: Channel[]
  total: number
}

export interface PointsResponse {
  points: Point[]
  total: number
}

export interface ProtocolsResponse {
  protocols: ProtocolInfo[]
}

export interface CreateChannelRequest {
  name: string
  description?: string
  protocol: ProtocolType
  device_name: string
  device_port: string
  config: ProtocolConfig
  enabled?: boolean
}

export interface UpdateChannelRequest {
  name?: string
  description?: string
  protocol?: ProtocolType
  device_name?: string
  device_port?: string
  config?: ProtocolConfig
  enabled?: boolean
}

export interface CreatePointRequest {
  name: string
  description?: string
  address: string
  data_type: DataType
  unit?: string
  scale?: number
  offset?: number
  enabled?: boolean
}

export interface UpdatePointRequest {
  name?: string
  description?: string
  address?: string
  data_type?: DataType
  unit?: string
  scale?: number
  offset?: number
  enabled?: boolean
}
