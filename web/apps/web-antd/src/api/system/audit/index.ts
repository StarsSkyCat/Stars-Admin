import type { Page, PageQuery, Status } from '../shared';

import { download, request } from '#/api/request';
import { resourcePath } from '#/api/resource';

export interface AuditFilters {
  status?: Status;
  username?: string;
}

export interface LoginLog {
  id: string;
  ip: string;
  loginAt: string;
  msg: string;
  status: Status;
  username: string;
}

export interface OperationLog {
  action: string;
  costMs: number;
  id: string;
  ip: string;
  method: string;
  module: string;
  operAt: string;
  response: string;
  status: Status;
  traceId: string;
  url: string;
  username: string;
}

export interface OnlineSession {
  browser: string;
  ip: string;
  lastSeenAt: string;
  loginAt: string;
  sid: string;
  userId: string;
  username: string;
}

function auditResource<Row>(kind: 'login' | 'oper') {
  const base = `/sys/audit/${kind}-logs`;
  return {
    list: (params: AuditFilters & PageQuery, signal?: AbortSignal) =>
      request<Page<Row>>(base, { params, signal }),
    clean: () => request(`${base}/clean`, { method: 'DELETE' }),
    export: (params: AuditFilters) =>
      download(`${base}/export`, { method: 'POST', params }),
  };
}

export const loginLogs = auditResource<LoginLog>('login');
export const operationLogs = auditResource<OperationLog>('oper');
export const listOnline = (params: PageQuery, signal?: AbortSignal) =>
  request<Page<OnlineSession>>('/sys/audit/online', { params, signal });
export const forceLogout = (userId: string) =>
  request(resourcePath('/sys/audit/online', userId), { method: 'DELETE' });
