import type { Department } from '#/api/system/departments';

import { message } from 'ant-design-vue';
import dayjs from 'dayjs';

import { errorText } from '#/api/errors';

export const statusOptions = [
  { label: '启用', value: 1 },
  { label: '停用', value: 2 },
] as const;

export const dataScopeOptions = [
  { label: '全部数据', value: 1 },
  { label: '自定义部门', value: 2 },
  { label: '本部门', value: 3 },
  { label: '本部门及下级', value: 4 },
  { label: '仅本人', value: 5 },
] as const;

export function departmentOptions(
  tree: Department[],
  excluded?: string,
): {
  children: ReturnType<typeof departmentOptions>;
  label: string;
  value: string;
}[] {
  return tree.flatMap((department) =>
    department.id === excluded
      ? []
      : [
          {
            children: departmentOptions(department.children, excluded),
            label: department.name,
            value: department.id,
          },
        ],
  );
}

export function departmentNames(
  tree: Department[],
  result = new Map<string, string>(),
) {
  for (const department of tree) {
    result.set(department.id, department.name);
    departmentNames(department.children, result);
  }
  return result;
}

export function showError(error: unknown) {
  message.error(errorText(error));
}

export function formatDate(value: null | string) {
  return value ? dayjs(value).format('YYYY-MM-DD HH:mm:ss') : '—';
}

export function optionalText(value?: null | string) {
  return value?.trim() || null;
}
