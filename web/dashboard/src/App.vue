<template>
  <el-container class="app-container">
    <el-header class="app-header">
      <div class="header-content">
        <h1>go_ProFiBus Dashboard</h1>
        <div class="header-actions">
          <el-tag :type="wsConnected ? 'success' : 'danger'">
            <el-icon><Connection /></el-icon>
            {{ wsConnected ? 'Connected' : 'Disconnected' }}
          </el-tag>
        </div>
      </div>
    </el-header>

    <el-main class="app-main">
      <router-view />
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { useTraceStore } from './stores/trace'

const traceStore = useTraceStore()

const wsConnected = computed(() => traceStore.wsConnected)

onMounted(() => {
  traceStore.connect()
})

onUnmounted(() => {
  traceStore.disconnect()
})
</script>

<style scoped>
.app-container {
  height: 100vh;
}

.app-header {
  background-color: #409eff;
  color: white;
  display: flex;
  align-items: center;
  padding: 0 20px;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.header-content h1 {
  margin: 0;
  font-size: 24px;
}

.app-main {
  background-color: #f5f7fa;
  padding: 20px;
}
</style>
