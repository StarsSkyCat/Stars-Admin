<script setup lang="ts">
import type {
  GeneratedColumn,
  GenerationInput,
  GenerationTable,
} from '#/api/gen/tables';

import { computed, ref } from 'vue';

import { useVbenModal } from '@vben/common-ui';

import {
  Alert,
  Checkbox,
  Form,
  FormItem,
  Input,
  message,
  Select,
  Table,
} from 'ant-design-vue';

import { saveGenerationTable } from '#/api/gen/tables';

import { showError } from '../../system/shared';

const emit = defineEmits<{ saved: [] }>();
const current = ref<GenerationTable>();
const working = ref(false);
const baseFields = new Set([
  'created_at',
  'created_by',
  'deleted_at',
  'id',
  'remark',
  'updated_at',
  'updated_by',
]);
const isEditable = (row: GeneratedColumn) =>
  !row.removed && !baseFields.has(row.columnName);
const mappedColumns = computed(() =>
  (current.value?.columns ?? [])
    .filter((row) => !row.removed && row.goType === 'model.ID')
    .map((row) => ({ label: row.columnName, value: row.columnName })),
);
const flags = [
  { field: 'isRequired', label: '必填' },
  { field: 'isInsert', label: '新增' },
  { field: 'isEdit', label: '编辑' },
  { field: 'isList', label: '列表' },
  { field: 'isQuery', label: '筛选' },
] as const;
const columns = [
  { dataIndex: 'columnName', key: 'columnName', title: '数据库列', width: 145 },
  { dataIndex: 'dbType', key: 'dbType', title: '数据库类型', width: 120 },
  { key: 'goField', title: 'Go 字段', width: 145 },
  { key: 'jsonField', title: 'JSON 字段', width: 145 },
  { key: 'flags', title: '生成选项', width: 285 },
  { key: 'queryType', title: '筛选方式', width: 105 },
  { key: 'htmlType', title: '控件', width: 115 },
  { key: 'dictType', title: '字典标识', width: 140 },
];
const queryOptions = ['EQ', 'NE', 'LIKE', 'BETWEEN'].map((value) => ({
  label: value,
  value,
}));
const controlOptions = [
  { label: '输入框', value: 'input' },
  { label: '下拉框', value: 'select' },
  { label: '单选', value: 'radio' },
  { label: '日期时间', value: 'datetime' },
];
const scopeOptions = [
  { label: '仅管理员', value: 'admin' },
  { label: '映射部门或归属人', value: 'mapped' },
];
const identifier = /^[a-z][a-z0-9_]*$/;
const changes: Record<string, string> = {
  added: '新增',
  changed: '已变更',
  removed: '已移除',
};

const [Editor, editor] = useVbenModal({
  async onConfirm() {
    const row = current.value;
    if (!row || working.value) return;
    const names = [row.moduleName, row.featureName, row.businessName].map(
      (value) => value.trim(),
    );
    const [moduleName = '', featureName = '', businessName = ''] = names;
    if (!names.every((value) => identifier.test(value))) {
      message.warning(
        '模块、功能和业务名称须以小写字母开头，只能使用小写字母、数字和下划线',
      );
      return;
    }
    if (row.scope.mode === 'deny') {
      message.warning('请选择数据范围');
      return;
    }
    if (
      row.scope.mode === 'mapped' &&
      !row.scope.departmentColumn &&
      !row.scope.ownerColumn
    ) {
      message.warning('请选择至少一个范围映射字段');
      return;
    }
    const scope =
      row.scope.mode === 'mapped'
        ? {
            ...row.scope,
            departmentColumn: row.scope.departmentColumn || '',
            ownerColumn: row.scope.ownerColumn || '',
          }
        : { departmentColumn: '', mode: row.scope.mode, ownerColumn: '' };
    const input: GenerationInput = {
      author: row.author.trim(),
      businessName,
      columns: row.columns.filter(isEditable).map((column) => ({
        columnName: column.columnName,
        dictType: column.dictType?.trim() || null,
        goField: column.goField.trim(),
        htmlType: column.htmlType,
        isEdit: column.isEdit,
        isInsert: column.isInsert,
        isList: column.isList,
        isQuery: column.isQuery,
        isRequired: column.isRequired,
        jsonField: column.jsonField.trim(),
        queryType: column.queryType,
      })),
      featureName,
      genType: 'zip',
      moduleName,
      scope,
    };
    working.value = true;
    editor.lock();
    try {
      await saveGenerationTable(row.id, input);
      message.success('生成配置已保存');
      await editor.close();
      emit('saved');
    } catch (error) {
      showError(error);
    } finally {
      working.value = false;
      editor.unlock();
    }
  },
});

