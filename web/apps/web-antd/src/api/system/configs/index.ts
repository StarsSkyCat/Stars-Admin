import { request } from '#/api/request';
import { crudResource } from '#/api/resource';

export interface ConfigInput {
  key: string;
  name: string;
  remark: null | string;
  value: string;
}

export interface Config extends ConfigInput {
  id: string;
  isSystem: boolean;
}

export const configs = crudResource<Config, ConfigInput>('/sys/configs');
export const refreshConfigCache = () =>
  request('/sys/configs/cache/refresh', { method: 'POST' });
