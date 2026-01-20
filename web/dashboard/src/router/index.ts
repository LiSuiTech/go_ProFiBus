import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import PipelineDetail from '../views/PipelineDetail.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: Dashboard,
    },
    {
      path: '/pipeline/:id',
      name: 'pipeline-detail',
      component: PipelineDetail,
    },
  ],
})

export default router
