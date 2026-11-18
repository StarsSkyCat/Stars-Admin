<script lang="ts" setup>
import { VbenButton } from '@vben/common-ui';

import { Input } from 'ant-design-vue';

defineOptions({ inheritAttrs: false });
defineProps<{ image: string; loading: boolean }>();
const emit = defineEmits<{ refresh: [] }>();
const value = defineModel<string>('value');
</script>

<template>
  <div
    class="grid w-full grid-cols-[minmax(0,1fr)_min(7.5rem,44%)] items-center gap-2"
  >
    <Input
      v-bind="$attrs"
      v-model:value="value"
      aria-label="图片验证码"
      autocomplete="off"
      class="text-sm"
      inputmode="numeric"
      :maxlength="6"
      placeholder="请输入图片验证码"
      size="large"
    />
    <VbenButton
      aria-label="换一张验证码"
      class="h-10 w-full overflow-hidden p-0"
      :loading="loading"
      title="换一张验证码"
      type="button"
      variant="outline"
      @click="emit('refresh')"
    >
      <img
        v-if="image && !loading"
        :src="image"
        alt="图片验证码"
        class="block h-full w-full object-contain"
        height="40"
        width="120"
      />
      <span v-else-if="!loading">获取验证码</span>
    </VbenButton>
  </div>
</template>
