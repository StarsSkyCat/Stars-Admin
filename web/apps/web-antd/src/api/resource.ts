import type { Page, PageQuery } from '#/api/system/shared';

import { request } from '#/api/request';

export interface ResourceRow {
  id: string;
}

export function resourcePath(base: string, ids: string | string[]) {
  const suffix = (Array.isArray(ids) ? ids : [ids])
    .map((id) => encodeURIComponent(id))
    .join(',');
  return `${base}/${suffix}`;
}

export function crudResource<Row extends ResourceRow, Input>(base: string) {
  return {
    list: (params: PageQuery, signal?: AbortSignal) =>
      request<Page<Row>>(base, { params, signal }),
    get: (id: string) => request<Row>(resourcePath(base, id)),
    create: (data: Input) => request<Row>(base, { data, method: 'POST' }),
    update: (id: string, data: Input) =>
      request<Row>(resourcePath(base, id), { data, method: 'PUT' }),
    remove: (ids: string[]) =>
      request(resourcePath(base, ids), { method: 'DELETE' }),
  };
}
