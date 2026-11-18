<script setup lang="ts">
import { Alert, Button } from 'ant-design-vue';

import { useAuthStore } from '#/store';
import PasswordSetting from '#/views/_core/profile/password-setting.vue';

const auth = useAuthStore();
</script>

<template>
  <div>
    <h1 class="mb-4 text-2xl font-semibold">修改密码</h1>
    <Alert
      class="mb-4"
      message="请先修改初始密码，再重新登录。"
      show-icon
      type="warning"
    />
    <PasswordSetting />
    <Button class="mt-4" type="link" @click="auth.logout">退出登录</Button>
  </div>
</template>
