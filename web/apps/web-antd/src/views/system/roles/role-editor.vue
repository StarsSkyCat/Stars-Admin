<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';
import type { Role, RoleInput } from '#/api/system/roles';

import { ref, shallowRef } from 'vue';

import { useVbenModal } from '@vben/common-ui';

import { message } from 'ant-design-vue';

import { useVbenForm, z } from '#/adapter/form';
import { createRole, updateRole } from '#/api/system/roles';

import { optionalText, showError } from '../shared';

const emit = defineEmits<{ saved: [] }>();
const editing = shallowRef<Role>();
const busy = ref(false);
const schema: VbenFormSchema<RoleInput>[] = [
  {
    component: 'Input',
    fieldName: 'name',
    label: '角色名称',
    rules: z.string().trim().min(1, '请输入角色名称').max(64),
  },
  {
    component: 'Input',
    componentProps: { maxlength: 64 },
    fieldName: 'code',
    label: '角色编码',
    rules: z
      .string()
      .trim()
      .toLowerCase()
      .regex(
        /^[a-z][a-z0-9_-]{0,63}$/,
        '以字母开头，使用字母、数字、下划线或连字符',
      ),
  },
  {
    component: 'InputNumber',
    componentProps: { max: 2_147_483_647, min: -2_147_483_648, precision: 0 },
    defaultValue: 0,
    fieldName: 'sort',
    label: '排序',
  },
  {
    component: 'Textarea',
    componentProps: { maxlength: 500 },
    fieldName: 'remark',
    label: '备注',
  },
];
const [Form, form] = useVbenForm<RoleInput>({
  commonConfig: { componentProps: { class: 'w-full' } },
  schema,
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
      const data: RoleInput = {
        code:
          editing.value?.id === '1'
            ? 'admin'
            : values.code.trim().toLowerCase(),
        name: values.name.trim(),
        remark: optionalText(values.remark),
        sort: values.sort ?? 0,
      };
      await (editing.value
        ? updateRole(editing.value.id, data)
        : createRole(data));
      message.success('角色已保存');
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
    const row = editing.value;
    form.updateSchema([
      {
        componentProps: { disabled: row?.id === '1', maxlength: 64 },
        fieldName: 'code',
      },
    ]);
    await form.reset();
    if (row)
      await form.setValues({
        code: row.code,
        name: row.name,
        remark: row.remark,
        sort: row.sort,
      });
  },
});

function open(row?: Role) {
  editing.value = row;
  modal.open();
}

defineExpose({ open });
</script>

<template>
  <Modal
    :title="editing ? '编辑角色' : '新增角色'"
    class="w-[min(520px,calc(100vw-24px))]"
  >
    <Form />
  </Modal>
</template>
