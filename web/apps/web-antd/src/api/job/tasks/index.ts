import type { Status } from '#/api/system/shared';

import { request } from '#/api/request';
import { crudResource, resourcePath } from '#/api/resource';

export interface TaskInput {
  concurrent: boolean;
  cron: string;
  misfirePolicy: 0 | 1 | 2;
  name: string;
  params: string;
  remark: null | string;
  target: string;
  taskGroup: string;
}

export interface Task extends TaskInput {
  id: string;
  status: Status;
}

export interface TaskFilters {
  name?: string;
  status?: Status;
}

const base = '/job/tasks';
export const tasks = crudResource<Task, TaskInput>(base);
export const runTask = (id: string) =>
  request<{ state: 'queued' | 'skipped' }>(`${resourcePath(base, id)}/run`, {
    method: 'POST',
  });
export const changeTaskStatus = (id: string, status: Status) =>
  request(`${resourcePath(base, id)}/status`, {
    data: { status },
    method: 'PUT',
  });
export const previewCron = (expr: string) =>
  request<string[]>(`${base}/cron/next`, { params: { expr } });
