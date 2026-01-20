<template>
  <div class="trace-timeline">
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span><el-icon><Clock /></el-icon> Trace Timeline</span>
          <div class="header-actions">
            <el-select v-model="filterPipeline" placeholder="Filter by pipeline" clearable style="width: 200px; margin-right: 10px;">
              <el-option
                v-for="pipeline in uniquePipelines"
                :key="pipeline"
                :label="pipeline"
                :value="pipeline"
              />
            </el-select>
            <el-select v-model="filterStatus" placeholder="Filter by status" clearable style="width: 150px; margin-right: 10px;">
              <el-option label="Success" value="success" />
              <el-option label="Error" value="error" />
              <el-option label="Skip" value="skip" />
            </el-select>
            <el-button @click="clearFilters" size="small">
              <el-icon><RefreshRight /></el-icon> Clear
            </el-button>
          </div>
        </div>
      </template>

      <el-scrollbar :height="height">
        <el-empty v-if="filteredTraces.length === 0" description="No trace events" />

        <el-timeline v-else>
          <el-timeline-item
            v-for="trace in displayTraces"
            :key="trace.id"
            :timestamp="formatTimestamp(trace.timestamp)"
            :color="getStatusColor(trace.status)"
            :hollow="trace.status === 'skip'"
          >
            <el-card class="trace-item">
              <div class="trace-header">
                <el-tag :type="getStatusTagType(trace.status)" size="small">
                  {{ trace.status }}
                </el-tag>
                <el-tag type="info" size="small">{{ trace.action }}</el-tag>
                <el-tag size="small">{{ trace.component_type }}</el-tag>
              </div>

              <div class="trace-content">
                <p><strong>Component:</strong> {{ trace.component_name }}</p>
                <p><strong>Pipeline:</strong> {{ trace.pipeline_id }}</p>
                <p v-if="trace.sample_id"><strong>Sample ID:</strong> {{ trace.sample_id }}</p>
                <p><strong>Duration:</strong> {{ formatDuration(trace.duration) }}</p>
              </div>

              <div v-if="trace.error" class="trace-error">
                <el-alert type="error" :closable="false">
                  <template #title>
                    <strong>Error:</strong> {{ trace.error }}
                  </template>
                </el-alert>
              </div>

              <div v-if="trace.metadata && Object.keys(trace.metadata).length > 0" class="trace-metadata">
                <el-collapse>
                  <el-collapse-item title="Metadata" name="metadata">
                    <pre>{{ JSON.stringify(trace.metadata, null, 2) }}</pre>
                  </el-collapse-item>
                </el-collapse>
              </div>
            </el-card>
          </el-timeline-item>
        </el-timeline>

        <div v-if="hasMore" class="load-more">
          <el-button @click="loadMore" type="primary" plain>
            Load More
          </el-button>
        </div>
      </el-scrollbar>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { TraceEvent } from '@/types'

interface Props {
  traces: TraceEvent[]
  height?: string
  pageSize?: number
}

const props = withDefaults(defineProps<Props>(), {
  height: '600px',
  pageSize: 20,
})

const filterPipeline = ref<string>('')
const filterStatus = ref<string>('')
const currentPage = ref(1)

const uniquePipelines = computed(() => {
  const pipelines = new Set(props.traces.map((t) => t.pipeline_id))
  return Array.from(pipelines)
})

const filteredTraces = computed(() => {
  let traces = [...props.traces]

  if (filterPipeline.value) {
    traces = traces.filter((t) => t.pipeline_id === filterPipeline.value)
  }

  if (filterStatus.value) {
    traces = traces.filter((t) => t.status === filterStatus.value)
  }

  return traces.sort((a, b) =>
    new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
  )
})

const displayTraces = computed(() => {
  const endIndex = currentPage.value * props.pageSize
  return filteredTraces.value.slice(0, endIndex)
})

const hasMore = computed(() => {
  return displayTraces.value.length < filteredTraces.value.length
})

function clearFilters() {
  filterPipeline.value = ''
  filterStatus.value = ''
  currentPage.value = 1
}

function loadMore() {
  currentPage.value++
}

function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp)
  return date.toLocaleString()
}

function formatDuration(duration: number): string {
  const ms = duration / 1000000
  if (ms < 1) {
    return `${(duration / 1000).toFixed(2)} μs`
  } else if (ms < 1000) {
    return `${ms.toFixed(2)} ms`
  } else {
    return `${(ms / 1000).toFixed(2)} s`
  }
}

function getStatusColor(status: string): string {
  switch (status) {
    case 'success':
      return '#67C23A'
    case 'error':
      return '#F56C6C'
    case 'skip':
      return '#E6A23C'
    default:
      return '#909399'
  }
}

function getStatusTagType(status: string): 'success' | 'danger' | 'warning' | 'info' {
  switch (status) {
    case 'success':
      return 'success'
    case 'error':
      return 'danger'
    case 'skip':
      return 'warning'
    default:
      return 'info'
  }
}
</script>

<style scoped>
.trace-timeline {
  width: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  align-items: center;
}

.trace-item {
  margin-bottom: 10px;
}

.trace-header {
  display: flex;
  gap: 8px;
  margin-bottom: 10px;
}

.trace-content p {
  margin: 5px 0;
  font-size: 14px;
}

.trace-error {
  margin-top: 10px;
}

.trace-metadata {
  margin-top: 10px;
}

.trace-metadata pre {
  background-color: #f5f7fa;
  padding: 10px;
  border-radius: 4px;
  font-size: 12px;
  overflow-x: auto;
}

.load-more {
  text-align: center;
  padding: 20px;
}
</style>
