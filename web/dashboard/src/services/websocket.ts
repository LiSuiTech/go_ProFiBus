import type { TraceEvent } from '@/types'

export type WebSocketEventCallback = (event: TraceEvent) => void

export class WebSocketService {
  private ws: WebSocket | null = null
  private url: string
  private listeners: WebSocketEventCallback[] = []
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 3000
  private isConnecting = false
  private shouldReconnect = true

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
          console.log('[WebSocket] Connected')
          this.isConnecting = false
          this.reconnectAttempts = 0
          resolve()
        }

        this.ws.onmessage = (event) => {
          try {
            const traceEvent: TraceEvent = JSON.parse(event.data)
            this.notifyListeners(traceEvent)
          } catch (error) {
            console.error('[WebSocket] Failed to parse message:', error)
          }
        }

        this.ws.onerror = (error) => {
          console.error('[WebSocket] Error:', error)
          this.isConnecting = false
          reject(error)
        }

        this.ws.onclose = (event) => {
          console.log('[WebSocket] Closed:', event.code, event.reason)
          this.isConnecting = false

          // Auto reconnect
          if (this.shouldReconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++
            console.log(`[WebSocket] Reconnecting... (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`)

            setTimeout(() => {
              this.connect().catch((err) => {
                console.error('[WebSocket] Reconnect failed:', err)
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
   * Subscribe to trace events
   */
  subscribe(callback: WebSocketEventCallback): void {
    this.listeners.push(callback)
  }

  /**
   * Unsubscribe from trace events
   */
  unsubscribe(callback: WebSocketEventCallback): void {
    const index = this.listeners.indexOf(callback)
    if (index > -1) {
      this.listeners.splice(index, 1)
    }
  }

  /**
   * Notify all listeners
   */
  private notifyListeners(event: TraceEvent): void {
    this.listeners.forEach((listener) => {
      try {
        listener(event)
      } catch (error) {
        console.error('[WebSocket] Listener error:', error)
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
let wsInstance: WebSocketService | null = null

/**
 * Get WebSocket service instance
 */
export function getWebSocketService(): WebSocketService {
  if (!wsInstance) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.hostname
    const port = import.meta.env.DEV ? '8080' : window.location.port
    const url = `${protocol}//${host}:${port}/ws/trace`

    wsInstance = new WebSocketService(url)
  }
  return wsInstance
}
