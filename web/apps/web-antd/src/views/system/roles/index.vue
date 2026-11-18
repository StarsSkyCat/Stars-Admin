<script setup lang="ts">
import type { TableColumnsType, TableProps } from 'ant-design-vue';

import type { ActionItem } from '@vben/common-ui';

import type { Role, RoleFilters } from '#/api/system/roles';

import { computed, ref, shallowRef } from 'vue';

import { Page, useVbenDrawer, VbenTableAction } from '@vben/common-ui';

import {
  Alert,
  Button,
  Card,
  Descriptions,
  DescriptionsItem,
  message,
  Modal,
  Pagination,
  Space,
  Table,
  Tag,
} from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { errorText } from '#/api/errors';
import {
  getRole,
  listRoles,
  removeRoles,
  updateRoleStatus,
} from '#/api/system/roles';

import {
  dataScopeOptions,
  formatDate,
  showError,
  statusOptions,
} from '../shared';
import { useSystemList } from '../use-system-list';
import RoleAccess from './role-access.vue';
import RoleEditor from './role-editor.vue';

const view = shallowRef<Role>();
const detailError = ref('');
const writing = ref(false);
const editor = ref<InstanceType<typeof RoleEditor>>();
const authorization = ref<InstanceType<typeof RoleAccess>>();
const [Drawer, drawer] = useVbenDrawer({
  cancelText: '关闭',
  showConfirmButton: false,
});
const {
  changePage,
  error: listError,
  load,
  loading,
  page,
  pageSize,
  rows,
  search,
  selected,
  total,
} = useSystemList<Role, RoleFilters>(
  (filters, pagination, signal) =>
    listRoles({ ...filters, ...pagination }, signal),
  {},
);
const [SearchForm] = useVbenForm<RoleFilters>({
  handleReset: () => search({}),
  handleSubmit: (values) =>
    search({
      code: values.code?.trim(),
      name: values.name?.trim(),
      status: values.status,
    }),
  layout: 'horizontal',
  schema: [
    { component: 'Input', fieldName: 'name', label: '角色名称' },
    { component: 'Input', fieldName: 'code', label: '角色编码' },
    {
      component: 'Select',
      componentProps: { allowClear: true, options: [...statusOptions] },
      fieldName: 'status',
      label: '状态',
    },
  ],
  submitButtonOptions: { content: '查询' },
  submitOnEnter: true,
  wrapperClass: 'grid-cols-1 sm:grid-cols-4',
});
const columns: TableColumnsType<Role> = [
  { dataIndex: 'name', key: 'name', title: '角色名称', width: 195 },
  { dataIndex: 'code', key: 'code', title: '角色编码', width: 175 },
  { dataIndex: 'dataScope', key: 'dataScope', title: '数据范围', width: 140 },
  { dataIndex: 'status', key: 'status', title: '状态', width: 80 },
  { dataIndex: 'sort', key: 'sort', title: '排序', width: 70 },
  { dataIndex: 'createdAt', key: 'createdAt', title: '创建时间', width: 175 },
  { key: 'actions', title: '操作', width: 190 },
];
const selection = computed<TableProps<Role>['rowSelection']>(() => ({
  getCheckboxProps: (row) => ({ disabled: row.id === '1' }),
  onChange: (keys) => {
    selected.value = keys.filter(
      (key): key is string => typeof key === 'string' && key !== '1',
    );
  },
  selectedRowKeys: selected.value,
}));

function scopeLabel(role: Role) {
  return (
    dataScopeOptions.find(({ value }) => value === role.dataScope)?.label ?? '—'
  );
}

async function detail(row: Role) {
  view.value = undefined;
  detailError.value = '';
  drawer.setState({ loading: true }).open();
  try {
    view.value = await getRole(row.id);
  } catch (error) {
    detailError.value = errorText(error);
  } finally {
    drawer.setState({ loading: false });
  }
}

async function openEditor(row?: Role) {
  if (writing.value) return;
  writing.value = true;
  try {
    editor.value?.open(row ? await getRole(row.id) : undefined);
  } catch (error) {
    showError(error);
  } finally {
    writing.value = false;
  }
}

function confirmChange(
  title: string,
  apply: () => Promise<unknown>,
  success: string,
  content?: string,
) {
  if (writing.value) return;
  Modal.confirm({
    content,
    async onOk() {
      if (writing.value) return;
      writing.value = true;
      try {
        await apply();
        message.success(success);
        await load();
      } catch (error) {
        showError(error);
        throw new Error('角色操作失败', { cause: error });
      } finally {
        writing.value = false;
      }
    },
    title,
  });
}

