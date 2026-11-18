<script setup lang="ts">
import type { User } from '#/api/system/users';

import { computed, ref, shallowRef } from 'vue';

import { useVbenModal } from '@vben/common-ui';

import { Alert, InputPassword, message, Select } from 'ant-design-vue';

import { validPassword } from '#/api/core/auth';
import { getPostOptions } from '#/api/system/posts';
import { getRoleOptions } from '#/api/system/roles';
import {
  getUser,
  resetUserPassword,
  setUserPosts,
  setUserRoles,
} from '#/api/system/users';

import { showError } from '../shared';

type Action = 'password' | 'posts' | 'roles';
const emit = defineEmits<{ saved: [] }>();
const titles = { password: '重置密码', posts: '分配岗位', roles: '分配角色' };
const target = shallowRef<User>();
const action = ref<Action>('password');
const chosen = ref<string[]>([]);
const choices = ref<{ disabled?: boolean; label: string; value: string }[]>([]);
const password = ref('');
const ready = ref(false);
const busy = ref(false);
let requestId = 0;
const title = computed(
  () => `${titles[action.value]} · ${target.value?.username ?? ''}`,
);

const [Modal, modal] = useVbenModal({
  async onConfirm() {
    if (!target.value || !ready.value || busy.value) return;
    if (action.value === 'password' && !validPassword(password.value)) {
      message.warning('密码至少 12 个字符，最多 72 字节');
      return;
    }
    if (chosen.value.length > 100) {
      message.warning('一次最多分配 100 项');
      return;
    }
    busy.value = true;
    modal.lock();
    try {
      const id = target.value.id;
      if (action.value === 'password')
        await resetUserPassword(id, password.value);
      else if (action.value === 'roles') await setUserRoles(id, chosen.value);
      else await setUserPosts(id, chosen.value);
      password.value = '';
      message.success('操作成功');
      await modal.close();
      emit('saved');
    } catch (error) {
      showError(error);
    } finally {
      busy.value = false;
      modal.unlock();
    }
  },
  onOpenChange(open) {
    if (!open) requestId++;
  },
});

async function open(row: User, kind: Action) {
  const id = ++requestId;
  target.value = row;
  action.value = kind;
  password.value = '';
  chosen.value = [];
  choices.value = [];
  ready.value = kind === 'password';
  modal
    .setState({ confirmDisabled: !ready.value, loading: !ready.value })
    .open();
  if (kind === 'password') return;
  try {
    const [user, available] = await Promise.all([
      getUser(row.id),
      kind === 'roles' ? getRoleOptions() : getPostOptions(),
    ]);
    if (id !== requestId) return;
    const selected = kind === 'roles' ? user.roleIds : user.postIds;
    choices.value = available
      .filter((item) => kind !== 'roles' || item.id !== '1' || row.id === '1')
      .map((item) => ({
        disabled: kind === 'roles' && item.id === '1',
        label: item.name,
        value: item.id,
      }));
    // 仍显示现有的停用关联，避免打开分配弹窗时将其静默丢弃。
    for (const id of selected) {
      if (!choices.value.some((item) => item.value === id)) {
        choices.value.push({ label: `当前不可选（${id}）`, value: id });
      }
    }
    chosen.value = selected;
    ready.value = true;
  } catch (error) {
    if (id === requestId) showError(error);
  } finally {
    if (id === requestId) {
      modal.setState({ confirmDisabled: !ready.value, loading: false });
    }
  }
}

defineExpose({ open });
</script>

<template>
  <Modal :title="title">
    <template v-if="action === 'password'">
      <Alert
        class="mb-4"
        message="用户的现有会话将失效，下次登录必须修改初始密码。"
        show-icon
        type="warning"
      />
      <InputPassword
        v-model:value="password"
        autocomplete="new-password"
        aria-label="新初始密码"
        placeholder="请输入新初始密码"
      />
    </template>
    <Select
      v-else
      v-model:value="chosen"
      :options="choices"
      class="w-full"
      mode="multiple"
      option-filter-prop="label"
      placeholder="请选择"
      show-search
    />
  </Modal>
</template>
