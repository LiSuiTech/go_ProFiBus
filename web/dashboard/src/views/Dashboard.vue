<template>
  <div class="dashboard">
    <el-row :gutter="20" class="stats-row">
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>
            <span><el-icon><DataAnalysis /></el-icon> Total Pipelines</span>
          </template>
          <div class="stat-content">
            <h2>{{ pipelines.length }}</h2>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>
            <span><el-icon><CircleCheck /></el-icon> Running</span>
          </template>
          <div class="stat-content success">
            <h2>{{ runningCount }}</h2>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>
            <span><el-icon><CircleClose /></el-icon> Stopped</span>
          </template>
          <div class="stat-content danger">
            <h2>{{ stoppedCount }}</h2>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="hover" class="pipelines-card">
      <template #header>
        <div class="card-header">
          <span><el-icon><List /></el-icon> Pipelines</span>
          <el-button @click="fetchData" :loading="loading" circle>
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
      </template>

      <el-table :data="pipelines" style="width: 100%" v-loading="loading">
        <el-table-column prop="id" label="Pipeline ID" width="250" />
        <el-table-column prop="name" label="Name" />
        <el-table-column label="Status" width="120">
          <template #default="{ row }">
            <el-tag :type="row.running ? 'success' : 'info'">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="250">
          <template #default="{ row }">
            <el-button-group>
              <el-button
                v-if="!row.running"
                @click="handleStart(row.id)"
                type="success"
                size="small"
              >
                <el-icon><VideoPlay /></el-icon> Start
              </el-button>
              <el-button
                v-else
                @click="handleStop(row.id)"
                type="danger"
                size="small"
              >
                <el-icon><VideoPause /></el-icon> Stop
              </el-button>
              <el-button
                @click="handleViewDetail(row.id)"
                type="primary"
                size="small"
              >
                <el-icon><View /></el-icon> Detail
              </el-button>
            </el-button-group>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="hover" class="traces-card">
      <template #header>
        <span><el-icon><Histogram /></el-icon> Recent Trace Events ({{ traces.length }})</span>
      </template>
      <el-scrollbar height="300px">
        <el-timeline>
          <el-timeline-item
            v-for="trace in recentTraces"
            :key="trace.id"
            :timestamp="formatTimestamp(trace.timestamp)"
            :color="getTraceColor(trace.status)"
          >
            <p>
              <strong>{{ trace.component_name }}</strong> ({{ trace.component_type }})
              - {{ trace.action }}
            </p>
            <p class="trace-detail">
              Pipeline: {{ trace.pipeline_id }} | Status: {{ trace.status }} |
              Duration: {{ (trace.duration / 1000000).toFixed(2) }}ms
            </p>
          </el-timeline-item>
        </el-timeline>
      </el-scrollbar>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { usePipelineStore } from '@/stores/pipeline'
import { useTraceStore } from '@/stores/trace'
import { ElMessage } from 'element-plus'

const router = useRouter()
const pipelineStore = usePipelineStore()
const traceStore = useTraceStore()

const loading = ref(false)

const pipelines = computed(() => pipelineStore.pipelines)
const runningCount = computed(() => pipelineStore.runningCount)
const stoppedCount = computed(() => pipelineStore.stoppedCount)
const traces = computed(() => traceStore.traces)
const recentTraces = computed(() => traces.value.slice(-20).reverse())

async function fetchData() {
  loading.value = true
  try {
    await pipelineStore.fetchPipelines()
  } catch (error: any) {
    ElMessage.error(`Failed to fetch pipelines: ${error.message}`)
  } finally {
    loading.value = false
  }
}

async function handleStart(pipelineId: string) {
  try {
    await pipelineStore.startPipeline(pipelineId)
    ElMessage.success('Pipeline started successfully')
  } catch (error: any) {
    ElMessage.error(`Failed to start pipeline: ${error.message}`)
  }
}

async function handleStop(pipelineId: string) {
  try {
    await pipelineStore.stopPipeline(pipelineId)
    ElMessage.success('Pipeline stopped successfully')
  } catch (error: any) {
    ElMessage.error(`Failed to stop pipeline: ${error.message}`)
  }
}

function handleViewDetail(pipelineId: string) {
  router.push(`/pipeline/${pipelineId}`)
}

function formatTimestamp(timestamp: string): string {
  return new Date(timestamp).toLocaleTimeString()
}

function getTraceColor(status: string): string {
  switch (status) {
    case 'success':
      return '#67C23A'
    case 'error':
      return '#F56C6C'
    default:
      return '#909399'
  }
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.dashboard {
  padding: 20px;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-content {
  text-align: center;
  padding: 20px 0;
}

.stat-content h2 {
  margin: 0;
  font-size: 36px;
  color: #409eff;
}

.stat-content.success h2 {
  color: #67c23a;
}

.stat-content.danger h2 {
  color: #f56c6c;
}

.pipelines-card,
.traces-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.trace-detail {
  font-size: 12px;
  color: #909399;
  margin: 5px 0 0 0;
}
</style>
