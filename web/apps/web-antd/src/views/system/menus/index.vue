<script setup lang="ts">
import type { TableColumnsType, TableProps } from 'ant-design-vue';

import type { ActionItem } from '@vben/common-ui';

import type { Menu, MenuFilters } from '#/api/system/menus';

import { computed, ref, shallowRef, watch } from 'vue';
import { useRouter } from 'vue-router';

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
  getMenu,
  getMenuOptions,
  getMenuTree,
  listMenus,
  removeMenus,
} from '#/api/system/menus';
import { systemPages } from '#/router/routes/modules/system';

import { formatDate, showError, statusOptions } from '../shared';
import { useSystemList } from '../use-system-list';
import MenuEditor from './menu-editor.vue';

const router = useRouter();
const writing = ref(false);
const expanded = ref<string[]>([]);
const view = shallowRef<Menu>();
const detailError = ref('');
const editor = ref<InstanceType<typeof MenuEditor>>();
const [Drawer, drawer] = useVbenDrawer({
  cancelText: '关闭',
  showConfirmButton: false,
});
const {
  changePage,
  error: listError,
  filters,
  load,
  loading,
  page,
  pageSize,
  rows,
  search,
  selected,
  total,
} = useSystemList<Menu, MenuFilters>(async (values, pagination, signal) => {
  if (values.name || values.status)
    return listMenus({ ...values, ...pagination }, signal);
  return { list: await getMenuTree(signal), total: 0 };
}, {});
const filtered = computed(() => !!(filters.value.name || filters.value.status));
watch(rows, (next, previous) => {
  if (!filtered.value && !previous.some((node) => node.children.length))
    expanded.value = next
      .filter((node) => node.type === 1)
      .map((node) => node.id);
});

