import type { Router } from 'vue-router';

import { preferences } from '@vben/preferences';
import { useAccessStore, useTabbarStore } from '@vben/stores';
import { startProgress, stopProgress } from '@vben/utils';

import { ApiError } from '#/api/errors';
import { useAuthStore } from '#/store';
import { sessions } from '#/store/session';

import { createAccessRoutes, mountAccessRoutes } from './access';
import {
  authRedirect,
  CHANGE_PASSWORD_PATH,
  HOME_PATH,
  PROFILE_ERROR_PATH,
} from './policy';

function createRouterGuard(router: Router, resetRoutes: () => void) {
  let mountedRoutes = '';
  router.beforeEach(async (to) => {
    if (preferences.transition.progress) void startProgress();
    const auth = useAuthStore();
    const access = useAccessStore();
    const session = sessions.read();
    if (!session) return authRedirect(to.path, false, false);
    if (to.path === PROFILE_ERROR_PATH) return true;

    let profile;
    try {
      profile = await auth.fetchProfile();
    } catch (error) {
      if (error instanceof ApiError && error.code === 2006) {
        await sessions.clear(session.id);
      }
      const current = sessions.read();
      if (!current) return authRedirect(to.path, false, false) ?? true;
      if (current.id !== session.id)
        return current.mustChangePassword ? CHANGE_PASSWORD_PATH : HOME_PATH;
      if (error instanceof ApiError && error.code === 2008)
        return authRedirect(to.path, true, true) ?? true;
      return PROFILE_ERROR_PATH;
    }

    const current = sessions.read();
    if (current?.id !== session.id) return false;
    const restricted = current.mustChangePassword || profile.mustChangePassword;
    const redirect = authRedirect(to.path, true, restricted);
    if (redirect) return redirect;
    if (restricted) return true;

    const routes = createAccessRoutes(profile);
    const snapshot = JSON.stringify(routes);
    const changed = !access.isAccessChecked || mountedRoutes !== snapshot;
    if (changed) {
      // 同步替换路由，避免旧会话的异步装配晚于新会话。
      resetRoutes();
      access.setAccessMenus(mountAccessRoutes(router, routes));
      access.setAccessRoutes(routes);
      access.setIsAccessChecked(true);
      useTabbarStore().$reset();
      mountedRoutes = snapshot;
    }
    const resolved = router.resolve(to.fullPath);
    if (
      resolved.name === 'FallbackNotFound' &&
      /^\/(?:system|menu-groups)(?:\/|$)/i.test(to.path)
    ) {
      return { name: 'Forbidden' };
    }
    if (changed) {
      return { ...resolved, replace: true };
    }
    return true;
  });
  router.afterEach(() => {
    if (preferences.transition.progress) void stopProgress();
  });
  router.onError(() => {
    if (preferences.transition.progress) void stopProgress();
  });
}

export { createRouterGuard };
