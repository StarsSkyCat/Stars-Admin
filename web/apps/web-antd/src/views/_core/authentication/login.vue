<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';

import type { Captcha } from '#/api/core/auth';

import {
  computed,
  markRaw,
  onMounted,
  onUnmounted,
  ref,
  shallowRef,
} from 'vue';

import { AuthenticationLogin, z } from '@vben/common-ui';

import { Alert, Button, message } from 'ant-design-vue';

import { getCaptcha } from '#/api/core/auth';
import { ApiError, errorText } from '#/api/errors';
import { useAuthStore } from '#/store';

import CaptchaInput from './captcha-input.vue';

// 离开登录页时销毁验证码和密码输入，重新进入时取新验证码。
defineOptions({ name: 'StarsLogin' });

const auth = useAuthStore();
const captcha = shallowRef<Captcha>();
const form = ref<InstanceType<typeof AuthenticationLogin>>();
const loading = ref(false);
const captchaError = ref('');
let expiresAt = 0;
const cancellation = new AbortController();

async function loadCaptcha() {
  if (loading.value) return;
  loading.value = true;
  captchaError.value = '';
  expiresAt = 0;
  try {
    if (captcha.value?.enabled)
      await form.value?.getFormApi().setFieldValue('captchaCode', '');
    captcha.value = await getCaptcha(cancellation.signal);
    expiresAt = Date.now() + captcha.value.expiresIn * 1000;
  } catch (error) {
    if (!cancellation.signal.aborted) captchaError.value = errorText(error);
  } finally {
    loading.value = false;
  }
}

const schema = computed<VbenFormSchema[]>(() => {
  const fields: VbenFormSchema[] = [
    {
      component: 'VbenInput',
      componentProps: { autocomplete: 'username', placeholder: '请输入用户名' },
      fieldName: 'username',
      label: '用户名',
      rules: z.string().trim().min(1, '请输入用户名').max(64, '用户名过长'),
    },
    {
      component: 'VbenInputPassword',
      componentProps: {
        autocomplete: 'current-password',
        placeholder: '请输入密码',
      },
      fieldName: 'password',
      label: '密码',
      rules: z.string().min(1, '请输入密码'),
    },
  ];
  if (captcha.value?.enabled) {
    fields.push({
      component: markRaw(CaptchaInput),
      componentProps: {
        image: captcha.value.image,
        loading: loading.value,
        onRefresh: loadCaptcha,
      },
      fieldName: 'captchaCode',
      label: '图片验证码',
      modelPropName: 'value',
      rules: z
        .string()
        .trim()
        .regex(/^\d{6}$/, '请输入 6 位数字验证码'),
    });
  }
  return fields;
});

async function submit(values: Record<string, unknown>) {
  if (loading.value || auth.loginLoading) return;
  if (!captcha.value || captchaError.value) {
    message.warning('请先获取验证码状态');
    return;
  }
  if (captcha.value.enabled && Date.now() >= expiresAt) {
    message.warning('验证码已过期，请输入新的验证码');
    await loadCaptcha();
    return;
  }
  try {
    await auth.authLogin({
      captchaCode: captcha.value.enabled
        ? String(values.captchaCode).trim()
        : undefined,
      captchaUuid: captcha.value.enabled ? captcha.value.uuid : undefined,
      password: String(values.password),
      username: String(values.username).trim(),
    });
  } catch (error) {
    message.error(errorText(error));
    if (error instanceof ApiError && error.status === 429) {
      expiresAt = 0;
      return;
    }
    await loadCaptcha();
  }
}

onMounted(loadCaptcha);
onUnmounted(() => cancellation.abort());
</script>

<template>
  <div>
    <Alert
      v-if="captchaError"
      :message="captchaError"
      class="mb-4"
      show-icon
      type="error"
    >
      <template #action>
        <Button size="small" @click="loadCaptcha">重试</Button>
      </template>
    </Alert>
    <AuthenticationLogin
      ref="form"
      :form-schema="schema"
      :loading="loading || auth.loginLoading"
      :show-code-login="false"
      :show-forget-password="false"
      :show-qrcode-login="false"
      :show-register="false"
      :show-third-party-login="false"
      @submit="submit"
    />
  </div>
</template>