function open(row: GenerationTable) {
  current.value = {
    ...row,
    columns: row.columns.map((column) => ({ ...column })),
    scope: { ...row.scope },
  };
  editor.open();
}

defineExpose({ open });
</script>

<template>
  <Editor
    :title="`编辑生成配置 · ${current?.tableName ?? ''}`"
    class="w-[min(1180px,calc(100vw-24px))]"
  >
    <template v-if="current">
      <Form layout="vertical" class="grid gap-x-4 sm:grid-cols-3">
        <FormItem label="模块名称" required>
          <Input
            v-model:value="current.moduleName"
            :maxlength="64"
            placeholder="模块名称"
          />
        </FormItem>
        <FormItem label="功能名称" required>
          <Input
            v-model:value="current.featureName"
            :maxlength="64"
            placeholder="功能名称"
          />
        </FormItem>
        <FormItem label="业务名称" required>
          <Input
            v-model:value="current.businessName"
            :maxlength="64"
            placeholder="业务名称"
          />
        </FormItem>
        <FormItem label="作者">
          <Input
            v-model:value="current.author"
            :maxlength="64"
            placeholder="作者"
          />
        </FormItem>
        <FormItem label="数据范围" required>
          <Select
            v-model:value="current.scope.mode"
            :options="scopeOptions"
            aria-label="数据范围"
          />
        </FormItem>
        <FormItem label="生成方式">
          <Input value="ZIP 下载" readonly />
        </FormItem>
        <template v-if="current.scope.mode === 'mapped'">
          <FormItem label="部门字段">
            <Select
              v-model:value="current.scope.departmentColumn"
              :options="mappedColumns"
              allow-clear
              aria-label="部门字段"
            />
          </FormItem>
          <FormItem label="归属人字段">
            <Select
              v-model:value="current.scope.ownerColumn"
              :options="mappedColumns"
              allow-clear
              aria-label="归属人字段"
            />
          </FormItem>
        </template>
      </Form>
      <Alert
        message="结构同步后请核对变更列，保存配置后可重新预览。"
        class="mb-3"
        show-icon
        type="info"
      />
      <Table
        :columns="columns"
        :data-source="current.columns"
        :pagination="false"
        :scroll="{ x: 1180, y: 330 }"
        row-key="id"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'columnName'">
            {{ record.columnName }}
            {{
              changes[record.changeKind]
                ? `（${changes[record.changeKind]}）`
                : ''
            }}
          </template>
          <template v-else-if="isEditable(record as GeneratedColumn)">
            <Input
              v-if="column.key === 'goField'"
              v-model:value="record.goField"
              :maxlength="64"
              :aria-label="`${record.columnName} Go 字段`"
            />
            <Input
              v-else-if="column.key === 'jsonField'"
              v-model:value="record.jsonField"
              :maxlength="64"
              :aria-label="`${record.columnName} JSON 字段`"
            />
            <div
              v-else-if="column.key === 'flags'"
              class="flex flex-wrap gap-1"
            >
              <Checkbox
                v-for="flag in flags"
                :key="flag.field"
                v-model:checked="record[flag.field]"
              >
                {{ flag.label }}
              </Checkbox>
            </div>
            <Select
              v-else-if="column.key === 'queryType'"
              v-model:value="record.queryType"
              :options="queryOptions"
              :aria-label="`${record.columnName} 筛选方式`"
              class="w-full"
            />
            <Select
              v-else-if="column.key === 'htmlType'"
              v-model:value="record.htmlType"
              :options="controlOptions"
              :aria-label="`${record.columnName} 控件`"
              class="w-full"
            />
            <Input
              v-else-if="column.key === 'dictType'"
              v-model:value="record.dictType"
              :maxlength="64"
              :aria-label="`${record.columnName} 字典标识`"
            />
          </template>
        </template>
      </Table>
    </template>
  </Editor>
</template>
