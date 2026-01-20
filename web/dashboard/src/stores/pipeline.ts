import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Pipeline, PipelineTopology, PipelineStatus } from '@/types'
import { apiClient } from '@/services/api'

export const usePipelineStore = defineStore('pipeline', () => {
  const pipelines = ref<Pipeline[]>([])
  const currentPipeline = ref<Pipeline | null>(null)
  const topology = ref<PipelineTopology | null>(null)
  const status = ref<PipelineStatus | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const runningCount = computed(() => pipelines.value.filter((p) => p.running).length)
  const stoppedCount = computed(() => pipelines.value.filter((p) => !p.running).length)

  async function fetchPipelines() {
    loading.value = true
    error.value = null
    try {
      const response = await apiClient.getPipelines()
      pipelines.value = response.pipelines
    } catch (err: any) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchTopology(pipelineId: string) {
    loading.value = true
    error.value = null
    try {
      topology.value = await apiClient.getPipelineTopology(pipelineId)
    } catch (err: any) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchStatus(pipelineId: string) {
    try {
      status.value = await apiClient.getPipelineStatus(pipelineId)
    } catch (err: any) {
      console.error('Failed to fetch status:', err)
    }
  }

  async function startPipeline(pipelineId: string) {
    try {
      await apiClient.startPipeline(pipelineId)
      await fetchPipelines()
    } catch (err: any) {
      error.value = err.message
      throw err
    }
  }

  async function stopPipeline(pipelineId: string) {
    try {
      await apiClient.stopPipeline(pipelineId)
      await fetchPipelines()
    } catch (err: any) {
      error.value = err.message
      throw err
    }
  }

  function selectPipeline(pipeline: Pipeline) {
    currentPipeline.value = pipeline
  }

  return {
    pipelines,
    currentPipeline,
    topology,
    status,
    loading,
    error,
    runningCount,
    stoppedCount,
    fetchPipelines,
    fetchTopology,
    fetchStatus,
    startPipeline,
    stopPipeline,
    selectPipeline,
  }
})
