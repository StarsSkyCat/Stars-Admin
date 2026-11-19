<script setup lang="ts">
import type { GeneratedFile, GenerationTable } from '#/api/gen/tables';

import { computed, ref, shallowRef } from 'vue';

import { Page, useVbenDrawer, useVbenModal } from '@vben/common-ui';

import { Alert, Button, Empty, message, Select } from 'ant-design-vue';

import {
  downloadGeneration,
  getGenerationTable,
  importTables,
  listDatabaseTables,
  listGenerationTables,
  previewGeneration,
  removeGenerationTables,
  syncGenerationTable,
} from '#/api/gen/tables';

import ResourceTable from '../../system/resource-table.vue';
import { confirmAction, saveBlob } from '../../system/resource-view';
import { formatDate, showError } from '../../system/shared';
import GenerationEditor from './generation-editor.vue';

const busy = ref(false);
const tableEditor = ref<InstanceType<typeof GenerationEditor>>();
const available = shallowRef<{ label: string; value: string }[]>([]);
const chosen = ref<string[]>([]);
const sources = shallowRef<GeneratedFile[]>([]);
const filename = ref('');
const reloadAfterEdit = shallowRef<() => Promise<void>>();
const reloadAfterImport = shallowRef<() => Promise<void>>();
const fileOptions = computed(() =>
  sources.value.map(({ path }) => ({ label: path, value: path })),
);
const source = computed(() =>
  sources.value.find(({ path }) => path === filename.value),
);
const previewError = ref('');
const resource = {
  get: getGenerationTable,
  list: listGenerationTables,
  remove: removeGenerationTables,
};
const columns = [
  {
    key: 'table',
    title: '数据库表',
    value: (row: GenerationTable) => row.tableName,
    width: 175,
  },
  {
    key: 'comment',
    title: '表说明',
    value: (row: GenerationTable) => row.tableComment,
  },
  {
    key: 'module',
    title: '模块',
    value: (row: GenerationTable) => row.moduleName,
  },
  {
    key: 'feature',
    title: '功能',
    value: (row: GenerationTable) => row.featureName,
  },
  {
    key: 'sync',
    title: '最近同步',
    value: (row: GenerationTable) => formatDate(row.syncedAt),
    width: 170,
  },
  {
    detailOnly: true,
    key: 'business',
    title: '业务名称',
    value: (row: GenerationTable) => row.businessName,
  },
  {
    detailOnly: true,
    key: 'author',
    title: '作者',
    value: (row: GenerationTable) => row.author,
  },
  {
    detailOnly: true,
    key: 'scope',
    title: '数据范围',
    value: (row: GenerationTable) =>
      ({ admin: '仅管理员', deny: '待配置', mapped: '已映射' })[row.scope.mode],
  },
  {
    detailOnly: true,
    key: 'changes',
    title: '有变更的字段',
    value: (row: GenerationTable) =>
      row.columns.filter((column) => column.changeKind !== 'same').length,
  },
];
const [Preview, drawer] = useVbenDrawer({
  cancelText: '关闭',
  showConfirmButton: false,
});
const [Import, modal] = useVbenModal({
  async onConfirm() {
    if (chosen.value.length === 0 || busy.value) return;
    if (chosen.value.length > 100) {
      message.warning('一次最多导入 100 张表');
      return;
    }
    busy.value = true;
    modal.lock();
    try {
      await importTables(chosen.value);
      await modal.close();
      await reloadAfterImport.value?.();
      message.success('表结构已导入');
    } catch (error) {
      showError(error);
    } finally {
      busy.value = false;
      modal.unlock();
    }
  },
});

async function openImport(reload: () => Promise<void>) {
  if (busy.value) return;
  busy.value = true;
  try {
    const tables = await listDatabaseTables();
    available.value = tables.map((table) => ({
      label: table.comment ? `${table.name}（${table.comment}）` : table.name,
      value: table.name,
    }));
    chosen.value = [];
    reloadAfterImport.value = reload;
    modal.open();
  } catch (error) {
    showError(error);
  } finally {
    busy.value = false;
  }
}

async function edit(row: GenerationTable, reload: () => Promise<void>) {
  if (busy.value) return;
  busy.value = true;
  try {
    const detail = await getGenerationTable(row.id);
    reloadAfterEdit.value = reload;
    tableEditor.value?.open(detail);
  } catch (error) {
    showError(error);
  } finally {
    busy.value = false;
  }
}

async function preview(row: GenerationTable) {
  sources.value = [];
  filename.value = '';
  previewError.value = '';
  drawer.setState({ loading: true }).open();
  try {
    const files = await previewGeneration(row.id);
    sources.value = files;
    filename.value = files[0]?.path ?? '';
  } catch (error) {
    previewError.value =
      error instanceof Error ? error.message : '无法预览，请检查生成配置';
  } finally {
    drawer.setState({ loading: false });
  }
}

async function zip(row: GenerationTable) {
  if (busy.value) return;
  if (row.genType !== 'zip') {
    message.warning('请先保存为 ZIP 生成配置');
    return;
  }
  busy.value = true;
  try {
    saveBlob(await downloadGeneration(row.id), `${row.tableName}.zip`);
  } catch (error) {
    showError(error);
  } finally {
    busy.value = false;
  }
}

function actions(row: GenerationTable, reload: () => Promise<void>) {
  return [
    { onClick: () => edit(row, reload), text: '编辑配置' },
    {
      onClick: () =>
        confirmAction(
          `确认同步「${row.tableName}」的表结构？`,
          async () => {
            await syncGenerationTable(row.id);
            await reload();
          },
          '同步完成，请核对并重新保存配置',
        ),
      text: '同步结构',
    },
    { onClick: () => preview(row), text: '预览代码' },
    { onClick: () => zip(row), text: '下载 ZIP' },
  ];
}
</script>

<template>
  <Page title="代码生成">
    <ResourceTable
      :columns="columns"
      :extra-actions="actions"
      :initial-filters="{}"
      :resource="resource"
      title="生成配置"
    >
      <template #toolbar="{ reload }">
        <Button :loading="busy" type="primary" @click="openImport(reload)">
          导入数据表
        </Button>
      </template>
    </ResourceTable>
    <GenerationEditor ref="tableEditor" @saved="reloadAfterEdit?.()" />
    <Import title="导入数据库表">
      <Select
        v-if="available.length"
        v-model:value="chosen"
        :options="available"
        aria-label="选择数据库表"
        class="w-full"
        mode="multiple"
        placeholder="选择可导入的数据库表"
        show-search
      />
      <Empty v-else description="当前没有可导入的数据库表" />
    </Import>
    <Preview title="代码预览" class="w-[min(880px,calc(100vw-24px))]">
      <Alert
        v-if="previewError"
        :message="previewError"
        show-icon
        type="error"
      />
      <template v-else-if="sources.length">
        <Select
          v-model:value="filename"
          :options="fileOptions"
          aria-label="预览文件"
          class="mb-4 w-full"
        />
        <pre
          v-if="source"
          class="bg-muted overflow-auto rounded-md p-4 text-xs"
          >{{ source.content }}</pre>
      </template>
    </Preview>
  </Page>
</template>
