import type { Page, PageQuery, Status } from '../shared';

import { request } from '#/api/request';

import { systemIdPath } from '../shared';

export type MenuType = 1 | 2 | 3;

export interface MenuInput {
  component: null | string;
  icon: null | string;
  name: string;
  parentId: string;
  path: null | string;
  perm: null | string;
  remark: null | string;
  sort: number;
  status: Status;
  type: MenuType;
  visible: Status;
}

export interface Menu extends MenuInput {
  children: Menu[];
  createdAt: string;
  id: string;
  updatedAt: string;
}

export interface MenuFilters {
  name?: string;
  status?: Status;
}

const path = '/sys/menus';
export const listMenus = (
  params: MenuFilters & PageQuery,
  signal?: AbortSignal,
) => request<Page<Menu>>(path, { params, signal });
export const getMenuTree = (signal?: AbortSignal) =>
  request<Menu[]>(`${path}/tree`, { signal });
export const getMenuOptions = (signal?: AbortSignal) =>
  request<Menu[]>(`${path}/options`, { signal });
export const getMenu = (id: string) => request<Menu>(systemIdPath('menus', id));
export const createMenu = (data: MenuInput) =>
  request<Menu>(path, { data, method: 'POST' });
export const updateMenu = (id: string, data: MenuInput) =>
  request<Menu>(systemIdPath('menus', id), { data, method: 'PUT' });
export const removeMenus = (ids: string[]) =>
  request(systemIdPath('menus', ids), { method: 'DELETE' });
