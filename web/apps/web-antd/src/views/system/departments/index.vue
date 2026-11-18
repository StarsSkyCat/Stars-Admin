<script setup lang="ts">
import type { TableColumnsType, TableProps } from 'ant-design-vue';

import type { ActionItem } from '@vben/common-ui';

import type { Department } from '#/api/system/departments';
import type { Status } from '#/api/system/shared';

import { computed, onMounted, onUnmounted, ref, shallowRef } from 'vue';

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
  getDepartment,
  getDepartmentOptions,
  getDepartmentTree,
  listDepartments,
  removeDepartments,
} from '#/api/system/departments';
import { canAccess } from '#/router/access';
import { useAuthStore } from '#/store';

import {
  departmentNames,
  formatDate,
  showError,
  statusOptions,
} from '../shared';
import DepartmentEditor from './department-editor.vue';

interface Search {
  name?: string;
  status?: Status;
}

const auth = useAuthStore();
const allowed = (code: string) => canAccess(auth.profile, `sys:dept:${code}`);
const view = shallowRef<Department>();
const rows = shallowRef<Department[]>([]);
const options = shallowRef<Department[]>([]);
const names = computed(() => departmentNames(options.value));
const selected = ref<string[]>([]);
const listError = ref('');
const detailError = ref('');
const loading = ref(false);
const writing = ref(false);
const page = ref(1);
const pageSize = ref(20);
const total = ref(0);
const filters = shallowRef<Search>({});
const filtered = computed(() => !!(filters.value.name || filters.value.status));
const editor = ref<InstanceType<typeof DepartmentEditor>>();
let activeQuery: AbortController | undefined;
const cancellation = new AbortController();
const [Drawer, drawer] = useVbenDrawer({
  cancelText: '关闭',
  showConfirmButton: false,
});

const columns: TableColumnsType<Department> = [
  { dataIndex: 'name', key: 'name', title: '部门名称', width: 240 },
  { dataIndex: 'sort', key: 'sort', title: '排序', width: 80 },
  { dataIndex: 'leader', key: 'leader', title: '负责人', width: 120 },
  { dataIndex: 'phone', key: 'phone', title: '联系电话', width: 140 },
  { dataIndex: 'status', key: 'status', title: '状态', width: 80 },
  { dataIndex: 'createdAt', key: 'createdAt', title: '创建时间', width: 180 },
  { key: 'actions', title: '操作', width: 186 },
];
const rowSelection = computed<TableProps<Department>['rowSelection']>(() =>
  allowed('remove')
    ? {
        onChange: (keys) => {
          selected.value = keys.filter(
            (key): key is string => typeof key === 'string',
          );
        },
        selectedRowKeys: selected.value,
      }
    : undefined,
);

const [SearchForm] = useVbenForm<Search>({
  async handleReset() {
    filters.value = {};
    page.value = 1;
    await load();
  },
  async handleSubmit(values) {
    filters.value = { name: values.name?.trim(), status: values.status };
    page.value = 1;
    await load();
  },
  layout: 'horizontal',
  schema: [
    { component: 'Input', fieldName: 'name', label: '部门名称' },
    {
      component: 'Select',
      componentProps: { allowClear: true, options: [...statusOptions] },
      fieldName: 'status',
      label: '状态',
    },
  ],
  submitButtonOptions: { content: '查询' },
  submitOnEnter: true,
  wrapperClass: 'grid-cols-1 sm:grid-cols-3',
});

async function load() {
  activeQuery?.abort();
  const controller = new AbortController();
  activeQuery = controller;
  loading.value = true;
  listError.value = '';
  selected.value = [];
  try {
    if (filtered.value) {
      const result = await listDepartments(
        { ...filters.value, page: page.value, pageSize: pageSize.value },
        controller.signal,
      );
      if (controller.signal.aborted) return;
      total.value = result.total;
      rows.value = result.list;
    } else {
      const result = await getDepartmentTree(controller.signal);
      if (controller.signal.aborted) return;
      rows.value = result;
      total.value = 0;
    }
  } catch (error) {
    if (!controller.signal.aborted) {
      rows.value = [];
      total.value = 0;
      listError.value = errorText(error);
    }
  } finally {
    if (!controller.signal.aborted) loading.value = false;
  }
}

async function refreshOptions() {
  try {
    options.value = await getDepartmentOptions(cancellation.signal);
    return true;
  } catch (error) {
    if (!cancellation.signal.aborted) showError(error);
    return false;
  }
}

onMounted(load);
onUnmounted(() => {
  activeQuery?.abort();
  cancellation.abort();
});

