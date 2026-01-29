import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import PipelineDetail from '../views/PipelineDetail.vue'
import Channels from '../views/Channels.vue'
import Protocols from '../views/Protocols.vue'

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
    {
      path: '/channels',
      name: 'channels',
      component: Channels,
    },
    {
      path: '/protocols',
      name: 'protocols',
      component: Protocols,
    },
  ],
})

export default router
