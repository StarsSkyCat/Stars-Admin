interface ErrorBody {
  code?: number;
  msg?: string;
}

export interface HttpResponse<T = unknown> {
  data: T;
  headers: Record<string, string | undefined>;
  status: number;
}

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status = 0,
    readonly code = 0,
    readonly requestId?: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export async function responseError(response: HttpResponse) {
  let body = response.data;
  if (body instanceof Blob && isJson(response)) {
    try {
      body = JSON.parse(await body.text());
    } catch {
      body = null;
    }
  }
  const { code, msg } = (
    body && typeof body === 'object' ? body : {}
  ) as ErrorBody;
  const messages: Record<number, string> = {
    403: '没有操作权限',
    404: '资源不存在或不在可见范围内',
    503: '服务暂不可用，请稍后查询确认',
  };
  const fallback = messages[response.status] ?? '请求失败，请稍后重试';
  return new ApiError(
    msg || fallback,
    response.status,
    code,
    response.headers['x-request-id'],
  );
}

function isJson(response: HttpResponse) {
  return /[/+]json\b/i.test(response.headers['content-type'] ?? '');
}

export async function readDownload(response: HttpResponse<Blob>) {
  if (
    response.status !== 200 ||
    isJson(response) ||
    !(response.data instanceof Blob)
  ) {
    throw await responseError(response);
  }
  return response.data;
}

export function errorText(error: unknown) {
  if (!(error instanceof ApiError)) return '操作失败，请稍后重试';
  return error.requestId
    ? `${error.message}（请求编号 ${error.requestId}）`
    : error.message;
}