function changePage(next: number, size: number) {
  page.value = size === pageSize.value ? next : 1;
  pageSize.value = size;
  void load();
}

async function openDetail(row: Department) {
  detailError.value = '';
  view.value = undefined;
  drawer.setState({ loading: true }).open();
  try {
    view.value = await getDepartment(row.id);
  } catch (error) {
    detailError.value = errorText(error);
  } finally {
    drawer.setState({ loading: false });
  }
}

async function openEditor(row?: Department, parentId = '0') {
  if (writing.value) return;
  writing.value = true;
  try {
    if (!(await refreshOptions())) return;
    // 范围过滤后的树可能把不可见父部门折叠为根节点，编辑前读取真实详情。
    const current = row ? await getDepartment(row.id) : undefined;
    editor.value?.open(current, parentId);
  } catch (error) {
    showError(error);
  } finally {
    writing.value = false;
  }
}

function remove(ids: string[]) {
  if (writing.value) return;
  if (ids.length === 0 || ids.length > 100) {
    message.warning('请选择 1 到 100 个要删除的部门');
    return;
  }
  Modal.confirm({
    async onOk() {
      if (writing.value) return;
      writing.value = true;
      try {
        await removeDepartments(ids);
        message.success('部门已删除');
        page.value = 1;
        await load();
      } catch (error) {
        showError(error);
        throw new Error('删除失败', { cause: error });
      } finally {
        writing.value = false;
      }
    },
    title: `确认删除选中的 ${ids.length} 个部门？`,
  });
}

function actions(row: Department): ActionItem[] {
  const items: ActionItem[] = [];
  if (allowed('get'))
    items.push({ onClick: () => openDetail(row), text: '详情' });
  if (allowed('edit') && allowed('get'))
    items.push({ onClick: () => openEditor(row), text: '编辑' });
  return items;
}

function moreActions(row: Department): ActionItem[] {
  const items: ActionItem[] = [];
  if (allowed('add'))
    items.push({
      onClick: () => openEditor(undefined, row.id),
      text: '新增下级',
    });
  if (allowed('remove'))
    items.push({ danger: true, onClick: () => remove([row.id]), text: '删除' });
  return items;
}
</script>

<template>
  <Page title="部门管理">
    <Card class="mb-4">
      <SearchForm />
    </Card>
    <Card>
      <template #extra>
        <Space wrap>
          <Button
            v-if="allowed('add')"
            :disabled="writing"
            type="primary"
            @click="openEditor()"
          >
            新增部门
          </Button>
          <Button
            v-if="allowed('remove')"
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
        :row-selection="rowSelection"
        :scroll="{ x: 1000 }"
        row-key="id"
      >
        <template #bodyCell="{ column, record, text }">
          <template v-if="column.key === 'status'">
            <Tag :color="record.status === 1 ? 'success' : 'default'">
              {{ record.status === 1 ? '启用' : '停用' }}
            </Tag>
          </template>
          <template v-else-if="column.key === 'createdAt'">
            {{ formatDate(record.createdAt) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <VbenTableAction
              :actions="actions(record as Department)"
              :dropdown-actions="moreActions(record as Department)"
              align="start"
              more-text="更多"
            />
          </template>
          <template v-else>{{ text ?? '—' }}</template>
        </template>
      </Table>
      <Pagination
        v-if="filtered"
        :current="page"
        :page-size="pageSize"
        :page-size-options="['10', '20', '50', '100']"
        :total="total"
        class="mt-4 justify-end"
        show-size-changer
        @change="changePage"
      />
    </Card>

    <DepartmentEditor ref="editor" :departments="options" @saved="load" />
    <Drawer title="部门详情">
      <Alert v-if="detailError" :message="detailError" show-icon type="error" />
      <Descriptions v-if="view" :column="1" bordered>
        <DescriptionsItem label="部门 ID">{{ view.id }}</DescriptionsItem>
        <DescriptionsItem label="名称">{{ view.name }}</DescriptionsItem>
        <DescriptionsItem label="上级">
          {{
            view.parentId === '0'
              ? '根部门'
              : (names.get(view.parentId) ?? view.parentId)
          }}
        </DescriptionsItem>
        <DescriptionsItem label="排序">{{ view.sort }}</DescriptionsItem>
        <DescriptionsItem label="负责人">
          {{ view.leader ?? '—' }}
        </DescriptionsItem>
        <DescriptionsItem label="电话">
          {{ view.phone ?? '—' }}
        </DescriptionsItem>
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
