import type { Page, PageQuery } from '#/api/system/shared';

import { download, request } from '#/api/request';
import { resourcePath } from '#/api/resource';

export interface DatabaseTable {
  comment: string;
  name: string;
}

export interface ColumnInput {
  columnName: string;
  dictType: null | string;
  goField: string;
  htmlType: string;
  isEdit: boolean;
  isInsert: boolean;
  isList: boolean;
  isQuery: boolean;
  isRequired: boolean;
  jsonField: string;
  queryType: string;
}

export interface GeneratedColumn extends ColumnInput {
  changeKind: string;
  dbType: string;
  goType: string;
  id: string;
  isNullable: boolean;
  isPk: boolean;
  removed: boolean;
}

export interface ScopeConfig {
  departmentColumn: string;
  mode: 'admin' | 'deny' | 'mapped';
  ownerColumn: string;
}

export interface GenerationInput {
  author: string;
  businessName: string;
  columns: ColumnInput[];
  featureName: string;
  genType: 'local' | 'zip';
  moduleName: string;
  scope: ScopeConfig;
}

export interface GenerationTable extends GenerationInput {
  columns: GeneratedColumn[];
  id: string;
  syncedAt: null | string;
  tableComment: string;
  tableName: string;
}

export interface GeneratedFile {
  content: string;
  path: string;
}

const base = '/gen/tables';
export const listGenerationTables = (params: PageQuery, signal?: AbortSignal) =>
  request<Page<GenerationTable>>(base, { params, signal });
export const listDatabaseTables = () => request<DatabaseTable[]>(`${base}/db`);
export const importTables = (tables: string[]) =>
  request<GenerationTable[]>(`${base}/import`, {
    data: { tables },
    method: 'POST',
  });
export const getGenerationTable = (id: string) =>
  request<GenerationTable>(resourcePath(base, id));
export const saveGenerationTable = (id: string, data: GenerationInput) =>
  request<GenerationTable>(resourcePath(base, id), { data, method: 'PUT' });
export const removeGenerationTables = (ids: string[]) =>
  request(resourcePath(base, ids), { method: 'DELETE' });
export const syncGenerationTable = (id: string) =>
  request<GenerationTable>(`${resourcePath(base, id)}/sync`, {
    method: 'POST',
  });
export const previewGeneration = (id: string) =>
  request<GeneratedFile[]>(`${resourcePath(base, id)}/preview`);
export const downloadGeneration = (id: string) =>
  download(`${resourcePath(base, id)}/generate`, { method: 'POST' });
