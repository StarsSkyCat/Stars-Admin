import type { HttpResponse } from './errors';

import type { Tokens } from '#/api/core/auth';
import type { Session } from '#/store/session';

import { describe, expect, it, vi } from 'vitest';

import { createSessionManager } from '#/store/session';

import { ApiError, readDownload, responseError } from './errors';
import { createProtectedRequest } from './request';

const original: Session = {
  accessToken: '旧访问令牌',
  expiresIn: 1800,
  id: '同一会话',
  mustChangePassword: false,
  refreshToken: '旧刷新令牌',
  tokenType: 'Bearer',
};
const replacement: Tokens = {
  ...original,
  accessToken: '新访问令牌',
  refreshToken: '新刷新令牌',
};
const response = (status: number, code: number): HttpResponse => ({
  data: { code, data: ['9223372036854775806'], msg: '服务端响应' },
  headers: { 'content-type': 'application/json' },
  status,
});
async function parse(reply: HttpResponse) {
  if (reply.status !== 200) throw await responseError(reply);
  return reply.data;
}

function fixture() {
  const storage = new Map<string, string>();
  const wrapped = {
    getItem: (key: string) => storage.get(key) ?? null,
    removeItem: (key: string) => {
      storage.delete(key);
    },
    setItem: (key: string, value: string) => {
      storage.set(key, value);
    },
  };
  const queue = new Map<string, Promise<unknown>>();
  async function lock<T>(name: string, operation: () => Promise<T> | T) {
    const next = (queue.get(name) ?? Promise.resolve()).then(operation);
    queue.set(
      name,
      next.catch(() => {}),
    );
    return next;
  }
  return {
    first: createSessionManager(wrapped, '测试会话', lock),
    second: createSessionManager(wrapped, '测试会话', lock),
  };
}

function transport(dispatch: (token?: string) => HttpResponse) {
  return async <T>(_url: string, _options: object, token?: string) =>
    dispatch(token) as HttpResponse<T>;
}

describe('业务请求和会话轮换', () => {
  it('两个标签页同时遇到 Access 过期，只消费一次 Refresh，分别重放一次', async () => {
    const { first, second } = fixture();
    await first.start(original);
    const refresh = vi.fn(async () => replacement);
    const send = vi.fn((token) =>
      token === original.accessToken ? response(401, 2001) : response(200, 0),
    );
    const browsers = [first, second].map((manager) =>
      createProtectedRequest(manager, transport(send), refresh),
    );
    const replies = await Promise.all(
      browsers.map((request) => request('/sys/users', {}, parse)),
    );
    expect(replies).toEqual([response(200, 0).data, response(200, 0).data]);
    expect(refresh).toHaveBeenCalledExactlyOnceWith(original.refreshToken);
    expect(send.mock.calls).toEqual([
      [original.accessToken],
      [original.accessToken],
      [replacement.accessToken],
      [replacement.accessToken],
    ]);
    expect(second.read()?.refreshToken).toBe(replacement.refreshToken);
  });

  it('登出后丢弃另一个标签页迟到的刷新结果', async () => {
    const { first, second } = fixture();
    await first.start(original);
    const deferred = Promise.withResolvers<Tokens>();
    const refresh = vi.fn(() => deferred.promise);
    const rotate = first.rotate(original, refresh);
    await vi.waitFor(() => expect(refresh).toHaveBeenCalledTimes(1));
    await second.clear(original.id);
    deferred.resolve(replacement);
    await expect(rotate).resolves.toBeNull();
    expect(first.read()).toBeNull();
  });

  it('迟到的刷新保留已切换的新账号会话', async () => {
    const { first, second } = fixture();
    await first.start(original);
    const deferred = Promise.withResolvers<Tokens>();
    const refresh = vi.fn(() => deferred.promise);
    const rotate = first.rotate(original, refresh);
    await vi.waitFor(() => expect(refresh).toHaveBeenCalledTimes(1));
    await second.start({ ...original, id: '另一个会话' });
    deferred.resolve(replacement);
    await expect(rotate).resolves.toBeNull();
    expect(first.read()).toMatchObject({
      id: '另一个会话',
      refreshToken: original.refreshToken,
    });
  });

  it('普通 403 不触发刷新，重放仍过期时清理会话', async () => {
    const { first } = fixture();
    const refresh = vi.fn(async () => replacement);
    await first.start(original);
    const denied = createProtectedRequest(
      first,
      transport(() => response(403, 2004)),
      refresh,
    );
    await expect(denied('/sys/users', {}, parse)).rejects.toMatchObject({
      code: 2004,
    });
    expect(refresh).not.toHaveBeenCalled();
    const send = vi.fn(() => response(401, 2001));
    const expired = createProtectedRequest(first, transport(send), refresh);
    await expect(expired('/sys/users', {}, parse)).rejects.toMatchObject({
      code: 2001,
    });
    expect(send).toHaveBeenCalledTimes(2);
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(first.read()).toBeNull();
  });

  it('首登状态拦截业务请求并允许本人请求', async () => {
    const { first } = fixture();
    await first.start({ ...original, mustChangePassword: true });
    const send = vi.fn(() => response(200, 0));
    const request = createProtectedRequest(first, transport(send), vi.fn());
    await expect(request('/sys/users', {}, parse)).rejects.toMatchObject({
      code: 2008,
    });
    expect(send).not.toHaveBeenCalled();
    await expect(
      request('/sys/auth/profile', { personal: true }, parse),
    ).resolves.toEqual(response(200, 0).data);
  });
});

describe('下载响应', () => {
  it('从 Blob 中读取 JSON 错误码，保留请求编号', async () => {
    const data = new Blob([
      JSON.stringify({ code: 2001, msg: '访问令牌过期' }),
    ]);
    await expect(
      readDownload({
        data,
        headers: {
          'content-type': 'application/json; charset=utf-8',
          'x-request-id': 'web-check',
        },
        status: 401,
      }),
    ).rejects.toMatchObject({
      code: 2001,
      requestId: 'web-check',
      status: 401,
    });
  });

  it('保留成功的文件流并拒绝返回 JSON 的下载', async () => {
    const data = new Blob(['文件内容']);
    await expect(
      readDownload({
        data,
        headers: { 'content-type': 'application/octet-stream' },
        status: 200,
      }),
    ).resolves.toBe(data);
    await expect(
      readDownload({
        data,
        headers: { 'content-type': 'application/json' },
        status: 200,
      }),
    ).rejects.toBeInstanceOf(ApiError);
  });
});
