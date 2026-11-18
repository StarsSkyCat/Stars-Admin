import type { Router, RouteRecordRaw } from 'vue-router';

import type { Profile, ProfileMenu } from '#/api/core/auth';

import { generateMenus } from '@vben/utils';

import dashboard from './routes/modules/dashboard';
import { systemPages } from './routes/modules/system';

const registeredPages = Object.values(systemPages);
const systemPath = /^\/system(?:\/[\da-z-]+)+$/;

export function canAccess(profile: null | Profile, permission: string) {
  return (
    !!profile &&
    !profile.mustChangePassword &&
    (profile.roleIds.includes('1') || profile.permissions.includes(permission))
  );
}

export function createAccessRoutes(profile: Profile): RouteRecordRaw[] {
  const usedPaths = new Set<string>();
  const usedNames = new Set<string>();

  function visit(nodes: ProfileMenu[], parentHidden = false): RouteRecordRaw[] {
    return nodes.flatMap((node): RouteRecordRaw[] => {
      const hidden = parentHidden || !node.visible;
      if (node.type === 1) {
        if (!/^\d+$/.test(node.id)) return [];
        const children = visit(node.children, hidden);
        const [fallback] = children;
        if (!fallback) return [];
        const first = children.find((route) => !route.meta?.hideInMenu);
        const path =
          node.path === '/system' && !usedPaths.has('/system')
            ? '/system'
            : `/menu-groups/${node.id}`;
        if (usedPaths.has(path)) return children;
        usedPaths.add(path);
        return [
          {
            children,
            meta: {
              hideInMenu: !first,
              icon: node.icon || 'lucide:folder',
              order: node.sort,
              title: node.name,
            },
            name: `MenuGroup${node.id}`,
            path,
            redirect: first?.path ?? fallback.path,
          },
        ];
      }

      if (node.type !== 2) return [];
      const children = visit(node.children, hidden);
      const declared = registeredPages.find((page) =>
        node.component
          ? node.component === page.componentKey
          : node.path === page.path || node.path === page.legacyPath,
      );
      if (
        !declared ||
        !canAccess(profile, declared.permission) ||
        (declared.adminOnly && !profile.roleIds.includes('1')) ||
        !node.path
      )
        return children;

      const path =
        node.path === declared.legacyPath ? declared.path : node.path;
      const reserved = registeredPages.some(
        (page) =>
          page !== declared && (path === page.path || path === page.legacyPath),
      );
      if (
        !systemPath.test(path) ||
        reserved ||
        usedPaths.has(path) ||
        usedNames.has(declared.name)
      )
        return children;

      usedPaths.add(path);
      usedNames.add(declared.name);
      return [
        {
          component: declared.component,
          meta: {
            hideInMenu: hidden,
            icon: node.icon || declared.icon,
            order: node.sort,
            title: node.name,
          },
          name: declared.name,
          path,
        },
        ...children,
      ];
    });
  }

  return profile.mustChangePassword
    ? []
    : [...dashboard, ...visit(profile.menus ?? [])];
}

export function mountAccessRoutes(router: Router, routes: RouteRecordRaw[]) {
  for (const route of routes) router.addRoute('Root', route);
  return generateMenus(routes, router);
}

export { systemPages };
