<template>
  <el-container class="app-container">
    <el-header class="app-header">
      <div class="header-content">
        <h1>go_ProFiBus Dashboard</h1>
        <div class="header-nav">
          <el-menu
            mode="horizontal"
            :default-active="activeMenu"
            class="header-menu"
            background-color="#409eff"
            text-color="#fff"
            active-text-color="#ffd04b"
            router
          >
            <el-menu-item index="/">
              <el-icon><Monitor /></el-icon>
              <span>控制面板</span>
            </el-menu-item>
            <el-menu-item index="/channels">
              <el-icon><Connection /></el-icon>
              <span>采集通道</span>
            </el-menu-item>
          </el-menu>
        </div>
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
import { useRoute } from 'vue-router'
import { useTraceStore } from './stores/trace'
import { Monitor } from '@element-plus/icons-vue'

const route = useRoute()
const traceStore = useTraceStore()

const wsConnected = computed(() => traceStore.wsConnected)
const activeMenu = computed(() => route.path)

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
  gap: 20px;
}

.header-content h1 {
  margin: 0;
  font-size: 24px;
  white-space: nowrap;
}

.header-nav {
  flex: 1;
}

.header-menu {
  border-bottom: none;
}

.header-menu .el-menu-item {
  border-bottom: none;
}

.app-main {
  background-color: #f5f7fa;
  padding: 0;
}
</style>
