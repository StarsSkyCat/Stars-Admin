<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';
import type { Department } from '#/api/system/departments';
import type { User, UserInput } from '#/api/system/users';

import { computed, ref } from 'vue';

import { useVbenModal } from '@vben/common-ui';

import { message } from 'ant-design-vue';

import { useVbenForm, z } from '#/adapter/form';
import { validPassword } from '#/api/core/auth';
import { createUser, updateUser } from '#/api/system/users';

import { departmentOptions, optionalText, showError } from '../shared';

const props = defineProps<{
  canReadDepartments: boolean;
  departments: Department[];
}>();
const emit = defineEmits<{ saved: [] }>();
const user = ref<User>();
const busy = ref(false);
const title = computed(() => (user.value ? '编辑用户' : '新增用户'));
type FormValues = Omit<UserInput, 'deptId'> & {
  deptId?: null | string;
  password?: string;
};

const schema = computed<VbenFormSchema<FormValues>[]>(() => {
  const fields: VbenFormSchema<FormValues>[] = [
    {
      component: 'Input',
      componentProps: { disabled: user.value?.id === '1' },
      fieldName: 'username',
      label: '用户名',
      rules: z
        .string()
        .trim()
        .regex(
          /^[a-z\d][a-z\d_.@-]{0,63}$/i,
          '用户名以字母或数字开头，最多 64 个字符',
        ),
    },
    {
      component: 'Input',
      fieldName: 'nickname',
      label: '昵称',
      rules: z.string().trim().min(1, '请输入昵称').max(64),
    },
    {
      component: 'TreeSelect',
      componentProps: {
        allowClear: true,
        disabled: !props.canReadDepartments,
        treeData: departmentOptions(props.departments),
      },
      fieldName: 'deptId',
      label: '部门',
    },
    {
      component: 'Select',
      componentProps: {
        options: [
          { label: '未知', value: 0 },
          { label: '男', value: 1 },
          { label: '女', value: 2 },
        ],
      },
      defaultValue: 0,
      fieldName: 'gender',
      label: '性别',
    },
    {
      component: 'Input',
      componentProps: { maxlength: 128 },
      fieldName: 'email',
      label: '邮箱',
    },
    {
      component: 'Input',
      componentProps: { maxlength: 20 },
      fieldName: 'phone',
      label: '手机号',
    },
    {
      component: 'Input',
      componentProps: { maxlength: 255 },
      fieldName: 'avatar',
      label: '头像地址',
    },
    {
      component: 'Textarea',
      componentProps: { maxlength: 500 },
      fieldName: 'remark',
      label: '备注',
    },
  ];
  if (!user.value) {
    fields.splice(2, 0, {
      component: 'VbenInputPassword',
      componentProps: { autocomplete: 'new-password' },
      fieldName: 'password',
      label: '初始密码',
      rules: z
        .string()
        .refine(validPassword, '密码至少 12 个字符，最多 72 字节'),
    });
  }
  return fields;
});

const [Form, form] = useVbenForm<FormValues>({
  commonConfig: { componentProps: { class: 'w-full' } },
  schema: schema.value,
  showDefaultActions: false,
});
const [Modal, modal] = useVbenModal({
  async onConfirm() {
    if (busy.value) return;
    busy.value = true;
    modal.lock();
    try {
      const { valid } = await form.validate();
      if (!valid) return;
      const values = await form.getValues();
      const data: UserInput = {
        avatar: optionalText(values.avatar),
        deptId: props.canReadDepartments
          ? (values.deptId ?? null)
          : (user.value?.deptId ?? null),
        email: optionalText(values.email),
        gender: values.gender ?? 0,
        nickname: values.nickname.trim(),
        phone: optionalText(values.phone),
        remark: optionalText(values.remark),
        username: values.username.trim(),
      };
      await (user.value
        ? updateUser(user.value.id, data)
        : createUser({ ...data, password: values.password ?? '' }));
      message.success('用户已保存');
      await modal.close();
      emit('saved');
    } catch (error) {
      showError(error);
    } finally {
      busy.value = false;
      modal.unlock();
    }
  },
  async onOpenChange(open) {
    if (!open) return;
    form.setState({ schema: schema.value });
    await form.reset();
    const existing = user.value;
    if (existing) {
      await form.setValues({
        avatar: existing.avatar,
        deptId: existing.deptId,
        email: existing.email,
        gender: existing.gender,
        nickname: existing.nickname,
        phone: existing.phone,
        remark: existing.remark,
        username: existing.username,
      });
    }
  },
});

function open(row?: User) {
  user.value = row;
  modal.open();
}

defineExpose({ open });
</script>

<template>
  <Modal :title="title" class="w-[min(560px,calc(100vw-24px))]">
    <Form />
  </Modal>
</template>
