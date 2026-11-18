<script setup lang="ts">
import { Button, Result, Space } from 'ant-design-vue';

import { HOME_PATH } from '#/router/policy';
import { useAuthStore } from '#/store';

const auth = useAuthStore();
</script>

<template>
  <Result status="500" title="暂时无法加载账户资料">
    <template #extra>
      <Space>
        <Button type="primary" @click="$router.replace(HOME_PATH)">重试</Button>
        <Button @click="auth.logout">退出登录</Button>
      </Space>
    </template>
  </Result>
</template>
