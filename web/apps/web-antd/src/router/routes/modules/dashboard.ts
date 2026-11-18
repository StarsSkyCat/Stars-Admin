import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    component: () => import('#/views/dashboard/workspace/index.vue'),
    meta: {
      affixTab: true,
      icon: 'lucide:layout-dashboard',
      order: 0,
      title: '工作台',
    },
    name: 'Workspace',
    path: '/dashboard',
  },
  {
    component: () => import('#/views/_core/profile/index.vue'),
    meta: { hideInMenu: true, icon: 'lucide:user', title: '个人中心' },
    name: 'Profile',
    path: '/profile',
  },
];

export default routes;
