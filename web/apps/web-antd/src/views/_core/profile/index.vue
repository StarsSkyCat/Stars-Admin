<script setup lang="ts">
import { ref } from 'vue';

import { Profile } from '@vben/common-ui';
import { useUserStore } from '@vben/stores';

import { Descriptions, DescriptionsItem } from 'ant-design-vue';

import { useAuthStore } from '#/store';

import PasswordSetting from './password-setting.vue';

const user = useUserStore();
const auth = useAuthStore();
const active = ref('basic');
const tabs = [
  { label: '基本资料', value: 'basic' },
  { label: '修改密码', value: 'password' },
];
</script>

<template>
  <Profile
    v-model:model-value="active"
    :tabs="tabs"
    :user-info="user.userInfo"
    title="个人中心"
  >
    <template #content>
      <Descriptions v-if="active === 'basic'" :column="1" bordered>
        <DescriptionsItem label="用户 ID">
          {{ auth.profile?.id }}
        </DescriptionsItem>
        <DescriptionsItem label="用户名">
          {{ auth.profile?.username }}
        </DescriptionsItem>
        <DescriptionsItem label="昵称">
          {{ auth.profile?.nickname || '—' }}
        </DescriptionsItem>
      </Descriptions>
      <PasswordSetting v-else class="max-w-lg" />
    </template>
  </Profile>
</template>
