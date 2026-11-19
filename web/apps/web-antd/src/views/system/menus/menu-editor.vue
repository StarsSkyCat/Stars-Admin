<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';
import type { Menu, MenuInput } from '#/api/system/menus';

import { ref, shallowRef } from 'vue';

import { useVbenModal } from '@vben/common-ui';

import { message } from 'ant-design-vue';

import { useVbenForm, z } from '#/adapter/form';
import { createMenu, updateMenu } from '#/api/system/menus';
import { registeredPages } from '#/router/routes/modules/system';

import { optionalText, showError, statusOptions } from '../shared';

const emit = defineEmits<{ saved: [] }>();
const editing = shallowRef<Menu>();
const busy = ref(false);
const tree = shallowRef<Menu[]>([]);
const parentId = ref('0');
const menuTypes = [
  { label: '目录', value: 1 },
  { label: '页面', value: 2 },
  { label: '按钮', value: 3 },
];
const components = registeredPages.map((page) => ({
  label: page.title,
  value: page.componentKey,
}));

interface ParentOption {
  children?: ParentOption[];
  label: string;
  value: string;
}

function parentOptions(nodes: Menu[]): ParentOption[] {
  return nodes.flatMap((node) =>
    node.type === 3 || node.id === editing.value?.id
      ? []
      : [
          {
            children: parentOptions(node.children),
            label: node.name,
            value: node.id,
          },
        ],
  );
}

function schema(): VbenFormSchema<MenuInput>[] {
  const current = editing.value?.component;
  const componentOptions =
    current && !components.some((item) => item.value === current)
      ? [...components, { label: current, value: current }]
      : components;
  return [
    {
      component: 'TreeSelect',
      componentProps: {
        treeData: [
          { children: parentOptions(tree.value), label: '根目录', value: '0' },
        ],
        treeDefaultExpandAll: true,
      },
      defaultValue: '0',
      fieldName: 'parentId',
      label: '上级菜单',
      rules: 'selectRequired',
    },
    {
      component: 'Select',
      componentProps: { options: menuTypes },
      defaultValue: 2,
      fieldName: 'type',
      label: '类型',
      rules: 'selectRequired',
    },
    {
      component: 'Input',
      fieldName: 'name',
      label: '菜单名称',
      rules: z.string().trim().min(1, '请输入菜单名称').max(64),
    },
    {
      component: 'Input',
      componentProps: { maxlength: 255, placeholder: '例如 /system/users' },
      dependencies: {
        resolve: ({ values }) => ({ if: values.type !== 3 }),
        triggerFields: ['type'],
      },
      fieldName: 'path',
      label: '路由路径',
    },
    {
      component: 'Select',
      componentProps: {
        allowClear: true,
        options: componentOptions,
        placeholder: '选择已登记的页面组件',
      },
      dependencies: {
        resolve: ({ values }) => ({ if: values.type === 2 }),
        triggerFields: ['type'],
      },
      fieldName: 'component',
      label: '页面组件',
    },
    {
      component: 'IconPicker',
      dependencies: {
        resolve: ({ values }) => ({ if: values.type !== 3 }),
        triggerFields: ['type'],
      },
      fieldName: 'icon',
      label: '图标',
    },
    {
      component: 'Input',
      componentProps: { maxlength: 128, placeholder: '例如 sys:user:list' },
      dependencies: {
        resolve: ({ values }) => ({
          rules:
            values.type === 3
              ? z
                  .string()
                  .trim()
                  .regex(
                    /^[a-z][a-z0-9_-]*:[a-z][a-z0-9_-]*:[a-zA-Z][a-zA-Z0-9_-]*$/,
                    '请输入模块:资源:动作格式的权限码',
                  )
              : undefined,
        }),
        triggerFields: ['type'],
      },
      fieldName: 'perm',
      label: '权限标识',
    },
    {
      component: 'InputNumber',
      componentProps: { max: 2_147_483_647, min: -2_147_483_648, precision: 0 },
      defaultValue: 0,
      fieldName: 'sort',
      label: '排序',
    },
    {
      component: 'Select',
      componentProps: {
        options: [
          { label: '显示', value: 1 },
          { label: '隐藏', value: 2 },
        ],
      },
      defaultValue: 1,
      fieldName: 'visible',
      label: '侧栏显示',
      rules: 'selectRequired',
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
}

const [Form, form] = useVbenForm<MenuInput>({
  commonConfig: { componentProps: { class: 'w-full' } },
  schema: schema(),
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
      const path = values.type === 3 ? null : optionalText(values.path);
      const component =
        values.type === 2 ? optionalText(values.component) : null;
      if (component && !path) {
        message.warning('请选择页面组件对应的路由路径');
        return;
      }
      const input: MenuInput = {
        component,
        icon: values.type === 3 ? null : optionalText(values.icon),
        name: values.name.trim(),
        parentId: values.parentId,
        path,
        perm: optionalText(values.perm),
        remark: optionalText(values.remark),
        sort: values.sort ?? 0,
        status: values.status,
        type: values.type,
        visible: values.visible,
      };
      await (editing.value
        ? updateMenu(editing.value.id, input)
        : createMenu(input));
      message.success('菜单已保存');
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
    form.setState({ schema: schema() });
    await form.reset();
    const row = editing.value;
    await form.setValues(
      row
        ? {
            component: row.component,
            icon: row.icon,
            name: row.name,
            parentId: row.parentId,
            path: row.path,
            perm: row.perm,
            remark: row.remark,
            sort: row.sort,
            status: row.status,
            type: row.type,
            visible: row.visible,
          }
        : { parentId: parentId.value },
    );
  },
});

function open(options: Menu[], row?: Menu, parent = '0') {
  tree.value = options;
  editing.value = row;
  parentId.value = parent;
  modal.open();
}

defineExpose({ open });
</script>

<template>
  <Modal
    :title="editing ? '编辑菜单' : '新增菜单'"
    class="w-[min(600px,calc(100vw-24px))]"
  >
    <Form />
  </Modal>
</template>
