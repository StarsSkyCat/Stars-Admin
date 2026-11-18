import type { RouteRecordRaw } from 'vue-router';

import { LOGIN_PATH } from '@vben/constants';

import { HOME_PATH } from '../policy';

const fallbackNotFoundRoute: RouteRecordRaw = {
  component: () => import('#/views/_core/fallback/not-found.vue'),
  meta: {
    hideInBreadcrumb: true,
    hideInMenu: true,
    hideInTab: true,
    title: '404',
  },
  name: 'FallbackNotFound',
  path: '/:path(.*)*',
};

const coreRoutes: RouteRecordRaw[] = [
  {
    children: [],
    component: () => import('#/layouts/basic.vue'),
    meta: { hideInBreadcrumb: true, title: 'Stars-Admin' },
    name: 'Root',
    path: '/',
    redirect: HOME_PATH,
  },
  {
    children: [
      {
        component: () => import('#/views/_core/authentication/login.vue'),
        meta: { title: '登录' },
        name: 'Login',
        path: 'login',
      },
      {
        component: () =>
          import('#/views/_core/authentication/change-password.vue'),
        meta: { title: '修改密码' },
        name: 'ChangePassword',
        path: 'change-password',
      },
      {
        component: () => import('#/views/_core/authentication/unavailable.vue'),
        meta: { title: '服务暂不可用' },
        name: 'ProfileUnavailable',
        path: 'unavailable',
      },
    ],
    component: () => import('#/layouts/auth.vue'),
    meta: { hideInTab: true, title: '身份验证' },
    name: 'Authentication',
    path: '/auth',
    redirect: LOGIN_PATH,
  },
  {
    component: () => import('#/views/_core/fallback/forbidden.vue'),
    meta: { hideInMenu: true, hideInTab: true, title: '没有访问权限' },
    name: 'Forbidden',
    path: '/403',
  },
];

export { coreRoutes, fallbackNotFoundRoute };
