import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { TraceEvent } from '@/types'
import { getWebSocketService } from '@/services/websocket'

const MAX_TRACES = 1000

export const useTraceStore = defineStore('trace', () => {
  const traces = ref<TraceEvent[]>([])
  const wsConnected = ref(false)
  const wsService = getWebSocketService()

  function connect() {
    wsService
      .connect()
      .then(() => {
        wsConnected.value = true
        console.log('[TraceStore] WebSocket connected')

        // Subscribe to trace events
        wsService.subscribe(handleTraceEvent)
      })
      .catch((error) => {
        console.error('[TraceStore] WebSocket connection failed:', error)
        wsConnected.value = false
      })
  }

  function disconnect() {
    wsService.disconnect()
    wsConnected.value = false
  }

  function handleTraceEvent(event: TraceEvent) {
    traces.value.push(event)

    // Keep only the most recent traces
    if (traces.value.length > MAX_TRACES) {
      traces.value.shift()
    }
  }

  function clearTraces() {
    traces.value = []
  }

  return {
    traces,
    wsConnected,
    connect,
    disconnect,
    clearTraces,
  }
})
