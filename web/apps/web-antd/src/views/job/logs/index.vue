<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';
import type { TaskLog, TaskLogFilters } from '#/api/job/logs';

import { computed } from 'vue';
import { useRoute } from 'vue-router';

import { Page } from '@vben/common-ui';

import { Button } from 'ant-design-vue';

import { z } from '#/adapter/form';
import { cleanTaskLogs, listTaskLogs } from '#/api/job/logs';

import ResourceTable from '../../system/resource-table.vue';
import { confirmAction, resultLabel } from '../../system/resource-view';
import { formatDate } from '../../system/shared';

const route = useRoute();
const taskId = computed(() => typeof route.query.taskId === 'string' && /^[1-9]\d{0,18}$/.test(route.query.taskId) ? route.query.taskId : undefined);
const resource = { list: listTaskLogs };
const searchFields = computed<VbenFormSchema<TaskLogFilters>[]>(() => [
  { component: 'Input', defaultValue: taskId.value, fieldName: 'taskId', label: '任务 ID', rules: z.string().regex(/^(?:[1-9]\d{0,18})?$/, '请输入有效任务 ID').optional() },
  { component: 'Select', componentProps: { allowClear: true, options: [{ label: '成功', value: 1 }, { label: '失败', value: 2 }] }, fieldName: 'status', label: '结果' },
]);
const columns = [
  { key: 'name', title: '任务名称', value: (row: TaskLog) => row.name },
  { key: 'taskId', title: '任务 ID', value: (row: TaskLog) => row.taskId, width: 180 },
  { key: 'status', title: '结果', value: (row: TaskLog) => resultLabel(row.status), width: 80 },
  { key: 'cost', title: '耗时（毫秒）', value: (row: TaskLog) => row.costMs, width: 110 },
  { key: 'started', title: '开始时间', value: (row: TaskLog) => formatDate(row.startedAt), width: 175 },
  { detailOnly: true, key: 'target', title: '执行目标', value: (row: TaskLog) => row.target },
  { detailOnly: true, key: 'group', title: '任务组', value: (row: TaskLog) => row.taskGroup },
  { detailOnly: true, key: 'params', title: '参数摘要', value: (row: TaskLog) => row.params },
  { detailOnly: true, key: 'error', title: '失败原因', value: (row: TaskLog) => row.errorMsg },
];

function clean(reload: () => Promise<void>) {
  confirmAction('确认清空全部任务日志？', async () => {
    await cleanTaskLogs();
    await reload();
  }, '任务日志已清空');
}
</script>

<template>
  <Page title="任务日志">
    <ResourceTable :key="taskId" :columns="columns" :initial-filters="{ taskId }" :resource="resource" :search-fields="searchFields" title="任务日志">
      <template #toolbar="{ reload }"><Button danger @click="clean(reload)">清空日志</Button></template>
    </ResourceTable>
  </Page>
</template>
