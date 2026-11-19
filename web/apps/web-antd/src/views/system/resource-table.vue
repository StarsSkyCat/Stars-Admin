<script
  setup
  lang="ts"
  generic="
    Row extends ResourceRow,
    Input extends object,
    Filters extends object
  "
>
import type { TableProps } from 'ant-design-vue';

import type { ActionItem, ExtendedFormApi } from '@vben/common-ui';

import type { ResourceColumn } from './resource-view';

import type { VbenFormSchema } from '#/adapter/form';
import type { ResourceRow } from '#/api/resource';
import type { Page, PageQuery } from '#/api/system/shared';

import { computed, ref, shallowRef } from 'vue';

import { useVbenDrawer, useVbenModal, VbenTableAction } from '@vben/common-ui';

import {
  Alert,
  Button,
  Card,
  Descriptions,
  DescriptionsItem,
  message,
  Pagination,
  Space,
  Table,
} from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { errorText } from '#/api/errors';

import { confirmAction } from './resource-view';
import { showError } from './shared';
import { useSystemList } from './use-system-list';

const props = defineProps<{
  columns: ResourceColumn<Row>[];
  extraActions?: (row: Row, reload: () => Promise<void>) => ActionItem[];
  fields?: (row?: Row) => VbenFormSchema<Input>[];
  initialFilters: Filters;
  input?: (values: Input, row?: Row) => Input;
  removeEnabled?: (row: Row) => boolean;
  resource: {
    create?: (data: Input) => Promise<unknown>;
    get?: (id: string) => Promise<Row>;
    list: (
      params: Filters & PageQuery,
      signal?: AbortSignal,
    ) => Promise<Page<Row>>;
    remove?: (ids: string[]) => Promise<unknown>;
    update?: (id: string, data: Input) => Promise<unknown>;
  };
  searchFields?: VbenFormSchema<Filters>[];
  title: string;
  values?: (row?: Row) => Parameters<ExtendedFormApi<Input>['setValues']>[0];
}>();

const emit = defineEmits<{ saved: [] }>();
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
} = useSystemList<Row, Filters>(
  (values, pagination, signal) =>
    props.resource.list({ ...values, ...pagination }, signal),
  props.initialFilters,
);
const view = shallowRef<Row>();
const editing = shallowRef<Row>();
const detailError = ref('');
const writing = ref(false);
let detailRequest = 0;
const [Drawer, drawer] = useVbenDrawer({
  cancelText: '关闭',
  showConfirmButton: false,
});
const [SearchForm] = useVbenForm<Filters>({
  handleReset: () => search(props.initialFilters),
  handleSubmit: (values) => search({ ...props.initialFilters, ...values }),
  layout: 'horizontal',
  schema: props.searchFields ?? [],
  submitButtonOptions: { content: '查询' },
  submitOnEnter: true,
  wrapperClass: 'grid-cols-1 sm:grid-cols-3',
});
const [EditForm, form] = useVbenForm<Input>({
  commonConfig: { componentProps: { class: 'w-full' } },
  schema: props.fields?.() ?? [],
  showDefaultActions: false,
});
const [Editor, editor] = useVbenModal({
  async onConfirm() {
    if (writing.value || !props.input) return;
    writing.value = true;
    editor.lock();
    try {
      if (!(await form.validate()).valid) return;
      const row = editing.value;
      const input = props.input(await form.getValues(), row);
      if (row) await props.resource.update?.(row.id, input);
      else await props.resource.create?.(input);
      message.success(`${props.title}已保存`);
      await editor.close();
      emit('saved');
      await load();
    } catch (error) {
      showError(error);
    } finally {
      writing.value = false;
      editor.unlock();
    }
  },
  async onOpenChange(open) {
    if (!open) return;
    const row = editing.value;
    form.setState({ schema: props.fields?.(row) ?? [] });
    await form.reset();
    const values = props.values?.(row);
    if (values) await form.setValues(values);
  },
});

