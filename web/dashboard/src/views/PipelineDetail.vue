<template>
  <div class="pipeline-detail">
    <el-page-header @back="goBack">
      <template #content>
        <span class="page-title">Pipeline: {{ pipelineId }}</span>
      </template>
      <template #extra>
        <el-button-group>
          <el-button @click="refreshData" :loading="loading">
            <el-icon><Refresh /></el-icon> Refresh
          </el-button>
          <el-button
            v-if="!pipeline?.running"
            @click="handleStart"
            type="success"
          >
            <el-icon><VideoPlay /></el-icon> Start
          </el-button>
          <el-button v-else @click="handleStop" type="danger">
            <el-icon><VideoPause /></el-icon> Stop
          </el-button>
        </el-button-group>
      </template>
    </el-page-header>

    <el-row :gutter="20" class="content-row">
      <!-- Topology Card -->
      <el-col :span="24">
        <el-card shadow="hover" v-loading="loading">
          <template #header>
            <span><el-icon><Share /></el-icon> Pipeline Topology</span>
          </template>
          <div v-if="topology" class="topology-container">
            <div class="component-flow">
              <!-- Source -->
              <div class="component-node source">
                <el-icon><Monitor /></el-icon>
                <div class="component-label">
                  <strong>Source</strong>
                  <p>{{ topology.components.source.name }}</p>
                </div>
              </div>

              <el-icon class="flow-arrow"><Right /></el-icon>

              <!-- Processors -->
              <div v-if="topology.components.processors.length > 0" class="component-group">
                <div
                  v-for="proc in topology.components.processors"
                  :key="proc.id"
                  class="component-node processor"
                >
                  <el-icon><Setting /></el-icon>
                  <div class="component-label">
                    <strong>Processor</strong>
                    <p>{{ proc.name }}</p>
                  </div>
                </div>
                <el-icon class="flow-arrow"><Right /></el-icon>
              </div>

              <!-- Analyzers -->
              <div v-if="topology.components.analyzers.length > 0" class="component-group">
                <div
                  v-for="analyzer in topology.components.analyzers"
                  :key="analyzer.id"
                  class="component-node analyzer"
                >
                  <el-icon><TrendCharts /></el-icon>
                  <div class="component-label">
                    <strong>Analyzer</strong>
                    <p>{{ analyzer.name }}</p>
                  </div>
                </div>
                <el-icon class="flow-arrow"><Right /></el-icon>
              </div>

              <!-- Sinks -->
              <div v-if="topology.components.sinks.length > 0" class="component-group">
                <div
                  v-for="sink in topology.components.sinks"
                  :key="sink.id"
                  class="component-node sink"
                >
                  <el-icon><DocumentCopy /></el-icon>
                  <div class="component-label">
                    <strong>Sink</strong>
                    <p>{{ sink.name }}</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Status and Metrics -->
    <el-row :gutter="20" class="content-row">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <span><el-icon><DataLine /></el-icon> Status</span>
          </template>
          <el-descriptions v-if="status" :column="1" border>
            <el-descriptions-item label="Pipeline ID">{{ status.pipeline_id }}</el-descriptions-item>
            <el-descriptions-item label="Name">{{ status.name }}</el-descriptions-item>
            <el-descriptions-item label="Running">
              <el-tag :type="status.running ? 'success' : 'info'">
                {{ status.running ? 'Yes' : 'No' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Status">
              <el-tag>{{ status.status }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Samples Processed">
              {{ status.samples_processed }}
            </el-descriptions-item>
            <el-descriptions-item label="Errors">{{ status.errors }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <span><el-icon><Histogram /></el-icon> Performance Metrics</span>
          </template>
          <div v-if="metrics" class="metrics-container">
            <el-statistic title="Throughput" :value="metrics.summary.throughput_per_sec.toFixed(2)" suffix=" samples/sec" />
            <el-divider />
            <el-statistic title="Avg Duration" :value="metrics.summary.avg_duration_ms.toFixed(2)" suffix=" ms" />
            <el-divider />
            <el-statistic title="Success Rate" :value="metrics.summary.success_rate.toFixed(1)" suffix=" %" />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { usePipelineStore } from '@/stores/pipeline'
import { apiClient } from '@/services/api'
import { ElMessage } from 'element-plus'
import type { PipelineMetrics } from '@/types'

const route = useRoute()
const router = useRouter()
const pipelineStore = usePipelineStore()

const pipelineId = route.params.id as string
const loading = ref(false)
const metrics = ref<PipelineMetrics | null>(null)

const pipeline = computed(() =>
  pipelineStore.pipelines.find((p) => p.id === pipelineId)
)
const topology = computed(() => pipelineStore.topology)
const status = computed(() => pipelineStore.status)

async function refreshData() {
  loading.value = true
  try {
    await Promise.all([
      pipelineStore.fetchTopology(pipelineId),
      pipelineStore.fetchStatus(pipelineId),
      fetchMetrics(),
    ])
  } catch (error: any) {
    ElMessage.error(`Failed to refresh data: ${error.message}`)
  } finally {
    loading.value = false
  }
}

async function fetchMetrics() {
  try {
    metrics.value = await apiClient.getPipelineMetrics(pipelineId)
  } catch (error) {
    console.error('Failed to fetch metrics:', error)
  }
}

async function handleStart() {
  try {
    await pipelineStore.startPipeline(pipelineId)
    ElMessage.success('Pipeline started')
    await refreshData()
  } catch (error: any) {
    ElMessage.error(`Failed to start: ${error.message}`)
  }
}

async function handleStop() {
  try {
    await pipelineStore.stopPipeline(pipelineId)
    ElMessage.success('Pipeline stopped')
    await refreshData()
  } catch (error: any) {
    ElMessage.error(`Failed to stop: ${error.message}`)
  }
}

function goBack() {
  router.push('/')
}

onMounted(() => {
  refreshData()
  // Auto refresh every 5 seconds
  const interval = setInterval(refreshData, 5000)
  // Cleanup on unmount
  onUnmounted(() => clearInterval(interval))
})

function onUnmounted(callback: () => void) {
  return callback
}
</script>

<style scoped>
.pipeline-detail {
  padding: 20px;
}

.page-title {
  font-size: 20px;
  font-weight: bold;
}

.content-row {
  margin-top: 20px;
}

.topology-container {
  min-height: 200px;
  padding: 20px;
}

.component-flow {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 20px;
}

.component-group {
  display: flex;
  align-items: center;
  gap: 10px;
}

.component-node {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 15px;
  border-radius: 8px;
  background-color: #f5f7fa;
  border: 2px solid #dcdfe6;
  min-width: 120px;
  transition: all 0.3s;
}

.component-node:hover {
  transform: translateY(-2px);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.component-node.source {
  border-color: #67c23a;
  background-color: #f0f9ff;
}

.component-node.processor {
  border-color: #409eff;
  background-color: #ecf5ff;
}

.component-node.analyzer {
  border-color: #e6a23c;
  background-color: #fdf6ec;
}

.component-node.sink {
  border-color: #f56c6c;
  background-color: #fef0f0;
}

.component-label {
  text-align: center;
  margin-top: 8px;
}

.component-label strong {
  display: block;
  font-size: 14px;
  margin-bottom: 4px;
}

.component-label p {
  margin: 0;
  font-size: 12px;
  color: #606266;
}

.flow-arrow {
  font-size: 24px;
  color: #909399;
}

.metrics-container {
  padding: 20px;
}
</style>
