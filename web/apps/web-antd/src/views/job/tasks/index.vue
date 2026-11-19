<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';
import type { Task, TaskFilters, TaskInput } from '#/api/job/tasks';

import { ref } from 'vue';
import { useRouter } from 'vue-router';

import { Page, useVbenModal } from '@vben/common-ui';

import { Button, Input, message } from 'ant-design-vue';

import { z } from '#/adapter/form';
import { changeTaskStatus, previewCron, runTask, tasks } from '#/api/job/tasks';
import { jobPages } from '#/router/routes/modules/system';

import ResourceTable from '../../system/resource-table.vue';
import {
  confirmAction,
  requiredText,
  statusLabel,
} from '../../system/resource-view';
import {
  formatDate,
  optionalText,
  showError,
  statusOptions,
} from '../../system/shared';

const router = useRouter();
const expression = ref('');
const times = ref<string[]>([]);
const calculating = ref(false);
const [CronModal, cronModal] = useVbenModal({ showConfirmButton: false });
const columns = [
  { key: 'name', title: '任务名称', value: (row: Task) => row.name },
  { key: 'group', title: '任务组', value: (row: Task) => row.taskGroup },
  { key: 'cron', title: 'Cron 表达式', value: (row: Task) => row.cron },
  { key: 'target', title: '执行目标', value: (row: Task) => row.target },
  {
    key: 'status',
    title: '状态',
    value: (row: Task) => statusLabel(row.status),
    width: 85,
  },
  {
    detailOnly: true,
    key: 'params',
    title: '参数',
    value: (row: Task) => row.params,
  },
  {
    detailOnly: true,
    key: 'misfire',
    title: '错过触发',
    value: (row: Task) =>
      row.misfirePolicy === 1 ? '启动时补跑一次' : '等待下一次计划',
  },
  {
    detailOnly: true,
    key: 'concurrent',
    title: '允许并发',
    value: (row: Task) => (row.concurrent ? '是' : '否'),
  },
  {
    detailOnly: true,
    key: 'remark',
    title: '备注',
    value: (row: Task) => row.remark,
  },
];
const searchFields: VbenFormSchema<TaskFilters>[] = [
  { component: 'Input', fieldName: 'name', label: '任务名称' },
  {
    component: 'Select',
    componentProps: { allowClear: true, options: [...statusOptions] },
    fieldName: 'status',
    label: '状态',
  },
];

function fields(): VbenFormSchema<TaskInput>[] {
  return [
    {
      component: 'Input',
      fieldName: 'name',
      label: '任务名称',
      rules: requiredText('任务名称'),
    },
    {
      component: 'Input',
      componentProps: { maxlength: 64 },
      defaultValue: 'default',
      fieldName: 'taskGroup',
      label: '任务组',
    },
    {
      component: 'Input',
      componentProps: { placeholder: '0 0 9 * * *' },
      fieldName: 'cron',
      label: 'Cron 表达式',
      rules: requiredText('Cron 表达式'),
    },
    {
      component: 'Input',
      componentProps: { placeholder: '例如 demoTask' },
      fieldName: 'target',
      label: '执行目标',
      rules: requiredText('已注册的执行目标', 255),
    },
    {
      component: 'Textarea',
      componentProps: { maxlength: 4096 },
      defaultValue: '{}',
      fieldName: 'params',
      label: 'JSON 参数',
      rules: z.string().refine((value) => {
        try {
          JSON.parse(value.trim() || '{}');
          return true;
        } catch {
          return false;
        }
      }, '请输入有效的 JSON'),
    },
    {
      component: 'Select',
      componentProps: {
        options: [
          { label: '等待下一次计划', value: 2 },
          { label: '启动时补跑一次', value: 1 },
        ],
      },
      defaultValue: 2,
      fieldName: 'misfirePolicy',
      label: '错过触发',
    },
    {
      component: 'Switch',
      defaultValue: false,
      fieldName: 'concurrent',
      label: '允许并发',
    },
    {
      component: 'Textarea',
      componentProps: { maxlength: 500 },
      fieldName: 'remark',
      label: '备注',
    },
  ];
}

function input(values: TaskInput): TaskInput {
  return {
    concurrent: !!values.concurrent,
    cron: values.cron.trim(),
    misfirePolicy: values.misfirePolicy ?? 2,
    name: values.name.trim(),
    params: values.params?.trim() || '{}',
    remark: optionalText(values.remark),
    target: values.target.trim(),
    taskGroup: values.taskGroup?.trim() || 'default',
  };
}

function openPreview(cron = '') {
  expression.value = cron;
  times.value = [];
  cronModal.open();
  if (cron) void calculate();
}

async function calculate() {
  const expr = expression.value.trim();
  if (!expr || calculating.value) return;
  calculating.value = true;
  times.value = [];
  try {
    const result = await previewCron(expr);
    if (expression.value.trim() === expr) times.value = result;
  } catch (error) {
    showError(error);
  } finally {
    calculating.value = false;
  }
}

function actions(row: Task, reload: () => Promise<void>) {
  const status = row.status === 1 ? 2 : 1;
  const label = status === 1 ? '启用' : '暂停';
  return [
    {
      onClick: () =>
        confirmAction(
          `确认${label}任务「${row.name}」？`,
          async () => {
            await changeTaskStatus(row.id, status);
            await reload();
          },
          `任务已${label}`,
        ),
      text: label,
    },
    {
      onClick: () =>
        confirmAction(
          `确认手动执行「${row.name}」？`,
          async () => {
            const outcome = await runTask(row.id);
            message.info(
              outcome.state === 'queued'
                ? '已投递，请在任务日志中查看执行结果'
                : '任务本次未投递',
            );
          },
          '执行请求已处理',
        ),
      text: '手动执行',
    },
    { onClick: () => openPreview(row.cron), text: '预览时刻' },
    ...(router.hasRoute(jobPages.logs.name)
      ? [
          {
            onClick: () =>
              router.push({
                name: jobPages.logs.name,
                query: { taskId: row.id },
              }),
            text: '执行日志',
          },
        ]
      : []),
  ];
}
</script>

<template>
  <Page title="定时任务">
    <ResourceTable
      :columns="columns"
      :extra-actions="actions"
      :fields="fields"
      :initial-filters="{}"
      :input="input"
      :resource="tasks"
      :search-fields="searchFields"
      :values="(row) => row ?? {}"
      title="任务"
    >
      <template #toolbar>
        <Button @click="openPreview()">Cron 预览</Button>
      </template>
    </ResourceTable>
    <CronModal title="Cron 触发预览">
      <form class="mb-4 flex gap-2" @submit.prevent="calculate">
        <Input
          v-model:value="expression"
          :maxlength="64"
          aria-label="Cron 表达式"
          placeholder="输入 Cron 表达式"
        />
        <Button
          :disabled="!expression.trim()"
          :loading="calculating"
          html-type="submit"
          type="primary"
        >
          预览
        </Button>
      </form>
      <ol v-if="times.length" class="list-decimal space-y-2 pl-6">
        <li v-for="time in times" :key="time">{{ formatDate(time) }}</li>
      </ol>
    </CronModal>
  </Page>
</template>
