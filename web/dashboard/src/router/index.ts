import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import PipelineDetail from '../views/PipelineDetail.vue'
import Channels from '../views/Channels.vue'
import Protocols from '../views/Protocols.vue'
import Devices from '../views/Devices.vue'
import Alerts from '../views/Alerts.vue'
import DeviceLayout from '../views/DeviceLayout.vue'
import Predictions from '../views/Predictions.vue'
import Control from '../views/Control.vue'
import DataManagement from '../views/DataManagement.vue'
import WorkflowEditor from '../views/WorkflowEditor.vue'
import RealtimeDataStream from '../views/RealtimeDataStream.vue'
import UserManagement from '../views/UserManagement.vue'
import RoleManagement from '../views/RoleManagement.vue'
import PermissionManagement from '../views/PermissionManagement.vue'

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
    {
      path: '/devices',
      name: 'devices',
      component: Devices,
    },
    {
      path: '/alerts',
      name: 'alerts',
      component: Alerts,
    },
    {
      path: '/device-layout',
      name: 'device-layout',
      component: DeviceLayout,
    },
    {
      path: '/predictions',
      name: 'predictions',
      component: Predictions,
    },
    {
      path: '/control',
      name: 'control',
      component: Control,
    },
    {
      path: '/data-management',
      name: 'data-management',
      component: DataManagement,
    },
    {
      path: '/users',
      name: 'users',
      component: UserManagement,
    },
    {
      path: '/roles',
      name: 'roles',
      component: RoleManagement,
    },
    {
      path: '/permissions',
      name: 'permissions',
      component: PermissionManagement,
    },
    {
      path: '/workflow-editor',
      name: 'workflow-editor',
      component: WorkflowEditor,
    },
    {
      path: '/realtime-data-stream',
      name: 'realtime-data-stream',
      component: RealtimeDataStream,
    },
  ],
})

export default router
