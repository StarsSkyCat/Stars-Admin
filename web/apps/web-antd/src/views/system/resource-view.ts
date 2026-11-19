import type { Status } from '#/api/system/shared';

import { downloadFileFromBlob } from '@vben/utils';

import { message, Modal } from 'ant-design-vue';

import { z } from '#/adapter/form';

import { showError } from './shared';

export interface ResourceColumn<Row> {
  detailOnly?: boolean;
  key: string;
  title: string;
  value: (row: Row) => boolean | null | number | string | undefined;
  width?: number;
}

export const requiredText = (label: string, max = 64) =>
  z.string().trim().min(1, `请输入${label}`).max(max);

export const statusLabel = (status: Status) => (status === 1 ? '启用' : '停用');
export const resultLabel = (status: Status) => (status === 1 ? '成功' : '失败');

export function saveBlob(blob: Blob, filename: string) {
  downloadFileFromBlob({ fileName: filename, source: blob });
}

export function confirmAction(
  title: string,
  apply: () => Promise<unknown>,
  success: string,
  content?: string,
) {
  Modal.confirm({
    content,
    async onOk() {
      try {
        await apply();
        message.success(success);
      } catch (error) {
        showError(error);
        throw error;
      }
    },
    title,
  });
}
