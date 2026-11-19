<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';
import type {
  AuditFilters,
  LoginLog,
  OnlineSession,
  OperationLog,
} from '#/api/system/audit';
import type { PageQuery } from '#/api/system/shared';

import { ref } from 'vue';

import { Page } from '@vben/common-ui';

import { Button, TabPane, Tabs } from 'ant-design-vue';

import {
  forceLogout,
  listOnline,
  loginLogs,
  operationLogs,
} from '#/api/system/audit';

import ResourceTable from '../resource-table.vue';
import { confirmAction, resultLabel, saveBlob } from '../resource-view';
import { formatDate, showError } from '../shared';

const active = ref('oper');
const building = ref(false);
const searchFields: VbenFormSchema<AuditFilters>[] = [
  { component: 'Input', fieldName: 'username', label: '用户名' },
  {
    component: 'Select',
    componentProps: {
      allowClear: true,
      options: [
        { label: '成功', value: 1 },
        { label: '失败', value: 2 },
      ],
    },
    fieldName: 'status',
    label: '结果',
  },
];
const operColumns = [
  {
    key: 'username',
    title: '操作人',
    value: (row: OperationLog) => row.username,
  },
  { key: 'action', title: '操作', value: (row: OperationLog) => row.action },
  {
    key: 'path',
    title: '请求',
    value: (row: OperationLog) => `${row.method} ${row.url}`,
    width: 250,
  },
  {
    key: 'status',
    title: '结果',
    value: (row: OperationLog) => resultLabel(row.status),
    width: 80,
  },
  {
    key: 'time',
    title: '操作时间',
    value: (row: OperationLog) => formatDate(row.operAt),
    width: 175,
  },
  {
    detailOnly: true,
    key: 'module',
    title: '模块',
    value: (row: OperationLog) => row.module,
  },
  {
    detailOnly: true,
    key: 'ip',
    title: 'IP 地址',
    value: (row: OperationLog) => row.ip,
  },
  {
    detailOnly: true,
    key: 'trace',
    title: '追踪 ID',
    value: (row: OperationLog) => row.traceId,
  },
  {
    detailOnly: true,
    key: 'cost',
    title: '耗时（毫秒）',
    value: (row: OperationLog) => row.costMs,
  },
  {
    detailOnly: true,
    key: 'response',
    title: '响应摘要',
    value: (row: OperationLog) => row.response,
  },
];
const loginColumns = [
  { key: 'username', title: '用户名', value: (row: LoginLog) => row.username },
  { key: 'ip', title: 'IP 地址', value: (row: LoginLog) => row.ip },
  {
    key: 'status',
    title: '结果',
    value: (row: LoginLog) => resultLabel(row.status),
    width: 80,
  },
  { key: 'msg', title: '消息', value: (row: LoginLog) => row.msg, width: 215 },
  {
    key: 'time',
    title: '登录时间',
    value: (row: LoginLog) => formatDate(row.loginAt),
    width: 175,
  },
];
const onlineColumns = [
  { key: 'user', title: '用户名', value: (row: OnlineSession) => row.username },
  {
    key: 'userId',
    title: '用户 ID',
    value: (row: OnlineSession) => row.userId,
  },
  { key: 'ip', title: 'IP 地址', value: (row: OnlineSession) => row.ip },
  {
    key: 'seen',
    title: '最近访问',
    value: (row: OnlineSession) => formatDate(row.lastSeenAt),
    width: 175,
  },
  {
    detailOnly: true,
    key: 'login',
    title: '登录时间',
    value: (row: OnlineSession) => formatDate(row.loginAt),
  },
  {
    detailOnly: true,
    key: 'browser',
    title: '客户端',
    value: (row: OnlineSession) => row.browser,
  },
];
const online = {
  async list(params: PageQuery, signal?: AbortSignal) {
    const result = await listOnline(params, signal);
    return {
      ...result,
      list: result.list.map((session) => ({ ...session, id: session.sid })),
    };
  },
};

function clean(kind: 'login' | 'oper', reload: () => Promise<void>) {
  const label = kind === 'oper' ? '操作日志' : '登录日志';
  confirmAction(
    `确认清空全部${label}？`,
    async () => {
      await (kind === 'oper' ? operationLogs.clean() : loginLogs.clean());
      await reload();
    },
    `${label}已清空`,
  );
}

async function exportLogs(kind: 'login' | 'oper', filters: AuditFilters) {
  if (building.value) return;
  building.value = true;
  try {
    const blob = await (kind === 'oper' ? operationLogs : loginLogs).export(
      filters,
    );
    saveBlob(blob, kind === 'oper' ? '操作日志.xlsx' : '登录日志.xlsx');
  } catch (error) {
    showError(error);
  } finally {
    building.value = false;
  }
}

function endSessions(row: OnlineSession, reload: () => Promise<void>) {
  confirmAction(
    `确认将「${row.username}」的全部会话下线？`,
    async () => {
      await forceLogout(row.userId);
      await reload();
    },
    '会话已撤销',
  );
}
</script>

<template>
  <Page title="日志审计">
    <Tabs v-model:active-key="active" destroy-inactive-tab-pane>
      <TabPane key="oper" tab="操作日志">
        <ResourceTable
          :columns="operColumns"
          :initial-filters="{}"
          :resource="operationLogs"
          :search-fields="searchFields"
          title="操作日志"
        >
          <template #toolbar="{ filters, reload }">
            <Button :loading="building" @click="exportLogs('oper', filters)">
              导出
            </Button>
            <Button danger @click="clean('oper', reload)">清空日志</Button>
          </template>
        </ResourceTable>
      </TabPane>
      <TabPane key="login" tab="登录日志">
        <ResourceTable
          :columns="loginColumns"
          :initial-filters="{}"
          :resource="loginLogs"
          :search-fields="searchFields"
          title="登录日志"
        >
          <template #toolbar="{ filters, reload }">
            <Button :loading="building" @click="exportLogs('login', filters)">
              导出
            </Button>
            <Button danger @click="clean('login', reload)">清空日志</Button>
          </template>
        </ResourceTable>
      </TabPane>
      <TabPane key="online" tab="在线会话">
        <ResourceTable
          :columns="onlineColumns"
          :extra-actions="
            (row, reload) => [
              {
                danger: true,
                onClick: () => endSessions(row, reload),
                text: '强制下线',
              },
            ]
          "
          :initial-filters="{}"
          :resource="online"
          title="会话"
        />
      </TabPane>
    </Tabs>
  </Page>
</template>
