<script setup lang="ts">
import type { WorkbenchQuickNavItem } from '@vben/common-ui';

import { computed } from 'vue';
import { useRouter } from 'vue-router';

import { Page, WorkbenchHeader, WorkbenchQuickNav } from '@vben/common-ui';
import { useAccessStore, useUserStore } from '@vben/stores';

import { systemPages } from '#/router/access';
import { useAuthStore } from '#/store';

const user = useUserStore();
const auth = useAuthStore();
const access = useAccessStore();
const router = useRouter();
const shortcuts = computed<WorkbenchQuickNavItem[]>(() => {
  const items: WorkbenchQuickNavItem[] = [
    {
      color: '#722ed1',
      icon: 'lucide:user',
      title: '个人中心',
      url: '/profile',
    },
  ];
  if (access.accessMenus.length === 0) return items;
  for (const [page, color] of [
    [systemPages.users, '#1677ff'],
    [systemPages.departments, '#389e0d'],
    [systemPages.menus, '#08979c'],
    [systemPages.roles, '#d46b08'],
  ] as const) {
    if (!router.hasRoute(page.name)) continue;
    const url = router.resolve({ name: page.name }).path;
    const menu = access.getMenuByPath(url);
    if (menu)
      items.push({
        color,
        icon: menu.icon || page.icon,
        title: menu.name,
        url,
      });
  }
  return items;
});
</script>

<template>
  <Page>
    <WorkbenchHeader :avatar="user.userInfo?.avatar">
      <template #title>欢迎，{{ user.userInfo?.realName }}</template>
      <template #description>{{ auth.profile?.username }}</template>
    </WorkbenchHeader>
    <WorkbenchQuickNav
      :items="shortcuts"
      class="mt-4"
      title="快捷入口"
      @click="(item) => item.url && $router.push(item.url)"
    />
  </Page>
</template>
