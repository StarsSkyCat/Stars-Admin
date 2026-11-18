<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';
import type { Department, DepartmentInput } from '#/api/system/departments';

import { computed, ref } from 'vue';

import { useVbenModal } from '@vben/common-ui';

import { message } from 'ant-design-vue';

import { useVbenForm, z } from '#/adapter/form';
import { createDepartment, updateDepartment } from '#/api/system/departments';

import {
  departmentNames,
  departmentOptions,
  optionalText,
  showError,
  statusOptions,
} from '../shared';

const props = defineProps<{ departments: Department[] }>();
const emit = defineEmits<{ saved: [] }>();
const editing = ref<Department>();
const busy = ref(false);
const initialParent = ref('0');
const schema = computed<VbenFormSchema<DepartmentInput>[]>(() => {
  const tree = departmentOptions(props.departments, editing.value?.id);
  const names = departmentNames(props.departments);
  const unavailable =
    editing.value &&
    editing.value.parentId !== '0' &&
    !names.has(editing.value.parentId)
      ? [
          {
            label: `当前上级（${editing.value.parentId}）`,
            value: editing.value.parentId,
          },
        ]
      : [];
  return [
    {
      component: 'TreeSelect',
      componentProps: {
        treeData: [
          { children: tree, label: '根部门', value: '0' },
          ...unavailable,
        ],
      },
      defaultValue: '0',
      fieldName: 'parentId',
      label: '上级部门',
      rules: 'selectRequired',
    },
    {
      component: 'Input',
      fieldName: 'name',
      label: '部门名称',
      rules: z.string().trim().min(1, '请输入名称').max(64),
    },
    {
      component: 'InputNumber',
      componentProps: { precision: 0 },
      defaultValue: 0,
      fieldName: 'sort',
      label: '排序',
    },
    {
      component: 'Input',
      componentProps: { maxlength: 64 },
      fieldName: 'leader',
      label: '负责人',
    },
    {
      component: 'Input',
      componentProps: { maxlength: 20 },
      fieldName: 'phone',
      label: '联系电话',
    },
    {
      component: 'Select',
      componentProps: { options: [...statusOptions] },
      defaultValue: 1,
      fieldName: 'status',
      label: '状态',
      rules: 'selectRequired',
    },
    {
      component: 'Textarea',
      componentProps: { maxlength: 500 },
      fieldName: 'remark',
      label: '备注',
    },
  ];
});
const [Form, form] = useVbenForm<DepartmentInput>({
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
      const data: DepartmentInput = {
        leader: optionalText(values.leader),
        name: values.name.trim(),
        parentId: values.parentId,
        phone: optionalText(values.phone),
        remark: optionalText(values.remark),
        sort: values.sort ?? 0,
        status: values.status,
      };
      await (editing.value
        ? updateDepartment(editing.value.id, data)
        : createDepartment(data));
      message.success('部门已保存');
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
    const row = editing.value;
    await form.setValues(
      row
        ? {
            leader: row.leader,
            name: row.name,
            parentId: row.parentId,
            phone: row.phone,
            remark: row.remark,
            sort: row.sort,
            status: row.status,
          }
        : { parentId: initialParent.value },
    );
  },
});

function open(row?: Department, parentId = '0') {
  editing.value = row;
  initialParent.value = parentId;
  modal.open();
}

defineExpose({ open });
</script>

<template>
  <Modal
    :title="editing ? '编辑部门' : '新增部门'"
    class="w-[min(560px,calc(100vw-24px))]"
  >
    <Form />
  </Modal>
</template>
