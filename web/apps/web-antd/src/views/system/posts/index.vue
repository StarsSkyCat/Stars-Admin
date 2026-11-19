<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';
import type { Post, PostInput } from '#/api/system/posts';

import { Page } from '@vben/common-ui';

import { posts } from '#/api/system/posts';

import ResourceTable from '../resource-table.vue';
import { requiredText, statusLabel } from '../resource-view';
import { formatDate, optionalText, statusOptions } from '../shared';

const columns = [
  { key: 'name', title: '岗位名称', value: (row: Post) => row.name },
  { key: 'code', title: '岗位编码', value: (row: Post) => row.code },
  { key: 'sort', title: '排序', value: (row: Post) => row.sort, width: 80 },
  {
    key: 'status',
    title: '状态',
    value: (row: Post) => statusLabel(row.status),
    width: 80,
  },
  {
    key: 'created',
    title: '创建时间',
    value: (row: Post) => formatDate(row.createdAt),
    width: 180,
  },
  {
    detailOnly: true,
    key: 'remark',
    title: '备注',
    value: (row: Post) => row.remark,
  },
];

function fields(): VbenFormSchema<PostInput>[] {
  return [
    {
      component: 'Input',
      fieldName: 'name',
      label: '岗位名称',
      rules: requiredText('岗位名称'),
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: '岗位编码',
      rules: requiredText('岗位编码')
        .toLowerCase()
        .regex(/^[a-z][a-z0-9_-]*$/, '编码须以字母开头'),
    },
    {
      component: 'InputNumber',
      componentProps: { precision: 0 },
      defaultValue: 0,
      fieldName: 'sort',
      label: '排序',
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

function input(values: PostInput): PostInput {
  return {
    code: values.code.trim().toLowerCase(),
    name: values.name.trim(),
    remark: optionalText(values.remark),
    sort: values.sort ?? 0,
    status: values.status,
  };
}
</script>

<template>
  <Page title="岗位管理">
    <ResourceTable
      :columns="columns"
      :fields="fields"
      :initial-filters="{}"
      :input="input"
      :resource="posts"
      :values="(row) => row ?? {}"
      title="岗位"
    />
  </Page>
</template>
