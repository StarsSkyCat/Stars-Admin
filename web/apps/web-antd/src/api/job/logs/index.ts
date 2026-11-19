import type { Page, PageQuery, Status } from '#/api/system/shared';

import { request } from '#/api/request';

export interface TaskLog {
  costMs: number;
  errorMsg: string;
  id: string;
  name: string;
  params: string;
  startedAt: string;
  status: Status;
  target: string;
  taskGroup: string;
  taskId: string;
}

export interface TaskLogFilters {
  status?: Status;
  taskId?: string;
}

export const listTaskLogs = (
  params: PageQuery & TaskLogFilters,
  signal?: AbortSignal,
) =>
  request<Page<TaskLog>>('/job/logs', {
    params: { ...params, taskId: params.taskId || undefined },
    signal,
  });
export const cleanTaskLogs = () =>
  request('/job/logs/clean', { method: 'DELETE' });
