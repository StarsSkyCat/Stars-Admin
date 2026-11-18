import type { HttpResponse } from './errors';

import type { Tokens } from '#/api/core/auth';

import { useAppConfig } from '@vben/hooks';
import { RequestClient } from '@vben/request';

import { sessions } from '#/store/session';

import { ApiError, readDownload, responseError } from './errors';

interface Envelope<T> {
  code: number;
  data: T;
  msg: string;
}

interface Options {
  data?: unknown;
  method?: 'DELETE' | 'GET' | 'POST' | 'PUT';
  params?: object;
  personal?: boolean;
  signal?: AbortSignal;
}

const { apiURL } = useAppConfig(import.meta.env, import.meta.env.PROD);
const client = new RequestClient({
  baseURL: apiURL,
  validateStatus: () => true,
  timeout: 15_000,
});

function expired() {
  return new ApiError('会话已失效，请重新登录', 401, 2000);
}

async function send<T>(
  url: string,
  options: Options,
  token?: string,
  download = false,
) {
  const headers = {
    'Content-Type': options.data instanceof FormData ? false : undefined,
    Authorization: token ? `Bearer ${token}` : undefined,
  };
  try {
    return await client.request<HttpResponse<T>>(url, {
      data: options.data,
      headers,
      method: options.method ?? 'GET',
      params: options.params,
      responseType: download ? 'blob' : 'json',
      signal: options.signal,
    });
  } catch {
    if (options.signal?.aborted)
      throw new DOMException('请求已取消', 'AbortError');
    throw new ApiError('无法连接服务，请检查网络后重试');
  }
}

async function json<T>(response: HttpResponse<Envelope<T>>) {
  if (response.status === 200 && response.data?.code === 0)
    return response.data.data;
  throw await responseError(response);
}

export async function publicRequest<T>(
  url: string,
  options: Options = {},
  token?: string,
) {
  return json(await send<Envelope<T>>(url, options, token));
}

export function createProtectedRequest(
  manager: typeof sessions,
  transport: typeof send,
  refresh: (token: string) => Promise<Tokens>,
) {
  return async function protectedRequest<T, Result>(
    url: string,
    options: Options,
    parse: (response: HttpResponse<T>) => Promise<Result>,
    download = false,
  ) {
    let session = manager.read();
    if (!session) throw expired();
    if (session.mustChangePassword && !options.personal) {
      throw new ApiError('请先修改密码', 403, 2008);
    }

    for (let attempt = 0; attempt < 2; attempt++) {
      options.signal?.throwIfAborted();
      const response = await transport<T>(
        url,
        options,
        session.accessToken,
        download,
      );
      if (manager.read()?.id !== session.id) throw expired();
      try {
        const result = await parse(response);
        const current = manager.read();
        if (current?.id !== session.id) throw expired();
        if (current.mustChangePassword && !options.personal) {
          throw new ApiError('请先修改密码', 403, 2008);
        }
        return result;
      } catch (error) {
        if (!(error instanceof ApiError)) throw error;
        if (error.code === 2008) {
          await manager.requirePasswordChange(session.id);
        } else if (error.status === 401) {
          if (error.code === 2001 && attempt === 0) {
            const next = await manager.rotate(session, refresh);
            if (!next) throw expired();
            if (next.mustChangePassword && !options.personal) {
              throw new ApiError('请先修改密码', 403, 2008);
            }
            session = next;
            continue;
          }
          if (
            error.code === 2000 ||
            error.code === 2001 ||
            error.code === 2007
          ) {
            await manager.clear(session.id, session.accessToken);
          }
        }
        throw error;
      }
    }
    throw expired();
  };
}

const authenticated = createProtectedRequest(sessions, send, (refreshToken) =>
  publicRequest('/sys/auth/refresh', {
    data: { refreshToken },
    method: 'POST',
  }),
);

export function request<T>(url: string, options: Options = {}) {
  return authenticated<Envelope<T>, T>(url, options, json);
}

export function download(url: string, options: Options) {
  return authenticated(url, options, readDownload, true);
}
