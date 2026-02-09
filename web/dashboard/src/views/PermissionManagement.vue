<template>
  <div class="page">
    <el-row :gutter="20" class="page-row">
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card stat-card--primary">
          <template #header>
            <span>权限点总数</span>
          </template>
          <div class="stat-value">—</div>
          <div class="stat-caption">系统中可配置的功能权限点</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <template #header>
            <span>敏感权限</span>
          </template>
          <div class="stat-value">—</div>
          <div class="stat-caption">涉及设备控制与系统配置的权限</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <template #header>
            <span>最近变更</span>
          </template>
          <div class="stat-value">—</div>
          <div class="stat-caption">近期被修改过的权限策略</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>权限策略</span>
          <el-button type="primary" size="small">
            新建策略
          </el-button>
        </div>
      </template>

      <el-table :data="[]" style="width: 100%">
        <el-table-column prop="name" label="策略名称" min-width="160" />
        <el-table-column prop="scope" label="作用范围" min-width="200" />
        <el-table-column prop="updated_at" label="最近更新" width="200" />
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

