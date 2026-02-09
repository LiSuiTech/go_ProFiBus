<template>
  <el-container class="app-shell">
    <el-aside class="app-sidebar" width="260px">
      <div class="sidebar-inner">
        <div class="sidebar-logo">
          <div class="logo-mark">PB</div>
          <div class="logo-text">
            <span class="logo-title">go_ProFiBus</span>
            <span class="logo-subtitle">Industrial Dashboard</span>
          </div>
        </div>

        <el-menu
          class="sidebar-menu"
          :default-active="activeMenu"
          background-color="transparent"
          text-color="var(--color-text-muted)"
          active-text-color="var(--color-accent)"
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
          <el-menu-item index="/devices">
            <el-icon><Setting /></el-icon>
            <span>设备管理</span>
          </el-menu-item>
          <el-menu-item index="/alerts">
            <el-icon><Bell /></el-icon>
            <span>告警中心</span>
          </el-menu-item>
          <el-menu-item index="/device-layout">
            <el-icon><MapLocation /></el-icon>
            <span>设备布局</span>
          </el-menu-item>
          <el-menu-item index="/predictions">
            <el-icon><TrendCharts /></el-icon>
            <span>预测分析</span>
          </el-menu-item>
          <el-menu-item index="/control">
            <el-icon><SwitchButton /></el-icon>
            <span>设备控制</span>
          </el-menu-item>
          <el-menu-item index="/data-management">
            <el-icon><FolderOpened /></el-icon>
            <span>数据管理</span>
          </el-menu-item>
          <el-menu-item index="/users">
            <el-icon><User /></el-icon>
            <span>用户管理</span>
          </el-menu-item>
          <el-menu-item index="/roles">
            <el-icon><UserFilled /></el-icon>
            <span>角色管理</span>
          </el-menu-item>
          <el-menu-item index="/permissions">
            <el-icon><Lock /></el-icon>
            <span>权限管理</span>
          </el-menu-item>
          <el-menu-item index="/workflow-editor">
            <el-icon><Operation /></el-icon>
            <span>工作流编辑器</span>
          </el-menu-item>
          <el-menu-item index="/realtime-data-stream">
            <el-icon><DataLine /></el-icon>
            <span>实时数据流</span>
          </el-menu-item>
        </el-menu>

        <div class="sidebar-footer">
          <div
            class="status-pill"
            :class="wsConnected ? 'status-pill--online' : 'status-pill--offline'"
          >
            <span class="status-dot" />
            <span class="status-label">
              {{ wsConnected ? 'Connected' : 'Disconnected' }}
            </span>
          </div>
        </div>
      </div>
    </el-aside>

    <el-container class="app-main-container">
      <el-header class="app-header" height="72px">
        <div class="header-main">
          <div>
            <h1 class="page-title">{{ currentTitle }}</h1>
            <p class="page-subtitle">
              {{ currentDescription }}
            </p>
          </div>
        </div>
      </el-header>

      <el-main class="app-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useTraceStore } from './stores/trace'
import {
  Monitor,
  Connection,
  Setting,
  Bell,
  MapLocation,
  TrendCharts,
  SwitchButton,
  FolderOpened,
  Operation,
  DataLine,
  User,
  UserFilled,
  Lock,
} from '@element-plus/icons-vue'

const route = useRoute()
const traceStore = useTraceStore()

const wsConnected = computed(() => traceStore.wsConnected)

const activeMenu = computed(() => {
  if (route.name === 'pipeline-detail') {
    return '/'
  }
  return route.path
})

const pageMeta: Record<string, { title: string; description: string }> = {
  '/': {
    title: '控制面板',
    description: '总览所有 Pipeline 的运行状态与最近追踪事件',
  },
  '/channels': {
    title: '采集通道',
    description: '配置与监控工业现场的数据采集通道',
  },
  '/devices': {
    title: '设备管理',
    description: '管理工业设备与健康状态',
  },
  '/alerts': {
    title: '告警中心',
    description: '集中查看与处理系统告警',
  },
  '/device-layout': {
    title: '设备布局',
    description: '可视化厂区与产线设备拓扑',
  },
  '/predictions': {
    title: '预测分析',
    description: '基于历史数据与模型的预测结果',
  },
  '/control': {
    title: '设备控制',
    description: '远程控制与操作工业设备',
  },
  '/data-management': {
    title: '数据管理',
    description: '数据清洗、归档与生命周期管理',
  },
  '/users': {
    title: '用户管理',
    description: '管理平台用户账户与登录访问',
  },
  '/roles': {
    title: '角色管理',
    description: '配置角色模型与职责边界',
  },
  '/permissions': {
    title: '权限管理',
    description: '维护系统权限点与访问策略',
  },
  '/workflow-editor': {
    title: '工作流编辑器',
    description: '设计数据处理与控制的 DAG 工作流',
  },
  '/realtime-data-stream': {
    title: '实时数据流',
    description: '监控实时数据与流量指标',
  },
}

const currentTitle = computed(() => {
  return pageMeta[route.path]?.title ?? 'go_ProFiBus Dashboard'
})

const currentDescription = computed(() => {
  return pageMeta[route.path]?.description ?? '工业现场总线数据采集、处理与分析平台'
})

onMounted(() => {
  traceStore.connect()
})

onUnmounted(() => {
  traceStore.disconnect()
})
</script>

