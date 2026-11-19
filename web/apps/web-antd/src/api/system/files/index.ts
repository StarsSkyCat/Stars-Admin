import { downloadAttachment, request } from '#/api/request';
import { resourcePath } from '#/api/resource';

export interface UploadedFile {
  createdAt: string;
  id: string;
  mime: string;
  originalName: string;
  size: number;
}

const base = '/sys/files';
export function uploadFile(file: File) {
  const data = new FormData();
  data.set('file', file);
  return request<UploadedFile>(`${base}/upload`, { data, method: 'POST' });
}
export async function downloadFile(id: string) {
  const { blob, disposition } = await downloadAttachment(
    `${resourcePath(base, id)}/download`,
  );
  const encoded = disposition?.match(/;\s*filename\*=utf-8''([^;]+)/i)?.[1];
  if (encoded) {
    try {
      return { blob, filename: decodeURIComponent(encoded) };
    } catch {
      // 遇到无效编码时继续使用普通文件名。
    }
  }
  const plain = disposition?.match(
    /;\s*filename=(?:"((?:\\.|[^"])*)"|([^;]+))/i,
  );
  return {
    blob,
    filename: plain?.[1]?.replaceAll(/\\(.)/g, '$1') ?? plain?.[2]?.trim(),
  };
}
export const removeFile = (id: string) =>
  request(resourcePath(base, id), { method: 'DELETE' });
