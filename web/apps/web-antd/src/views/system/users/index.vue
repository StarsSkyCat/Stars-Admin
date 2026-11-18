<script setup lang="ts">
import type { ActionItem } from '@vben/common-ui';

import type { VbenFormProps } from '#/adapter/form';
import type { VxeGridProps } from '#/adapter/vxe-table';
import type { Department } from '#/api/system/departments';
import type { User, UserFilters } from '#/api/system/users';

import { computed, onMounted, onUnmounted, ref, shallowRef } from 'vue';

import {
  Page,
  useVbenDrawer,
  useVbenModal,
  VbenTableAction,
} from '@vben/common-ui';

import {
  Alert,
  Button,
  Descriptions,
  DescriptionsItem,
  message,
  Modal,
  Space,
  Tag,
  Upload,
} from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { errorText } from '#/api/errors';
import { getDepartmentOptions } from '#/api/system/departments';
import {
  exportUsers,
  getUser,
  importUsers,
  listUsers,
  removeUsers,
  updateUserStatus,
} from '#/api/system/users';
import { canAccess } from '#/router/access';
import { useAuthStore } from '#/store';

import {
  departmentNames,
  departmentOptions,
  formatDate,
  showError,
  statusOptions,
} from '../shared';
import AccountAction from './account-action.vue';
import UserEditor from './user-editor.vue';

const auth = useAuthStore();
const departments = ref<Department[]>([]);
const names = computed(() => departmentNames(departments.value));
const editor = ref<InstanceType<typeof UserEditor>>();
const accountAction = ref<InstanceType<typeof AccountAction>>();
const view = shallowRef<User>();
const detailError = ref('');
const listError = ref('');
const busy = ref(false);
const importErrors = ref<{ message: string; row: number }[]>([]);
const filters = shallowRef<UserFilters>({});
let activeQuery: AbortController | undefined;
const cancellation = new AbortController();
const allowed = (code: string) => canAccess(auth.profile, `sys:user:${code}`);
const canReadDepartments = computed(() =>
  canAccess(auth.profile, 'sys:dept:list'),
);
const canGrantRoles = computed(
  () =>
    allowed('grant') &&
    allowed('get') &&
    canAccess(auth.profile, 'sys:role:list') &&
    auth.profile?.roleIds.includes('1'),
);
const canGrantPosts = computed(
  () =>
    allowed('grant') &&
    allowed('get') &&
    canAccess(auth.profile, 'sys:post:list') &&
    auth.profile?.roleIds.includes('1'),
);

const [Drawer, drawer] = useVbenDrawer({
  cancelText: '关闭',
  showConfirmButton: false,
});
const [ImportIssues, importIssues] = useVbenModal({ showConfirmButton: false });

const formOptions: VbenFormProps<UserFilters> = {
  schema: [
    { component: 'Input', fieldName: 'username', label: '用户名' },
    { component: 'Input', fieldName: 'nickname', label: '昵称' },
    {
      component: 'Select',
      componentProps: { allowClear: true, options: [...statusOptions] },
      fieldName: 'status',
      label: '状态',
    },
    ...(canReadDepartments.value
      ? [
          {
            component: 'TreeSelect' as const,
            componentProps: { allowClear: true, treeData: [] },
            fieldName: 'deptId' as const,
            label: '部门',
          },
        ]
      : []),
  ],
  submitButtonOptions: { content: '查询' },
  submitOnEnter: true,
};
const gridOptions: VxeGridProps<User> = {
  columns: [
    ...(allowed('remove') ? [{ type: 'checkbox' as const, width: 44 }] : []),
    { field: 'username', minWidth: 120, title: '用户名' },
    { field: 'nickname', minWidth: 110, title: '昵称' },
    {
      field: 'deptId',
      minWidth: 120,
      slots: { default: 'department' },
      title: '部门',
    },
    { field: 'status', slots: { default: 'status' }, title: '状态', width: 80 },
    {
      field: 'createdAt',
      minWidth: 166,
      slots: { default: 'createdAt' },
      title: '创建时间',
    },
    {
      field: 'operation',
      fixed: 'right',
      slots: { default: 'actions' },
      title: '操作',
      width: 154,
    },
  ],
  checkboxConfig: { checkMethod: ({ row }) => row.id !== '1' },
  pagerConfig: { pageSize: 20, pageSizes: [10, 20, 50, 100] },
  proxyConfig: {
    ajax: {
      query: async ({ page }, values: UserFilters) => {
        activeQuery?.abort();
        const controller = new AbortController();
        activeQuery = controller;
        listError.value = '';
        filters.value = {
          deptId: values.deptId || undefined,
          nickname: values.nickname?.trim() || undefined,
          status: values.status,
          username: values.username?.trim() || undefined,
        };
        try {
          return await listUsers(
            {
              ...filters.value,
              page: page.currentPage,
              pageSize: page.pageSize,
            },
            controller.signal,
          );
        } catch (error) {
          if (!controller.signal.aborted) listError.value = errorText(error);
          return { list: [], total: 0 };
        }
      },
    },
    response: { list: 'list', result: 'list', total: 'total' },
  },
  rowConfig: { keyField: 'id' },
  toolbarConfig: { refresh: true },
};

