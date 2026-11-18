<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';

import { ref } from 'vue';

import { ProfilePasswordSetting, z } from '@vben/common-ui';

import { message, Spin } from 'ant-design-vue';

import { validPassword } from '#/api/core/auth';
import { errorText } from '#/api/errors';
import { useAuthStore } from '#/store';

const auth = useAuthStore();
const busy = ref(false);
const formSchema: VbenFormSchema[] = [
  {
    component: 'VbenInputPassword',
    componentProps: { autocomplete: 'current-password' },
    fieldName: 'oldPassword',
    label: '当前密码',
    rules: z.string().min(1, '请输入当前密码'),
  },
  {
    component: 'VbenInputPassword',
    componentProps: { autocomplete: 'new-password', passwordStrength: true },
    fieldName: 'newPassword',
    label: '新密码',
    rules: z
      .string()
      .refine(validPassword, '新密码至少 12 个字符，最多 72 字节'),
  },
  {
    component: 'VbenInputPassword',
    componentProps: { autocomplete: 'new-password' },
    dependencies: {
      rules: (values) =>
        z
          .string()
          .min(1, '请再次输入新密码')
          .refine(
            (value) => value === values.newPassword,
            '两次输入的密码不一致',
          ),
      triggerFields: ['newPassword'],
    },
    fieldName: 'confirmPassword',
    label: '确认密码',
  },
];

async function submit(values: Record<string, unknown>) {
  if (busy.value) return;
  busy.value = true;
  try {
    await auth.updatePassword({
      newPassword: String(values.newPassword),
      oldPassword: String(values.oldPassword),
    });
  } catch (error) {
    message.error(errorText(error));
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <Spin :spinning="busy">
    <ProfilePasswordSetting :form-schema="formSchema" @submit="submit" />
  </Spin>
</template>