function remove(ids: string[]) {
  if (ids.length === 0 || ids.length > 100) {
    message.warning('请选择 1 到 100 个要删除的角色');
    return;
  }
  confirmChange(
    `确认删除选中的 ${ids.length} 个角色？`,
    async () => {
      await removeRoles(ids);
      page.value = 1;
    },
    '角色已删除',
    '仍有用户关联的角色无法删除。',
  );
}

function actions(row: Role): ActionItem[] {
  return [
    { onClick: () => detail(row), text: '详情' },
    { onClick: () => openEditor(row), text: '编辑' },
  ];
}

function moreActions(row: Role): ActionItem[] {
  if (row.id === '1') return [];
  const next = row.status === 1 ? 2 : 1;
  const label = next === 1 ? '启用' : '停用';
  return [
    {
      onClick: () => authorization.value?.open(row.id, 'menus'),
      text: '菜单授权',
    },
    {
      onClick: () => authorization.value?.open(row.id, 'scope'),
      text: '数据范围',
    },
    {
      danger: next === 2,
      onClick: () =>
        confirmChange(
          `确认${label}角色「${row.name}」？`,
          () => updateRoleStatus(row.id, next),
          `角色已${label}`,
          '关联账号的会话将失效。',
        ),
      text: label,
    },
    { danger: true, onClick: () => remove([row.id]), text: '删除' },
  ];
}
</script>

<template>
  <Page title="角色管理">
    <Card class="mb-4"><SearchForm /></Card>
    <Card>
      <template #extra>
        <Space wrap>
          <Button :disabled="writing" type="primary" @click="openEditor()">
            新增角色
          </Button>
          <Button
            :disabled="writing || selected.length === 0"
            danger
            @click="remove(selected)"
          >
            批量删除
          </Button>
          <Button :loading="loading" @click="load">刷新</Button>
        </Space>
      </template>
      <Alert
        v-if="listError"
        :message="listError"
        class="mb-4"
        show-icon
        type="error"
      />
      <Table
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :pagination="false"
        :row-selection="selection"
        :scroll="{ x: 1025 }"
        row-key="id"
      >
        <template #bodyCell="{ column, record, text }">
          <Tag
            v-if="column.key === 'status'"
            :color="record.status === 1 ? 'success' : 'default'"
          >
            {{ record.status === 1 ? '启用' : '停用' }}
          </Tag>
          <template v-else-if="column.key === 'dataScope'">
            {{ scopeLabel(record as Role) }}
          </template>
          <template v-else-if="column.key === 'createdAt'">
            {{ formatDate(record.createdAt) }}
          </template>
          <VbenTableAction
            v-else-if="column.key === 'actions'"
            :actions="actions(record as Role)"
            :dropdown-actions="moreActions(record as Role)"
            align="start"
            more-text="更多"
          />
          <template v-else>{{ text ?? '—' }}</template>
        </template>
      </Table>
      <Pagination
        :current="page"
        :page-size="pageSize"
        :page-size-options="['10', '20', '50', '100']"
        :total="total"
        class="mt-4 justify-end"
        show-size-changer
        @change="changePage"
      />
    </Card>
    <RoleEditor ref="editor" @saved="load" />
    <RoleAccess ref="authorization" @saved="load" />
    <Drawer title="角色详情">
      <Alert v-if="detailError" :message="detailError" show-icon type="error" />
      <Descriptions v-if="view" :column="1" bordered>
        <DescriptionsItem label="角色 ID">{{ view.id }}</DescriptionsItem>
        <DescriptionsItem label="名称">{{ view.name }}</DescriptionsItem>
        <DescriptionsItem label="编码">{{ view.code }}</DescriptionsItem>
        <DescriptionsItem label="数据范围">
          {{ scopeLabel(view) }}
        </DescriptionsItem>
        <DescriptionsItem label="已授权菜单和按钮">
          {{ view.menuIds.length }}
        </DescriptionsItem>
        <DescriptionsItem v-if="view.dataScope === 2" label="指定部门">
          {{ view.deptIds.length }}
        </DescriptionsItem>
        <DescriptionsItem label="排序">{{ view.sort }}</DescriptionsItem>
        <DescriptionsItem label="状态">
          {{ view.status === 1 ? '启用' : '停用' }}
        </DescriptionsItem>
        <DescriptionsItem label="备注">
          {{ view.remark ?? '—' }}
        </DescriptionsItem>
      </Descriptions>
    </Drawer>
  </Page>
</template>