const [Grid, grid] = useVbenVxeGrid<User, UserFilters>({
  formOptions,
  gridOptions,
});

onMounted(async () => {
  if (!canReadDepartments.value) return;
  try {
    departments.value = await getDepartmentOptions(cancellation.signal);
    grid.formApi.updateSchema([
      {
        componentProps: {
          allowClear: true,
          treeData: departmentOptions(departments.value),
        },
        fieldName: 'deptId',
      },
    ]);
  } catch (error) {
    if (!cancellation.signal.aborted) showError(error);
  }
});
onUnmounted(() => {
  cancellation.abort();
  activeQuery?.abort();
});

function refresh() {
  void grid.reload();
}

async function openDetail(row: User) {
  view.value = undefined;
  detailError.value = '';
  drawer.setState({ loading: true }).open();
  try {
    view.value = await getUser(row.id);
  } catch (error) {
    detailError.value = errorText(error);
  } finally {
    drawer.setState({ loading: false });
  }
}

function confirmAction(title: string, operation: () => Promise<unknown>) {
  if (busy.value) return;
  Modal.confirm({
    async onOk() {
      if (busy.value) return;
      busy.value = true;
      try {
        await operation();
        message.success('操作成功');
        refresh();
      } catch (error) {
        showError(error);
        throw new Error('操作失败', { cause: error });
      } finally {
        busy.value = false;
      }
    },
    title,
  });
}

function remove(ids: string[]) {
  if (ids.length === 0 || ids.length > 100) {
    message.warning('请选择 1 到 100 位要删除的用户');
    return;
  }
  confirmAction(`确认删除选中的 ${ids.length} 位用户？`, () =>
    removeUsers(ids),
  );
}

function actions(row: User): ActionItem[] {
  const items: ActionItem[] = [];
  if (allowed('get'))
    items.push({ onClick: () => openDetail(row), text: '详情' });
  if (allowed('edit'))
    items.push({ onClick: () => editor.value?.open(row), text: '编辑' });
  return items;
}

function moreActions(row: User): ActionItem[] {
  const items: ActionItem[] = [];
  if (allowed('edit') && row.id !== '1') {
    const status = row.status === 1 ? 2 : 1;
    const label = status === 1 ? '启用' : '停用';
    items.push({
      onClick: () =>
        confirmAction(`确认${label}该账号？`, () =>
          updateUserStatus(row.id, status),
        ),
      text: label,
    });
  }
  if (allowed('resetPwd'))
    items.push({
      onClick: () => accountAction.value?.open(row, 'password'),
      text: '重置密码',
    });
  if (canGrantRoles.value)
    items.push({
      onClick: () => accountAction.value?.open(row, 'roles'),
      text: '分配角色',
    });
  if (canGrantPosts.value)
    items.push({
      onClick: () => accountAction.value?.open(row, 'posts'),
      text: '分配岗位',
    });
  if (allowed('remove') && row.id !== '1')
    items.push({ danger: true, onClick: () => remove([row.id]), text: '删除' });
  return items;
}

function beforeImport(file: File) {
  if (!/\.xlsx$/i.test(file.name) || file.size > 5 * 1024 * 1024) {
    message.warning('请选择不超过 5 MB 的 XLSX 文件');
    return Upload.LIST_IGNORE;
  }
  void upload(file);
  return Upload.LIST_IGNORE;
}

async function upload(file: File) {
  if (busy.value) return;
  busy.value = true;
  try {
    const result = await importUsers(file);
    if (result.errors?.length) {
      importErrors.value = result.errors;
      importIssues.open();
      return;
    }
    message.success(`已导入 ${result.imported} 位用户`);
    refresh();
  } catch (error) {
    showError(error);
  } finally {
    busy.value = false;
  }
}