const [SearchForm] = useVbenForm<MenuFilters>({
  handleReset: () => search({}),
  handleSubmit: (values) =>
    search({ name: values.name?.trim(), status: values.status }),
  layout: 'horizontal',
  schema: [
    { component: 'Input', fieldName: 'name', label: '菜单名称' },
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
const types = { 1: '目录', 2: '页面', 3: '按钮' } as const;
const columns: TableColumnsType<Menu> = [
  { dataIndex: 'name', key: 'name', title: '菜单名称', width: 225 },
  { dataIndex: 'type', key: 'type', title: '类型', width: 75 },
  { dataIndex: 'path', key: 'path', title: '路由路径', width: 185 },
  { dataIndex: 'perm', key: 'perm', title: '权限标识', width: 185 },
  { dataIndex: 'visible', key: 'visible', title: '侧栏', width: 75 },
  { dataIndex: 'status', key: 'status', title: '状态', width: 75 },
  { dataIndex: 'sort', key: 'sort', title: '排序', width: 70 },
  { key: 'actions', title: '操作', width: 180 },
];
const selection = computed<TableProps<Menu>['rowSelection']>(() => ({
  onChange: (keys) => {
    selected.value = keys.filter(
      (key): key is string => typeof key === 'string',
    );
  },
  selectedRowKeys: selected.value,
}));

async function detail(row: Menu) {
  view.value = undefined;
  detailError.value = '';
  drawer.setState({ loading: true }).open();
  try {
    view.value = await getMenu(row.id);
  } catch (error) {
    detailError.value = errorText(error);
  } finally {
    drawer.setState({ loading: false });
  }
}

async function openEditor(row?: Menu, parent = '0') {
  if (writing.value) return;
  writing.value = true;
  try {
    const [options, current] = await Promise.all([
      getMenuOptions(),
      row && getMenu(row.id),
    ]);
    editor.value?.open(options, current, parent);
  } catch (error) {
    showError(error);
  } finally {
    writing.value = false;
  }
}

async function refreshNavigation() {
  const { hash, name, path, query } = router.currentRoute.value;
  if (name !== systemPages.menus.name) return;
  await router.replace({ force: true, hash, path, query });
}

async function saved() {
  await Promise.all([load(), refreshNavigation()]);
}

function remove(ids: string[]) {
  if (writing.value) return;
  if (ids.length === 0 || ids.length > 100) {
    message.warning('请选择 1 到 100 个要删除的菜单');
    return;
  }
  Modal.confirm({
    content: '含有下级的节点需先删除其下级。',
    async onOk() {
      if (writing.value) return;
      writing.value = true;
      try {
        await removeMenus(ids);
        message.success('菜单已删除');
        page.value = 1;
        await saved();
      } catch (error) {
        showError(error);
        throw new Error('删除菜单失败', { cause: error });
      } finally {
        writing.value = false;
      }
    },
    title: `确认删除选中的 ${ids.length} 个菜单节点？`,
  });
}

function actions(row: Menu): ActionItem[] {
  return [
    { onClick: () => detail(row), text: '详情' },
    { onClick: () => openEditor(row), text: '编辑' },
  ];
}

function moreActions(row: Menu): ActionItem[] {
  return [
    ...(row.type === 3
      ? []
      : [{ onClick: () => openEditor(undefined, row.id), text: '新增下级' }]),
    { danger: true, onClick: () => remove([row.id]), text: '删除' },
  ];
}
</script>

<template>
  <Page title="菜单管理">
    <Card class="mb-4"><SearchForm /></Card>
    <Card>
      <template #extra>
        <Space wrap>
          <Button :disabled="writing" type="primary" @click="openEditor()">
            新增菜单
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
        v-model:expanded-row-keys="expanded"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :pagination="false"
        :row-selection="selection"
        :scroll="{ x: 1070 }"
        row-key="id"
      >
        <template #bodyCell="{ column, record, text }">
          <Tag v-if="column.key === 'type'">
            {{ types[(record as Menu).type] }}
          </Tag>
          <Tag
            v-else-if="column.key === 'status'"
            :color="record.status === 1 ? 'success' : 'default'"
          >
            {{ record.status === 1 ? '启用' : '停用' }}
          </Tag>
          <template v-else-if="column.key === 'visible'">
            {{
              record.type === 3 ? '—' : record.visible === 1 ? '显示' : '隐藏'
            }}
          </template>
          <VbenTableAction
            v-else-if="column.key === 'actions'"
            :actions="actions(record as Menu)"
            :dropdown-actions="moreActions(record as Menu)"
            align="start"
            more-text="更多"
          />
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
    <MenuEditor ref="editor" @saved="saved" />
    <Drawer title="菜单详情">
      <Alert v-if="detailError" :message="detailError" show-icon type="error" />
      <Descriptions v-if="view" :column="1" bordered>
        <DescriptionsItem label="菜单 ID">{{ view.id }}</DescriptionsItem>
        <DescriptionsItem label="上级菜单 ID">
          {{ view.parentId === '0' ? '根目录' : view.parentId }}
        </DescriptionsItem>
        <DescriptionsItem label="名称">{{ view.name }}</DescriptionsItem>
        <DescriptionsItem label="类型">{{ types[view.type] }}</DescriptionsItem>
        <DescriptionsItem label="路径">{{ view.path ?? '—' }}</DescriptionsItem>
        <DescriptionsItem label="组件">
          {{ view.component ?? '—' }}
        </DescriptionsItem>
        <DescriptionsItem label="权限标识">
          {{ view.perm ?? '—' }}
        </DescriptionsItem>
        <DescriptionsItem label="图标">{{ view.icon ?? '—' }}</DescriptionsItem>
        <DescriptionsItem label="排序">{{ view.sort }}</DescriptionsItem>
        <DescriptionsItem label="状态">
          {{ view.status === 1 ? '启用' : '停用' }}
        </DescriptionsItem>
        <DescriptionsItem label="备注">
          {{ view.remark ?? '—' }}
        </DescriptionsItem>
        <DescriptionsItem label="更新时间">
          {{ formatDate(view.updatedAt) }}
        </DescriptionsItem>
      </Descriptions>
    </Drawer>
  </Page>
</template>
