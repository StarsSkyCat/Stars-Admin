import type { NamedOption, Page, PageQuery, Status } from '../shared';

import { request } from '#/api/request';

import { systemIdPath } from '../shared';

export type DataScope = 1 | 2 | 3 | 4 | 5;

export interface RoleInput {
  code: string;
  name: string;
  remark: null | string;
  sort: number;
}

export interface Role extends RoleInput {
  createdAt: string;
  dataScope: DataScope;
  deptIds: string[];
  id: string;
  menuIds: string[];
  status: Status;
}

export interface RoleFilters {
  code?: string;
  name?: string;
  status?: Status;
}

const path = '/sys/roles';
export const listRoles = (
  params: PageQuery & RoleFilters,
  signal?: AbortSignal,
) => request<Page<Role>>(path, { params, signal });
export const getRole = (id: string) => request<Role>(systemIdPath('roles', id));
export const createRole = (data: RoleInput) =>
  request<Role>(path, { data, method: 'POST' });
export const updateRole = (id: string, data: RoleInput) =>
  request<Role>(systemIdPath('roles', id), { data, method: 'PUT' });
export const removeRoles = (ids: string[]) =>
  request(systemIdPath('roles', ids), { method: 'DELETE' });
export const updateRoleStatus = (id: string, status: Status) =>
  request(`${systemIdPath('roles', id)}/status`, {
    data: { status },
    method: 'PUT',
  });
export const updateRoleMenus = (id: string, menuIds: string[]) =>
  request(`${systemIdPath('roles', id)}/menus`, {
    data: { menuIds },
    method: 'PUT',
  });
export const updateRoleScope = (
  id: string,
  dataScope: DataScope,
  deptIds: string[],
) =>
  request(`${systemIdPath('roles', id)}/data-scope`, {
    data: { dataScope, deptIds },
    method: 'PUT',
  });

export async function getRoleOptions() {
  const roles: NamedOption[] = [];
  for (let page = 1; ; page++) {
    const result = await listRoles({ page, pageSize: 100, status: 1 });
    roles.push(...result.list);
    if (result.list.length === 0 || roles.length >= result.total) return roles;
  }
}
