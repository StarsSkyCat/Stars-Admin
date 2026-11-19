<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';
import type { Config, ConfigInput } from '#/api/system/configs';

import { Page } from '@vben/common-ui';

import { Button, message } from 'ant-design-vue';

import { configs, refreshConfigCache } from '#/api/system/configs';

import ResourceTable from '../resource-table.vue';
import { requiredText } from '../resource-view';
import { optionalText, showError } from '../shared';

const columns = [
  { key: 'name', title: '参数名称', value: (row: Config) => row.name },
  { key: 'key', title: '参数键', value: (row: Config) => row.key, width: 250 },
  {
    key: 'value',
    title: '参数值',
    value: (row: Config) => row.value,
    width: 280,
  },
  {
    key: 'kind',
    title: '类型',
    value: (row: Config) => (row.isSystem ? '内置' : '自定义'),
    width: 90,
  },
  {
    detailOnly: true,
    key: 'remark',
    title: '备注',
    value: (row: Config) => row.remark,
  },
];

function fields(row?: Config): VbenFormSchema<ConfigInput>[] {
  return [
    {
      component: 'Input',
      fieldName: 'name',
      label: '参数名称',
      rules: requiredText('参数名称'),
    },
    {
      component: 'Input',
      componentProps: { disabled: row?.isSystem },
      fieldName: 'key',
      label: '参数键',
      rules: requiredText('参数键', 128).regex(
        /^[a-zA-Z][a-zA-Z0-9_.-]*$/,
        '参数键须以字母开头',
      ),
    },
    {
      component: 'Textarea',
      componentProps: { maxlength: 4096, rows: 4 },
      fieldName: 'value',
      label: '参数值',
    },
    {
      component: 'Textarea',
      componentProps: { maxlength: 500 },
      fieldName: 'remark',
      label: '备注',
    },
  ];
}

function input(values: ConfigInput, row?: Config): ConfigInput {
  return {
    key: row?.isSystem ? row.key : values.key.trim(),
    name: values.name.trim(),
    remark: optionalText(values.remark),
    value: values.value ?? '',
  };
}

async function refresh() {
  try {
    await refreshConfigCache();
    message.success('参数缓存已刷新');
  } catch (error) {
    showError(error);
  }
}
</script>

<template>
  <Page title="参数管理">
    <ResourceTable
      :columns="columns"
      :fields="fields"
      :initial-filters="{}"
      :input="input"
      :remove-enabled="
        (row) => !row.isSystem && !row.key.startsWith('sys.account.')
      "
      :resource="configs"
      :values="(row) => row ?? {}"
      title="参数"
    >
      <template #toolbar><Button @click="refresh">刷新缓存</Button></template>
    </ResourceTable>
  </Page>
</template>
