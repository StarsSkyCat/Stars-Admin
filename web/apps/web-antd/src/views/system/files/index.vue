<script setup lang="ts">
import type { UploadProps } from 'ant-design-vue';

import type { UploadedFile } from '#/api/system/files';

import { ref, shallowRef } from 'vue';

import { Page } from '@vben/common-ui';

import {
  Button,
  Card,
  Input,
  message,
  Space,
  Table,
  Upload,
} from 'ant-design-vue';

import { downloadFile, removeFile, uploadFile } from '#/api/system/files';
import { canAccess } from '#/router/access';
import { useAuthStore } from '#/store';

import { confirmAction, saveBlob } from '../resource-view';
import { formatDate, showError } from '../shared';

const auth = useAuthStore();
const allowed = (action: string) =>
  canAccess(auth.profile, `sys:file:${action}`);
const receipts = shallowRef<UploadedFile[]>([]);
const selection = shallowRef<File>();
const fileId = ref('');
const busy = ref(false);
const columns = [
  { dataIndex: 'originalName', key: 'originalName', title: '文件名' },
  { dataIndex: 'id', key: 'id', title: '文件 ID' },
  { dataIndex: 'size', key: 'size', title: '大小（字节）' },
  { dataIndex: 'createdAt', key: 'createdAt', title: '上传时间' },
  { key: 'actions', title: '操作', width: 160 },
];
const beforeUpload: NonNullable<UploadProps['beforeUpload']> = (file) => {
  selection.value = file;
  return false;
};

async function upload() {
  if (!selection.value || busy.value) return;
  busy.value = true;
  try {
    const result = await uploadFile(selection.value);
    receipts.value = [
      result,
      ...receipts.value.filter(({ id }) => id !== result.id),
    ];
    fileId.value = result.id;
    selection.value = undefined;
    message.success('上传成功');
  } catch (error) {
    showError(error);
  } finally {
    busy.value = false;
  }
}

function validId(value: string) {
  const id = value.trim();
  if (/^[1-9]\d{0,18}$/.test(id)) return id;
  message.warning('请输入有效的文件 ID');
}

async function getFile(value: string, name?: string) {
  const id = validId(value);
  if (!id || busy.value) return;
  busy.value = true;
  try {
    const result = await downloadFile(id);
    saveBlob(result.blob, name ?? result.filename ?? `文件-${id}`);
  } catch (error) {
    showError(error);
  } finally {
    busy.value = false;
  }
}

function remove(value: string) {
  const id = validId(value);
  if (!id || busy.value) return;
  confirmAction(
    `确认删除文件 ${id}？`,
    async () => {
      await removeFile(id);
      receipts.value = receipts.value.filter((receipt) => receipt.id !== id);
      if (fileId.value.trim() === id) fileId.value = '';
    },
    '文件已删除',
  );
}
</script>

<template>
  <Page title="文件管理">
    <Card v-if="allowed('add')" title="上传文件" class="mb-4">
      <Space wrap>
        <Upload
          :before-upload="beforeUpload"
          :file-list="[]"
          :show-upload-list="false"
          :disabled="busy"
        >
          <Button>选择文件</Button>
        </Upload>
        <span v-if="selection" class="break-all">{{ selection.name }}</span>
        <Button
          :disabled="!selection"
          :loading="busy"
          type="primary"
          @click="upload"
        >
          上传
        </Button>
      </Space>
    </Card>
    <Card
      v-if="allowed('get') || allowed('remove')"
      title="按 ID 操作"
      class="mb-4"
    >
      <Space wrap>
        <Input
          v-model:value="fileId"
          :maxlength="19"
          aria-label="文件 ID"
          inputmode="numeric"
          placeholder="输入文件 ID"
        />
        <Button v-if="allowed('get')" :disabled="busy" @click="getFile(fileId)">
          下载
        </Button>
        <Button
          v-if="allowed('remove')"
          :disabled="busy"
          danger
          @click="remove(fileId)"
        >
          删除
        </Button>
      </Space>
    </Card>
    <Card v-if="allowed('add')" title="本次上传">
      <Table
        :columns="columns"
        :data-source="receipts"
        :pagination="false"
        :scroll="{ x: 780 }"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'createdAt'">
            {{ formatDate(record.createdAt) }}
          </template>
          <Space v-else-if="column.key === 'actions'">
            <Button
              v-if="allowed('get')"
              :disabled="busy"
              type="link"
              @click="getFile(record.id, record.originalName)"
            >
              下载
            </Button>
            <Button
              v-if="allowed('remove')"
              :disabled="busy"
              danger
              type="link"
              @click="remove(record.id)"
            >
              删除
            </Button>
          </Space>
        </template>
      </Table>
    </Card>
  </Page>
</template>
