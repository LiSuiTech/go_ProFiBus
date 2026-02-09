<template>
  <div class="page">
    <el-row :gutter="20" class="page-row">
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card stat-card--primary">
          <template #header>
            <span>角色总数</span>
          </template>
          <div class="stat-value">—</div>
          <div class="stat-caption">系统中定义的角色数量</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <template #header>
            <span>带控制权限的角色</span>
          </template>
          <div class="stat-value">—</div>
          <div class="stat-caption">具备设备控制能力的角色</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <template #header>
            <span>只读角色</span>
          </template>
          <div class="stat-value">—</div>
          <div class="stat-caption">仅允许查看数据与告警的角色</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>角色列表</span>
          <el-button type="primary" size="small">
            新建角色
          </el-button>
        </div>
      </template>

      <el-table :data="[]" style="width: 100%">
        <el-table-column prop="name" label="角色名称" min-width="160" />
        <el-table-column prop="description" label="描述" min-width="220" />
        <el-table-column prop="user_count" label="关联用户数" width="140" />
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

