import type { Page, PageQuery, Status } from '../shared';

import { request } from '#/api/request';

import { systemIdPath } from '../shared';

export interface DepartmentInput {
  leader: null | string;
  name: string;
  parentId: string;
  phone: null | string;
  remark: null | string;
  sort: number;
  status: Status;
}

export interface Department extends DepartmentInput {
  children: Department[];
  createdAt: string;
  id: string;
}

export const listDepartments = (
  params: PageQuery & { name?: string; status?: Status },
  signal?: AbortSignal,
) => request<Page<Department>>('/sys/depts', { params, signal });
export const getDepartmentTree = (signal?: AbortSignal) =>
  request<Department[]>('/sys/depts/tree', { signal });
export const getDepartmentOptions = (signal?: AbortSignal) =>
  request<Department[]>('/sys/depts/options', { signal });
export const getDepartment = (id: string) =>
  request<Department>(systemIdPath('depts', id));
export const createDepartment = (data: DepartmentInput) =>
  request<Department>('/sys/depts', { data, method: 'POST' });
export const updateDepartment = (id: string, data: DepartmentInput) =>
  request<Department>(systemIdPath('depts', id), { data, method: 'PUT' });
export const removeDepartments = (ids: string[]) =>
  request(systemIdPath('depts', ids), { method: 'DELETE' });
