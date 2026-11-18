import type { Page, PageQuery, Status } from '../shared';

import { download, request } from '#/api/request';

import { systemIdPath } from '../shared';

export type Gender = 0 | 1 | 2;

export interface UserInput {
  avatar: null | string;
  deptId: null | string;
  email: null | string;
  gender: Gender;
  nickname: string;
  phone: null | string;
  remark: null | string;
  username: string;
}

export interface User extends UserInput {
  createdAt: string;
  id: string;
  loginAt: null | string;
  loginIp: null | string;
  postIds: string[];
  pwdUpdatedAt: null | string;
  roleIds: string[];
  status: Status;
  updatedAt: string;
}

export interface UserFilters {
  deptId?: string;
  nickname?: string;
  status?: Status;
  username?: string;
}

export interface ImportResult {
  errors: { message: string; row: number }[];
  imported: number;
}

export const listUsers = (
  params: PageQuery & UserFilters,
  signal?: AbortSignal,
) => request<Page<User>>('/sys/users', { params, signal });

export const getUser = (id: string) => request<User>(systemIdPath('users', id));
export const createUser = (data: UserInput & { password: string }) =>
  request<User>('/sys/users', { data, method: 'POST' });
export const updateUser = (id: string, data: UserInput) =>
  request<User>(systemIdPath('users', id), { data, method: 'PUT' });
export const removeUsers = (ids: string[]) =>
  request(systemIdPath('users', ids), { method: 'DELETE' });
export const updateUserStatus = (id: string, status: Status) =>
  request(`${systemIdPath('users', id)}/status`, {
    data: { status },
    method: 'PUT',
  });
export const resetUserPassword = (id: string, password: string) =>
  request(`${systemIdPath('users', id)}/password`, {
    data: { password },
    method: 'PUT',
  });
export const setUserRoles = (id: string, roleIds: string[]) =>
  request(`${systemIdPath('users', id)}/roles`, {
    data: { roleIds },
    method: 'PUT',
  });
export const setUserPosts = (id: string, postIds: string[]) =>
  request(`${systemIdPath('users', id)}/posts`, {
    data: { postIds },
    method: 'PUT',
  });

export function importUsers(file: File) {
  const data = new FormData();
  data.append('file', file);
  return request<ImportResult>('/sys/users/import', { data, method: 'POST' });
}

export const exportUsers = (filters: UserFilters) =>
  download('/sys/users/export', { method: 'POST', params: filters });
