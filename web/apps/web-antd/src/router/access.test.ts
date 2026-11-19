import type { Profile, ProfileMenu } from '#/api/core/auth';

import { createMemoryHistory, createRouter } from 'vue-router';

import { describe, expect, it } from 'vitest';

import { canAccess, createAccessRoutes, mountAccessRoutes } from './access';
import { authRedirect, CHANGE_PASSWORD_PATH, HOME_PATH } from './policy';
import { generationPage, jobPages, systemPages } from './routes/modules/system';

const { departments, users } = systemPages;
const userMenu: ProfileMenu = {
  children: [],
  component: users.componentKey,
  icon: 'lucide:user-cog',
  id: '9223372036854775805',
  name: '人员维护',
  path: users.legacyPath,
  sort: 3,
  type: 2,
  visible: true,
};
const departmentMenu: ProfileMenu = {
  ...userMenu,
  component: departments.componentKey,
  id: '9223372036854775804',
  name: '组织机构',
  path: departments.path,
};
const profile: Profile = {
  id: '9223372036854775806',
  menus: [
    {
      ...userMenu,
      children: [userMenu, departmentMenu],
      component: null,
      id: '9223372036854775803',
      name: '组织管理',
      path: '/system',
      type: 1,
    },
  ],
  mustChangePassword: false,
  nickname: '测试账号',
  permissions: ['sys:user:list'],
  roleIds: ['23'],
  username: 'admin',
};

function routesFor(account: Profile) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { component: {}, name: 'Root', path: '/' },
      { component: {}, name: 'FallbackNotFound', path: '/:path(.*)*' },
    ],
  });
  const menus = mountAccessRoutes(router, createAccessRoutes(account));
  return { menus, router };
}

describe('应用路由权限', () => {
  it('首登带着管理页地址进入时转到改密页，匿名用户进入登录页', () => {
    expect(authRedirect(systemPages.users.path, true, true)).toBe(
      CHANGE_PASSWORD_PATH,
    );
    expect(authRedirect(CHANGE_PASSWORD_PATH, true, true)).toBeUndefined();
    expect(authRedirect(systemPages.users.path, false, false)).toBe(
      '/auth/login',
    );
    expect(authRedirect(CHANGE_PASSWORD_PATH, true, false)).toBe(HOME_PATH);
  });

  it('精确匹配权限，普通用户无法通过用户名或通配符获得管理权限', () => {
    expect(canAccess(profile, 'sys:user:list')).toBe(true);
    expect(canAccess(profile, 'sys:user:edit')).toBe(false);
    expect(routesFor(profile).router.resolve(departments.path).name).toBe(
      'FallbackNotFound',
    );
    expect(
      routesFor({ ...profile, permissions: ['sys:*'] }).router.hasRoute(
        users.name,
      ),
    ).toBe(false);
    expect(
      routesFor({ ...profile, roleIds: ['1'] }).router.hasRoute(
        departments.name,
      ),
    ).toBe(true);
    expect(
      canAccess(
        { ...profile, mustChangePassword: true, roleIds: ['1'] },
        'sys:user:list',
      ),
    ).toBe(false);
  });

  it('用后端授权树生成 Vben 菜单，兼容旧路径并使用后端标题和图标', () => {
    const { menus, router } = routesFor(profile);
    expect(router.resolve(users.path).name).toBe(users.name);
    const group = menus.find((menu) => menu.path === '/system');
    expect(group?.name).toBe('组织管理');
    expect(group?.children).toMatchObject([
      {
        icon: 'lucide:user-cog',
        name: '人员维护',
        path: users.path,
      },
    ]);
    expect(
      routesFor({ ...profile, menus: [] }).router.hasRoute(users.name),
    ).toBe(false);
  });

  it('隐藏页面保留直达权限，未知组件和保留路径不能注册', () => {
    const hidden = { ...userMenu, path: '/system/staff', visible: false };
    const { menus, router } = routesFor({ ...profile, menus: [hidden] });
    expect(router.resolve('/system/staff').name).toBe(users.name);
    expect(menus.some((menu) => menu.path === '/system/staff')).toBe(false);
    for (const node of [
      { ...userMenu, component: '/_core/profile/index' },
      { ...userMenu, path: departments.path },
      { ...userMenu, path: '/auth/login' },
      { ...userMenu, path: '//example.com' },
    ]) {
      expect(
        routesFor({ ...profile, menus: [node] }).router.hasRoute(users.name),
      ).toBe(false);
    }
    expect(
      createAccessRoutes({ ...profile, mustChangePassword: true }),
    ).toEqual([]);
  });

  it('普通角色即使持有管理模块菜单与动作码也无法访问管理页', () => {
    const { menus, roles } = systemPages;
    const nodes = [menus, roles].map((page, index) => ({
      ...userMenu,
      component: null,
      id: String(index + 1),
      name: page.title,
      path: page.legacyPath,
    }));
    const account = {
      ...profile,
      menus: nodes,
      permissions: [menus.permission, roles.permission],
    };
    const ordinary = routesFor(account).router;
    expect(ordinary.resolve(menus.path).name).toBe('FallbackNotFound');
    expect(ordinary.resolve(roles.path).name).toBe('FallbackNotFound');
    const admin = routesFor({ ...account, roleIds: ['1'] }).router;
    expect(admin.resolve(menus.path).name).toBe(menus.name);
    expect(admin.resolve(roles.path).name).toBe(roles.name);
  });

  it('按文件动作和管理员角色分别开放新页面，并限制页面路由命名空间', () => {
    const { files } = systemPages;
    const menu = (
      page: typeof files | typeof generationPage | typeof jobPages.tasks,
      id: string,
    ) => ({
      ...userMenu,
      component: null,
      id,
      path: page.legacyPath,
    });
    const menus = [
      menu(files, '10'),
      menu(jobPages.tasks, '11'),
      menu(generationPage, '12'),
    ];
    const account = {
      ...profile,
      menus,
      permissions: ['sys:file:remove', 'job:task:list', 'gen:table:list'],
    };
    const ordinary = routesFor(account).router;
    expect(ordinary.resolve(files.path).name).toBe(files.name);
    expect(ordinary.resolve(jobPages.tasks.path).name).toBe('FallbackNotFound');
    expect(ordinary.resolve(generationPage.path).name).toBe('FallbackNotFound');
    const admin = { ...account, roleIds: ['1'] };
    expect(routesFor(admin).router.resolve(jobPages.tasks.path).name).toBe(
      jobPages.tasks.name,
    );
    expect(routesFor(admin).router.resolve(generationPage.path).name).toBe(
      generationPage.name,
    );
    const misplaced = {
      ...menu(jobPages.tasks, '11'),
      component: jobPages.tasks.componentKey,
      path: '/system/misplaced',
    };
    expect(
      routesFor({ ...admin, menus: [misplaced] }).router.hasRoute(
        jobPages.tasks.name,
      ),
    ).toBe(false);
  });
});
