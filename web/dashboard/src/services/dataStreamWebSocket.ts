export interface DeviceDataEvent {
  device_id: string
  timestamp: string
  data: Record<string, any>
  source_id?: string
  quality?: number
}

export type DataStreamCallback = (event: DeviceDataEvent) => void

export class DataStreamWebSocketService {
  private ws: WebSocket | null = null
  private url: string
  private listeners: DataStreamCallback[] = []
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 3000
  private isConnecting = false
  private shouldReconnect = true
  private filter: {
    device_ids?: string[]
    source_ids?: string[]
    fields?: string[]
    min_quality?: number
  } = {}

  constructor(url: string) {
    this.url = url
  }

  /**
   * Connect to WebSocket server
   */
  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        resolve()
        return
      }

      if (this.isConnecting) {
        reject(new Error('Already connecting'))
        return
      }

      this.isConnecting = true

      try {
        this.ws = new WebSocket(this.url)

        this.ws.onopen = () => {
          console.log('[DataStream WebSocket] Connected')
          this.isConnecting = false
          this.reconnectAttempts = 0
          
          // 发送过滤器配置
          if (Object.keys(this.filter).length > 0) {
            this.setFilter(this.filter)
          }
          
          resolve()
        }

        this.ws.onmessage = (event) => {
          try {
            const dataEvent: DeviceDataEvent = JSON.parse(event.data)
            this.notifyListeners(dataEvent)
          } catch (error) {
            console.error('[DataStream WebSocket] Failed to parse message:', error)
          }
        }

        this.ws.onerror = (error) => {
          console.error('[DataStream WebSocket] Error:', error)
          this.isConnecting = false
          reject(error)
        }

        this.ws.onclose = (event) => {
          console.log('[DataStream WebSocket] Closed:', event.code, event.reason)
          this.isConnecting = false

          // Auto reconnect
          if (this.shouldReconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++
            console.log(`[DataStream WebSocket] Reconnecting... (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`)

            setTimeout(() => {
              this.connect().catch((err) => {
                console.error('[DataStream WebSocket] Reconnect failed:', err)
              })
            }, this.reconnectDelay)
          }
        }
      } catch (error) {
        this.isConnecting = false
        reject(error)
      }
    })
  }

  /**
   * Disconnect from WebSocket server
   */
  disconnect(): void {
    this.shouldReconnect = false

    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  /**
   * Set filter for data stream
   */
  setFilter(filter: {
    device_ids?: string[]
    source_ids?: string[]
    fields?: string[]
    min_quality?: number
  }): void {
    this.filter = filter

    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(filter))
    }
  }

  /**
   * Subscribe to data events
   */
  subscribe(callback: DataStreamCallback): void {
    this.listeners.push(callback)
  }

  /**
   * Unsubscribe from data events
   */
  unsubscribe(callback: DataStreamCallback): void {
    const index = this.listeners.indexOf(callback)
    if (index > -1) {
      this.listeners.splice(index, 1)
    }
  }

  /**
   * Notify all listeners
   */
  private notifyListeners(event: DeviceDataEvent): void {
    this.listeners.forEach((listener) => {
      try {
        listener(event)
      } catch (error) {
        console.error('[DataStream WebSocket] Listener error:', error)
      }
    })
  }

  /**
   * Get connection status
   */
  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  /**
   * Get WebSocket ready state
   */
  get readyState(): number {
    return this.ws?.readyState ?? WebSocket.CLOSED
  }
}

// Singleton instance
let dataStreamWsInstance: DataStreamWebSocketService | null = null

/**
 * Get DataStream WebSocket service instance
 */
export function getDataStreamWebSocketService(): DataStreamWebSocketService {
  if (!dataStreamWsInstance) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.hostname
    const port = import.meta.env.DEV ? '8080' : window.location.port
    const url = `${protocol}//${host}:${port}/ws/data`

    dataStreamWsInstance = new DataStreamWebSocketService(url)
  }
  return dataStreamWsInstance
}