const tableColumns = computed(() => [
  ...props.columns
    .filter((column) => !column.detailOnly)
    .map(({ key, title, width }) => ({ key, title, width: width ?? 150 })),
  { key: 'actions', title: '操作', width: 180 },
]);
const selection = computed<TableProps<Row>['rowSelection']>(() =>
  props.resource.remove
    ? {
        getCheckboxProps: (row) => ({
          disabled: props.removeEnabled?.(row) === false,
        }),
        onChange: (keys) => {
          selected.value = keys.filter(
            (key): key is string => typeof key === 'string',
          );
        },
        selectedRowKeys: selected.value,
      }
    : undefined,
);

async function openDetail(row: Row) {
  const current = ++detailRequest;
  detailError.value = '';
  view.value = undefined;
  drawer.setState({ loading: true }).open();
  try {
    const result = props.resource.get ? await props.resource.get(row.id) : row;
    if (current === detailRequest) view.value = result;
  } catch (error) {
    if (current === detailRequest) detailError.value = errorText(error);
  } finally {
    if (current === detailRequest) drawer.setState({ loading: false });
  }
}

async function openEditor(row?: Row) {
  if (writing.value) return;
  writing.value = true;
  try {
    editing.value =
      row && props.resource.get ? await props.resource.get(row.id) : row;
    editor.open();
  } catch (error) {
    showError(error);
  } finally {
    writing.value = false;
  }
}

function remove(ids: string[]) {
  if (!ids.length || ids.length > 100) {
    message.warning('请选择 1 到 100 条记录');
    return;
  }
  confirmAction(
    `确认删除选中的 ${ids.length} 条${props.title}？`,
    async () => {
      await props.resource.remove?.(ids);
      page.value = 1;
      emit('saved');
      await load();
    },
    '删除成功',
  );
}

function actions(row: Row): ActionItem[] {
  return [
    { onClick: () => openDetail(row), text: '详情' },
    ...(props.resource.update && props.fields
      ? [{ onClick: () => openEditor(row), text: '编辑' }]
      : []),
  ];
}

function moreActions(row: Row): ActionItem[] {
  const items = props.extraActions?.(row, load) ?? [];
  if (props.resource.remove && props.removeEnabled?.(row) !== false)
    items.push({ danger: true, onClick: () => remove([row.id]), text: '删除' });
  return items;
}
</script>

<template>
  <Card v-if="searchFields?.length" class="mb-4"><SearchForm /></Card>
  <Card>
    <template #extra>
      <Space wrap>
        <Button
          v-if="resource.create && fields"
          :disabled="writing"
          type="primary"
          @click="openEditor()"
        >
          新增{{ title }}
        </Button>
        <Button
          v-if="resource.remove"
          :disabled="selected.length === 0"
          danger
          @click="remove(selected)"
        >
          批量删除
        </Button>
        <slot name="toolbar" :filters="filters" :reload="load"></slot>
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
      :columns="tableColumns"
      :data-source="rows"
      :loading="loading"
      :pagination="false"
      :row-selection="selection"
      :scroll="{ x: 900 }"
      row-key="id"
    >
      <template #bodyCell="{ column, record }">
        <VbenTableAction
          v-if="column.key === 'actions'"
          :actions="actions(record as Row)"
          :dropdown-actions="moreActions(record as Row)"
          align="start"
          more-text="更多"
        />
        <span v-else class="break-all">
          {{
            columns
              .find(({ key }) => key === column.key)
              ?.value(record as Row) ?? '—'
          }}
        </span>
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
  <Editor
    :title="`${editing ? '编辑' : '新增'}${title}`"
    class="w-[min(620px,calc(100vw-24px))]"
  >
    <EditForm />
  </Editor>
  <Drawer :title="`${title}详情`">
    <Alert v-if="detailError" :message="detailError" show-icon type="error" />
    <Descriptions v-if="view" :column="1" bordered>
      <DescriptionsItem label="ID">{{ view.id }}</DescriptionsItem>
      <DescriptionsItem
        v-for="column in columns"
        :key="column.key"
        :label="column.title"
      >
        <span class="whitespace-pre-wrap break-all">{{
          column.value(view) ?? '—'
        }}</span>
      </DescriptionsItem>
    </Descriptions>
  </Drawer>
</template>
