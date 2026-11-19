import type { Status } from '../shared';

import { request } from '#/api/request';
import { crudResource } from '#/api/resource';

export interface DictionaryInput {
  name: string;
  remark: null | string;
  status: Status;
  type: string;
}

export interface Dictionary extends DictionaryInput {
  id: string;
}

export interface DictionaryEntryInput {
  dictType: string;
  isDefault: boolean;
  label: string;
  remark: null | string;
  sort: number;
  status: Status;
  tag: null | string;
  value: string;
}

export interface DictionaryEntry extends DictionaryEntryInput {
  id: string;
}

export const dictionaries = crudResource<Dictionary, DictionaryInput>(
  '/sys/dict/types',
);
export const entries = crudResource<DictionaryEntry, DictionaryEntryInput>(
  '/sys/dict/data',
);
export const refreshDictionaryCache = () =>
  request('/sys/dict/cache/refresh', { method: 'POST' });
