import type { LoginInput, PasswordInput, Profile } from '#/api/core/auth';

import { onScopeDispose, ref, shallowRef } from 'vue';
import { useRouter } from 'vue-router';

import { LOGIN_PATH } from '@vben/constants';
import { preferences } from '@vben/preferences';
import { useAccessStore, useTabbarStore, useUserStore } from '@vben/stores';

import { message } from 'ant-design-vue';
import { defineStore } from 'pinia';

import {
  changePassword,
  loginApi,
  logoutApi,
  profileApi,
} from '#/api/core/auth';
import { ApiError } from '#/api/errors';
import { resetRoutes } from '#/router';
import { CHANGE_PASSWORD_PATH, HOME_PATH } from '#/router/policy';

import { sessions } from './session';

export const useAuthStore = defineStore('auth', () => {
  const user = useUserStore();
  const access = useAccessStore();
  const tabs = useTabbarStore();
  const router = useRouter();
  const current = shallowRef(sessions.read());
  const profile = shallowRef<null | Profile>(null);
  const loginLoading = ref(false);
  let pendingProfile: undefined | { id: string; result: Promise<Profile> };

  // 旧模板持久化的 Access 和权限码不参与会话恢复。
  access.$reset();

  function clearIdentity() {
    profile.value = null;
    user.$reset();
    access.$reset();
    tabs.$reset();
    resetRoutes();
  }

  const unsubscribe = sessions.subscribe((next) => {
    const previous = current.value;
    current.value = next;
    const changed = previous?.id !== next?.id;
    const restricted =
      !previous?.mustChangePassword && next?.mustChangePassword;
    if (changed || restricted) clearIdentity();
    const route = router.currentRoute.value;
    if (!route.name) return;
    if (!next && changed && route.path !== LOGIN_PATH) {
      void router.replace(LOGIN_PATH);
    } else if (next?.mustChangePassword && (changed || restricted)) {
      void router.replace(CHANGE_PASSWORD_PATH);
    } else if (next && changed) {
      void router.replace(HOME_PATH);
    }
  });
  onScopeDispose(unsubscribe);

  async function fetchProfile() {
    const session = sessions.read();
    if (!session) throw new ApiError('会话已失效，请重新登录', 401, 2000);
    if (pendingProfile?.id === session.id) return pendingProfile.result;

    const result = (async () => {
      const data = await profileApi();
      if (sessions.read()?.id !== session.id) {
        throw new ApiError('会话已失效，请重新登录', 401, 2000);
      }
      if (data.mustChangePassword)
        await sessions.requirePasswordChange(session.id);
      if (sessions.read()?.id !== session.id) {
        throw new ApiError('会话已失效，请重新登录', 401, 2000);
      }
      profile.value = data;
      access.setAccessCodes(data.permissions);
      user.setUserInfo({
        avatar: preferences.app.defaultAvatar,
        realName: data.nickname || data.username,
        userId: data.id,
        username: data.username,
      });
      return data;
    })();
    pendingProfile = { id: session.id, result };
    try {
      return await result;
    } finally {
      if (pendingProfile?.result === result) pendingProfile = undefined;
    }
  }

  async function authLogin(input: LoginInput) {
    if (loginLoading.value) return;
    loginLoading.value = true;
    try {
      const tokens = await loginApi(input);
      const id = Array.from(
        crypto.getRandomValues(new Uint8Array(16)),
        (byte) => byte.toString(16).padStart(2, '0'),
      ).join('');
      await sessions.start({ ...tokens, id });
      await router.replace(
        tokens.mustChangePassword ? CHANGE_PASSWORD_PATH : HOME_PATH,
      );
    } finally {
      loginLoading.value = false;
    }
  }

  async function logout() {
    const session = sessions.read();
    if (!session) {
      await router.replace(LOGIN_PATH);
      return;
    }
    const ended = await sessions.clear(session.id);
    if (!ended) return;
    try {
      await logoutApi(ended.accessToken);
    } catch {
      message.warning('本地已退出，暂无法确认服务端会话已撤销');
    }
  }

  async function updatePassword(input: PasswordInput) {
    const session = sessions.read();
    if (!session) return;
    await changePassword(input);
    await sessions.clear(session.id);
    message.success('密码已修改，请重新登录');
  }

  return {
    authLogin,
    current,
    fetchProfile,
    loginLoading,
    logout,
    profile,
    updatePassword,
  };
});