<style>
:root {
  --color-primary: #1e40af;
  --color-primary-soft: #1d4ed8;
  --color-accent: #06b6d4;
  --color-bg-page: #f8fafc;
  --color-bg-sidebar: #020617;
  --color-bg-main: #f8fafc;
  --color-surface: #ffffff;
  --color-border-subtle: rgba(148, 163, 184, 0.35);
  --color-text-main: #0f172a;
  --color-text-muted: #64748b;
  --color-success: #22c55e;
  --color-warning: #f59e0b;
  --color-danger: #ef4444;
  --shadow-soft: 0 18px 45px rgba(15, 23, 42, 0.18);
  --radius-lg: 18px;
  --radius-md: 12px;
  --radius-pill: 999px;
  --transition-fast: 180ms ease-out;
  --transition-normal: 240ms ease-out;
}

html,
body,
#app {
  height: 100%;
}

body {
  margin: 0;
  font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  background: radial-gradient(circle at top left, #e0f2fe 0, #f8fafc 36%, #e5e7eb 100%);
  color: var(--color-text-main);
}

/* 通用卡片风格：工业现代主义 */
.el-card {
  border-radius: var(--radius-lg);
  border-color: rgba(148, 163, 184, 0.25);
  box-shadow: var(--shadow-soft);
  background: rgba(255, 255, 255, 0.96);
  backdrop-filter: blur(12px);
}

.el-card__header {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-main);
  padding: 14px 18px;
  border-bottom-color: rgba(226, 232, 240, 0.7);
}

/* 表格细线与交替行背景 */
.el-table {
  --el-table-border-color: rgba(226, 232, 240, 0.8);
}

.el-table tr:nth-child(2n) > td {
  background-color: rgba(248, 250, 252, 0.8);
}

/* 滚动条统一样式 */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-thumb {
  background-color: rgba(148, 163, 184, 0.8);
  border-radius: 999px;
}

::-webkit-scrollbar-track {
  background-color: transparent;
}
</style>

<style scoped>
.app-shell {
  height: 100vh;
  background: transparent;
}

.app-sidebar {
  display: flex;
  flex-direction: column;
  padding: 20px 16px;
  background: linear-gradient(180deg, #020617 0%, #020617 40%, #020617 100%);
  color: #e5e7eb;
  border-right: 1px solid rgba(30, 64, 175, 0.65);
  box-shadow: 18px 0 40px rgba(15, 23, 42, 0.55);
}

.sidebar-inner {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.sidebar-logo {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 8px 16px;
  border-bottom: 1px solid rgba(30, 64, 175, 0.7);
  margin-bottom: 16px;
}

.logo-mark {
  width: 34px;
  height: 34px;
  border-radius: 12px;
  background: radial-gradient(circle at 20% 0, #38bdf8 0, #1e40af 40%, #020617 90%);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 14px;
  letter-spacing: 0.03em;
  color: #e5e7eb;
  box-shadow: 0 10px 25px rgba(15, 23, 42, 0.75);
}

.logo-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.logo-title {
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.04em;
}

.logo-subtitle {
  font-size: 11px;
  color: rgba(148, 163, 184, 0.9);
}

.sidebar-menu {
  margin-top: 12px;
  flex: 1;
  border-right: none;
  background-color: transparent;
}

.sidebar-menu .el-menu-item {
  border-radius: 999px;
  margin-bottom: 4px;
  padding-left: 14px !important;
  padding-right: 14px !important;
  height: 40px;
  line-height: 40px;
  display: flex;
  align-items: center;
  gap: 10px;
  transition:
    background-color var(--transition-fast),
    color var(--transition-fast),
    transform var(--transition-fast);
}

.sidebar-menu .el-menu-item .el-icon {
  font-size: 16px;
}

.sidebar-menu .el-menu-item:hover {
  background-color: rgba(15, 23, 42, 0.9);
  color: #e5e7eb;
  transform: translateX(2px);
}

.sidebar-menu .el-menu-item.is-active {
  background: linear-gradient(90deg, #1e40af, #06b6d4);
  color: #f9fafb;
  box-shadow: 0 10px 25px rgba(15, 23, 42, 0.65);
}

.sidebar-menu .el-menu-item.is-active .el-icon {
  color: #e0f2fe;
}

.sidebar-footer {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid rgba(30, 64, 175, 0.7);
  display: flex;
  justify-content: flex-start;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border-radius: var(--radius-pill);
  border: 1px solid rgba(148, 163, 184, 0.6);
  background: rgba(15, 23, 42, 0.9);
  font-size: 12px;
  color: rgba(226, 232, 240, 0.9);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background-color: var(--color-danger);
}

.status-label {
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.status-pill--online {
  border-color: rgba(34, 197, 94, 0.7);
  background: rgba(22, 163, 74, 0.15);
}

.status-pill--online .status-dot {
  background-color: var(--color-success);
  box-shadow: 0 0 0 0 rgba(34, 197, 94, 0.8);
  animation: statusPulse 1.8s infinite;
}

.status-pill--offline {
  border-color: rgba(239, 68, 68, 0.7);
  background: rgba(127, 29, 29, 0.25);
}

.status-pill--offline .status-dot {
  background-color: var(--color-danger);
}

.app-main-container {
  background: transparent;
}

.app-header {
  display: flex;
  align-items: center;
  padding: 18px 28px 10px;
  background-color: transparent;
  border-bottom: none;
}

.header-main {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.page-title {
  margin: 0 0 4px;
  font-size: 24px;
  font-weight: 600;
  letter-spacing: 0.02em;
  color: var(--color-text-main);
}

.page-subtitle {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-muted);
}

.app-main {
  background-color: transparent;
  padding: 8px 28px 24px;
}

@keyframes statusPulse {
  0% {
    box-shadow: 0 0 0 0 rgba(34, 197, 94, 0.7);
  }
  70% {
    box-shadow: 0 0 0 10px rgba(34, 197, 94, 0);
  }
  100% {
    box-shadow: 0 0 0 0 rgba(34, 197, 94, 0);
  }
}
</style>
