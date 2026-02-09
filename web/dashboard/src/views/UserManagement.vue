<template>
  <div class="page">
    <el-row :gutter="20" class="page-row">
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card stat-card--primary">
          <template #header>
            <span>用户总数</span>
          </template>
          <div class="stat-value">—</div>
          <div class="stat-caption">接入系统的账户数量</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <template #header>
            <span>在线用户</span>
          </template>
          <div class="stat-value">—</div>
          <div class="stat-caption">当前活跃会话</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <template #header>
            <span>禁用账户</span>
          </template>
          <div class="stat-value">—</div>
          <div class="stat-caption">因安全原因被禁用的用户</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>用户列表</span>
          <el-button type="primary" size="small">
            新建用户
          </el-button>
        </div>
      </template>

      <el-table :data="[]" style="width: 100%">
        <el-table-column prop="username" label="用户名" min-width="140" />
        <el-table-column prop="display_name" label="显示名称" min-width="160" />
        <el-table-column prop="roles" label="角色" min-width="160" />
        <el-table-column prop="status" label="状态" width="120" />
        <el-table-column prop="last_login" label="最近登录" width="200" />
        <el-table-column label="操作" width="220" fixed="right" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped>
.page {
  padding: 8px 0 0;
}

.page-row {
  margin-bottom: 20px;
}

.stat-card {
  border-left: 3px solid rgba(148, 163, 184, 0.6);
}

.stat-card--primary {
  border-left-color: #06b6d4;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  margin-bottom: 4px;
}

.stat-caption {
  font-size: 12px;
  color: #64748b;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>

