export interface Page<T> {
  list: T[];
  page: number;
  pageSize: number;
  total: number;
}

export interface PageQuery {
  page?: number;
  pageSize?: number;
}

export type Status = 1 | 2;

export interface NamedOption {
  id: string;
  name: string;
}

export function systemIdPath(resource: string, ids: string | string[]) {
  const idsList = Array.isArray(ids) ? ids : [ids];
  return `/sys/${resource}/${idsList
    .map((id) => encodeURIComponent(id))
    .join(',')}`;
}
