<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';
import type {
  Dictionary,
  DictionaryEntry,
  DictionaryEntryInput,
  DictionaryInput,
} from '#/api/system/dictionaries';

import { shallowRef } from 'vue';

import { Page } from '@vben/common-ui';

import { Button, message } from 'ant-design-vue';

import { z } from '#/adapter/form';
import {
  dictionaries,
  entries,
  refreshDictionaryCache,
} from '#/api/system/dictionaries';

import ResourceTable from '../resource-table.vue';
import { requiredText, statusLabel } from '../resource-view';
import { optionalText, showError, statusOptions } from '../shared';

const current = shallowRef<Dictionary>();
const typeColumns = [
  { key: 'name', title: '字典名称', value: (row: Dictionary) => row.name },
  { key: 'type', title: '字典标识', value: (row: Dictionary) => row.type },
  {
    key: 'status',
    title: '状态',
    value: (row: Dictionary) => statusLabel(row.status),
  },
  { key: 'remark', title: '备注', value: (row: Dictionary) => row.remark },
];
const entryColumns = [
  {
    key: 'label',
    title: '显示名称',
    value: (row: DictionaryEntry) => row.label,
  },
  { key: 'value', title: '字典值', value: (row: DictionaryEntry) => row.value },
  { key: 'tag', title: '标签', value: (row: DictionaryEntry) => row.tag },
  {
    key: 'sort',
    title: '排序',
    value: (row: DictionaryEntry) => row.sort,
    width: 80,
  },
  {
    key: 'default',
    title: '默认项',
    value: (row: DictionaryEntry) => (row.isDefault ? '是' : '否'),
    width: 80,
  },
  {
    key: 'status',
    title: '状态',
    value: (row: DictionaryEntry) => statusLabel(row.status),
    width: 80,
  },
  {
    detailOnly: true,
    key: 'remark',
    title: '备注',
    value: (row: DictionaryEntry) => row.remark,
  },
];

function typeFields(row?: Dictionary): VbenFormSchema<DictionaryInput>[] {
  return [
    {
      component: 'Input',
      fieldName: 'name',
      label: '字典名称',
      rules: requiredText('字典名称'),
    },
    {
      component: 'Input',
      componentProps: { disabled: !!row },
      fieldName: 'type',
      label: '字典标识',
      rules: requiredText('字典标识')
        .toLowerCase()
        .regex(/^[a-z][a-z0-9_]*$/, '使用字母、数字和下划线'),
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

function entryFields(): VbenFormSchema<DictionaryEntryInput>[] {
  return [
    {
      component: 'Input',
      fieldName: 'label',
      label: '显示名称',
      rules: requiredText('显示名称'),
    },
    {
      component: 'Input',
      fieldName: 'value',
      label: '字典值',
      rules: z.string().min(1, '请输入字典值').max(64),
    },
    {
      component: 'Input',
      componentProps: { maxlength: 64 },
      fieldName: 'tag',
      label: '标签',
    },
    {
      component: 'InputNumber',
      componentProps: { precision: 0 },
      defaultValue: 0,
      fieldName: 'sort',
      label: '排序',
    },
    {
      component: 'Switch',
      defaultValue: false,
      fieldName: 'isDefault',
      label: '默认项',
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

function typeInput(values: DictionaryInput, row?: Dictionary): DictionaryInput {
  return {
    name: values.name.trim(),
    remark: optionalText(values.remark),
    status: values.status,
    type: row?.type ?? values.type.trim().toLowerCase(),
  };
}

function entryInput(
  values: DictionaryEntryInput,
  row?: DictionaryEntry,
): DictionaryEntryInput {
  return {
    dictType: row?.dictType ?? current.value?.type ?? '',
    isDefault: !!values.isDefault,
    label: values.label.trim(),
    remark: optionalText(values.remark),
    sort: values.sort ?? 0,
    status: values.status,
    tag: optionalText(values.tag),
    value: values.value,
  };
}

async function refresh() {
  try {
    await refreshDictionaryCache();
    message.success('字典缓存已刷新');
  } catch (error) {
    showError(error);
  }
}
</script>

<template>
  <Page :title="current ? `${current.name} · 字典项` : '字典管理'">
    <ResourceTable
      v-if="current"
      :key="current.type"
      :columns="entryColumns"
      :fields="entryFields"
      :initial-filters="{ dictType: current.type }"
      :input="entryInput"
      :resource="entries"
      :values="(row) => row ?? {}"
      title="字典项"
    >
      <template #toolbar>
        <Button @click="current = undefined">返回字典类型</Button>
      </template>
    </ResourceTable>
    <ResourceTable
      v-else
      :columns="typeColumns"
      :extra-actions="
        (row) => [
          {
            onClick: () => {
              current = row;
            },
            text: '字典项',
          },
        ]
      "
      :fields="typeFields"
      :initial-filters="{}"
      :input="typeInput"
      :resource="dictionaries"
      :values="(row) => row ?? {}"
      title="字典类型"
    >
      <template #toolbar><Button @click="refresh">刷新缓存</Button></template>
    </ResourceTable>
  </Page>
</template>