async function download() {
  if (busy.value) return;
  busy.value = true;
  try {
    const file = await exportUsers(filters.value);
    const url = URL.createObjectURL(file);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'users.xlsx';
    link.click();
    window.setTimeout(() => URL.revokeObjectURL(url), 1000);
  } catch (error) {
    showError(error);
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <Page title="用户管理">
    <Alert
      v-if="listError"
      :message="listError"
      class="mb-4"
      show-icon
      type="error"
    >
      <template #action>
        <Button size="small" @click="refresh">重试</Button>
      </template>
    </Alert>
    <Grid>
      <template #toolbar-tools>
        <Space wrap>
          <Button v-if="allowed('add')" type="primary" @click="editor?.open()">
            新增用户
          </Button>
          <Button
            v-if="allowed('remove')"
            :disabled="busy"
            danger
            @click="remove(grid.grid.getCheckboxRecords().map((row) => row.id))"
          >
            批量删除
          </Button>
          <Upload
            v-if="allowed('import')"
            :before-upload="beforeImport"
            :show-upload-list="false"
            accept=".xlsx"
          >
            <Button :disabled="busy">导入 Excel</Button>
          </Upload>
          <Button v-if="allowed('export')" :disabled="busy" @click="download">
            导出 Excel
          </Button>
        </Space>
      </template>
      <template #department="{ row }">
        {{ row.deptId ? (names.get(row.deptId) ?? row.deptId) : '—' }}
      </template>
      <template #status="{ row }">
        <Tag :color="row.status === 1 ? 'success' : 'default'">
          {{ row.status === 1 ? '启用' : '停用' }}
        </Tag>
      </template>
      <template #createdAt="{ row }">{{ formatDate(row.createdAt) }}</template>
      <template #actions="{ row }">
        <VbenTableAction
          :actions="actions(row)"
          :dropdown-actions="moreActions(row)"
          align="start"
          more-text="更多"
        />
      </template>
    </Grid>

    <UserEditor
      ref="editor"
      :can-read-departments="canReadDepartments"
      :departments="departments"
      @saved="refresh"
    />
    <AccountAction ref="accountAction" @saved="refresh" />

    <Drawer title="用户详情">
      <Alert v-if="detailError" :message="detailError" show-icon type="error" />
      <Descriptions v-if="view" :column="1" bordered>
        <DescriptionsItem label="用户 ID">{{ view.id }}</DescriptionsItem>
        <DescriptionsItem label="用户名">{{ view.username }}</DescriptionsItem>
        <DescriptionsItem label="昵称">{{ view.nickname }}</DescriptionsItem>
        <DescriptionsItem label="部门">
          {{ view.deptId ? (names.get(view.deptId) ?? view.deptId) : '—' }}
        </DescriptionsItem>
        <DescriptionsItem label="性别">
          {{ ['未知', '男', '女'][view.gender] }}
        </DescriptionsItem>
        <DescriptionsItem label="状态">
          {{ view.status === 1 ? '启用' : '停用' }}
        </DescriptionsItem>
        <DescriptionsItem label="邮箱">
          {{ view.email ?? '—' }}
        </DescriptionsItem>
        <DescriptionsItem label="手机号">
          {{ view.phone ?? '—' }}
        </DescriptionsItem>
        <DescriptionsItem label="角色 ID">
          {{ view.roleIds.join('、') || '—' }}
        </DescriptionsItem>
        <DescriptionsItem label="岗位 ID">
          {{ view.postIds.join('、') || '—' }}
        </DescriptionsItem>
        <DescriptionsItem label="最近登录">
          {{ formatDate(view.loginAt) }}
        </DescriptionsItem>
        <DescriptionsItem label="备注">
          {{ view.remark ?? '—' }}
        </DescriptionsItem>
      </Descriptions>
    </Drawer>
    <ImportIssues title="导入失败">
      <Alert
        class="mb-3"
        message="本次导入未写入任何用户。"
        show-icon
        type="error"
      />
      <ul class="max-h-80 list-disc overflow-auto pl-5">
        <li v-for="(issue, index) in importErrors" :key="index">
          第 {{ issue.row }} 行 · {{ issue.message }}
        </li>
      </ul>
    </ImportIssues>
  </Page>
</template>
