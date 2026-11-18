<script lang="ts" setup>
import { computed, watch } from 'vue';
import { useRouter } from 'vue-router';

import { useWatermark } from '@vben/hooks';
import { BasicLayout, LockScreen, UserDropdown } from '@vben/layouts';
import { preferences, usePreferences } from '@vben/preferences';
import { useUserStore } from '@vben/stores';

import { useAuthStore } from '#/store';

const router = useRouter();
const user = useUserStore();
const auth = useAuthStore();
const avatar = computed(
  () => user.userInfo?.avatar ?? preferences.app.defaultAvatar,
);
const menus = [
  {
    handler: () => router.push({ name: 'Profile' }),
    icon: 'lucide:user',
    text: '个人中心',
  },
];
const { destroyWatermark, updateWatermark } = useWatermark();
const { isDark } = usePreferences();

watch(
  () => ({
    content: preferences.app.watermarkContent,
    enable: preferences.app.watermark,
    isDark: isDark.value,
    username: user.userInfo?.username,
  }),
  async ({ content, enable, isDark: dark, username }) => {
    if (!enable) {
      destroyWatermark();
      return;
    }
    await updateWatermark({
      advancedStyle: {
        colorStops: [
          { color: dark ? '#ffffff22' : '#00000022', offset: 0 },
          { color: dark ? '#ffffff22' : '#00000022', offset: 1 },
        ],
        type: 'linear',
      },
      content: content || username || 'Stars-Admin',
    });
  },
  { immediate: true },
);
</script>

<template>
  <BasicLayout
    :avatar="avatar"
    :text="user.userInfo?.realName"
    @clear-preferences-and-logout="auth.logout"
    @logout="auth.logout"
  >
    <template #user-dropdown>
      <UserDropdown
        :avatar="avatar"
        :description="user.userInfo?.username"
        :menus="menus"
        :text="user.userInfo?.realName"
        @clear-preferences-and-logout="auth.logout"
        @logout="auth.logout"
      />
    </template>
    <template #lock-screen>
      <LockScreen :avatar="avatar" @to-login="auth.logout" />
    </template>
  </BasicLayout>
</template>
